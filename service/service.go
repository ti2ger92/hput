package service

import (
	"context"
	"errors"
	"fmt"
	"hput"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

type input string

const (
	text   input = "Text"
	js           = "Javascript"
	binary       = "Binary"
)

type runnable struct {
	Type input
	Val  []byte
}

// Saver Saves stateful data for the service
type Saver interface {
	SaveText(ctx context.Context, s string, p url.URL, r *hput.PutResult) error
	SaveCode(ctx context.Context, s string, p url.URL, r *hput.PutResult) error
	SaveBinary(ctx context.Context, b []byte, p url.URL, r *hput.PutResult) error
	GetRunnable(ctx context.Context, p url.URL) (hput.Runnable, error)
	SendRunnables(ctx context.Context, p string, runnables chan<- hput.Runnable, done chan<- bool) error
}

// Interpreter understands code
type Interpreter interface {
	IsCode(s string) (bool, string)
	Run(c string, r *http.Request, w http.ResponseWriter) error
}

// Logger logs out.
type Logger interface {
	Debug(msg string)
	Debugf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
}

type Service struct {
	Saver       Saver
	Interpreter Interpreter
	Logger      Logger
}

var (
	ErrPutToDump = errors.New("attempted to add something to /dump which is not allowed")
	ErrPutToLogs = errors.New("attempted to add something to /logs which is not allowed")
)

const (
	// invalidRune is a symbol which, if found, suggests this string is a binary
	invalidRune = rune('�')
)

// Put accepts a Put request and saves it
func (s *Service) Put(ctx context.Context, r *http.Request) (*hput.PutResult, error) {
	s.Logger.Debug("processing PUT service")
	b, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		s.Logger.Errorf("processing read payload error in put request with err: %v", err)
		return nil, hput.ErrCannotReadPostPayload
	}
	// Cannot put at "/dump"
	if strings.ToLower(lastN(r.URL.Path, 5)) == "/dump" {
		return nil, ErrPutToDump
	}
	// Cannot put at "/logs"
	if strings.ToLower(lastN(r.URL.Path, 5)) == "/logs" {
		return nil, ErrPutToLogs
	}
	// Test whether input is a string by checking the first 200 characters for an invalid rune: �
	shortStr := string(b[:int(math.Min(200, float64(len(b))))])
	if strings.ContainsRune(shortStr, invalidRune) {
		s.Logger.Debug("got bytes that don't look like a string")
		res := &hput.PutResult{
			Input:   hput.Binary,
			Message: "I think this is a binary file, saving it as such",
		}
		err := s.Saver.SaveBinary(ctx, b, *r.URL, res)
		return res, err
	}

	str := string(b)
	isCode, msg := s.Interpreter.IsCode(str)
	if !isCode {
		s.Logger.Debugf("processing PUT text service with text: %s to path: %s", str, r.URL.Path)
		res := &hput.PutResult{
			Input:   hput.Text,
			Message: msg,
		}
		err := s.Saver.SaveText(ctx, str, *r.URL, res)
		return res, err
	}
	s.Logger.Debugf("processing PUT code service with text: %s to path: %s", str, r.URL.Path)
	res := &hput.PutResult{
		Input: hput.Js,
	}
	err = s.Saver.SaveCode(ctx, str, *r.URL, res)
	return res, err
}

// Run executes and whatever is at this path on the server. If text was saved that text is returned.
// Code can write out to the http.ResponseWriter, and also return something to output.
func (s *Service) Run(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if strings.ToLower(lastN(r.URL.Path, 5)) == "/dump" {
		s.dumpPath(ctx, *r.URL, w)
		return nil
	}
	s.Logger.Debugf("processing RUN service with path, %s", r.URL.Path)
	runnable, err := s.getPathRunnable(ctx, *r.URL)
	if err != nil {
		s.Logger.Warnf("processing RUN service got an error, %+v", err)
		return fmt.Errorf("Unexpected error running service at path: %s ,:%v", r.URL.Path, err)
	}
	if runnable == nil || string(runnable.Val) == "" {
		s.Logger.Debug("processing RUN service got nil runnable")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("There is nothing at path: '%s', you can use a PUT verb to add something\n", r.URL.Path)))
		return nil
	}
	switch runnable.Type {
	case binary:
		s.Logger.Debugf("processing RUN service got binary length %d", len(runnable.Val))
		w.WriteHeader(http.StatusOK)
		w.Write(runnable.Val)
		return nil
	case text:
		s.Logger.Debugf("processing RUN service got text, %s", runnable.Val)
		w.WriteHeader(http.StatusOK)
		w.Write(runnable.Val)
		return nil
	case js:
		s.Logger.Debugf("processing RUN service got javascript, %s", runnable.Val)
		err := s.Interpreter.Run(string(runnable.Val), r, w)
		if err != nil {
			s.Logger.Debugf("got an error running JS: %+v", err)
			return err
		}
	}
	return nil
}

func (s *Service) dumpPath(ctx context.Context, p url.URL, w http.ResponseWriter) {
	runnablesChan := make(chan hput.Runnable)
	doneChan := make(chan bool, 1)
	pStr := p.Path[:len(p.Path)-5]
	var err error
	go func() {
		s.Logger.Debugf("sending runnables for %s", pStr)
		err = s.Saver.SendRunnables(ctx, pStr, runnablesChan, doneChan)
	}()
	if err != nil {
		s.Logger.Errorf("got an error dumping from path %+v: %+v", p, err)
		return
	}
	w.Write([]byte("//Dumping creation instructions v0.2\n"))
	var dumpedFirst bool
	for {
		select {
		case run := <-runnablesChan:
			s.Logger.Debugf("Got runnable %+v", run)
			switch run.Type {
			case hput.Text, hput.Js:
				if !dumpedFirst {
					dumpedFirst = true
					_, err := w.Write([]byte("var xhr = new XMLHttpRequest();\n"))
					if err != nil {
						s.Logger.Errorf("Error writing binary text out: %w", err)
					}
				} else {
					w.Write([]byte("xhr = new XMLHttpRequest();\n"))
				}
				w.Write([]byte("xhr.withCredentials = true;\n"))
				w.Write([]byte(fmt.Sprintf(`xhr.open("PUT", "http://localhost%s");
`, run.Path)))
				w.Write([]byte(fmt.Sprintf("xhr.send(`%s`);\n", run.Text)))
			case hput.Binary:
				_, err := w.Write([]byte(fmt.Sprintf("// binary at http://localhost%s\n", run.Path)))
				if err != nil {
					s.Logger.Errorf("Error writing binary comment out: %w", err)
				}
			}
		case <-doneChan:
			return
		}
	}
}

// getPathRunnable retrieves the runnable at a path, if it exists. May return nil
func (s *Service) getPathRunnable(ctx context.Context, p url.URL) (*runnable, error) {
	s.Logger.Debugf("processing getPathRunnable with path, %#v", p)
	val, err := s.Saver.GetRunnable(ctx, p)
	if err != nil {
		s.Logger.Errorf("service.getPathRunnable(): unexpected error retrieving a runnable: %+v", err)
		return nil, fmt.Errorf("could not retrieve runnable: %w", err)
	}
	if val.Type == "" {
		s.Logger.Debug("processing getPathRunnable did not find the path")
		return nil, nil
	}
	s.Logger.Debugf("processing getPathRunnable found the path as: %#v", val)
	if input(val.Type) == binary {
		return &runnable{
			Val:  val.Binary,
			Type: input(val.Type),
		}, nil
	}
	return &runnable{
		Val:  []byte(val.Text),
		Type: input(val.Type),
	}, nil
}

// lastN get the last n values from a string
func lastN(s string, n int) string {
	idx := len(s) - n
	if idx < 0 {
		return s
	}
	return s[idx:]
}
