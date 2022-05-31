package service

import (
	"bytes"
	"fmt"
	"hput"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestSaver struct {
	GiveRunnable hput.Runnable
}

func (t *TestSaver) SaveText(s string, p url.URL, r *hput.PutResult) error {
	r.Input = hput.Text
	r.Message = fmt.Sprintf("Saved Text %s at %s", s, p.Path)
	return nil
}

func (t *TestSaver) GetRunnable(p url.URL) (hput.Runnable, error) {
	return t.GiveRunnable, nil
}

func (t *TestSaver) SendRunnables(p string, runnables chan<- hput.Runnable, done chan<- bool) error {
	runnables <- hput.Runnable{
		Type: hput.Text,
		Text: "aText",
		Path: "/pth",
	}
	done <- true
	return nil
}

func (t *TestSaver) SaveCode(s string, p url.URL, r *hput.PutResult) error {
	r.Input = hput.Js
	r.Message = fmt.Sprintf("Saved Js %s at %s", s, p.Path)
	return nil
}

func (t *TestSaver) SaveBinary(b []byte, p url.URL, r *hput.PutResult) error {
	r.Input = hput.Binary
	r.Message = fmt.Sprintf("Saved Binary at %s", p.Path)
	return nil
}

type TestInterpreter struct {
	ReturnIsCode bool
	R            *http.Request
}

func (t *TestInterpreter) IsCode(s string) (bool, string) {
	return t.ReturnIsCode, "Preset"
}

func (t *TestInterpreter) Run(c string, r *http.Request, w http.ResponseWriter) error {
	w.Write([]byte(fmt.Sprintf("Interpreter Ran %s", c)))
	return nil
}

type TestLogger struct{}

func (t *TestLogger) Debugf(msg string, args ...interface{}) {}

func (t *TestLogger) Debug(msg string) {}

func (t *TestLogger) Warnf(msg string, args ...interface{}) {}

func (t *TestLogger) Errorf(msg string, args ...interface{}) {}

// TestPut tests that the service can accept PUT requests
func TestPut(t *testing.T) {
	tt := []struct {
		name   string
		req    *http.Request
		isCode bool
		res    *hput.PutResult
	}{
		{
			name: "Put Text",
			req: &http.Request{
				Method: http.MethodPut,
				URL:    &url.URL{Path: "/pth"},
				Body:   io.NopCloser(bytes.NewBufferString("aText")),
			},
			res: &hput.PutResult{
				Input:   hput.Text,
				Message: "Saved Text aText at /pth",
			},
		},
		{
			name: "Put Code",
			req: &http.Request{
				Method: http.MethodPut,
				URL:    &url.URL{Path: "/pth"},
				Body:   io.NopCloser(bytes.NewBufferString("return 1;")),
			},
			isCode: true,
			res: &hput.PutResult{
				Input:   hput.Js,
				Message: "Saved Js return 1; at /pth",
			},
		},
		{
			name: "Put Binary",
			req: &http.Request{
				Method: http.MethodPut,
				URL:    &url.URL{Path: "/pth"},
				Body:   io.NopCloser(bytes.NewBuffer([]byte{200, 200, 200, 0, 1})),
			},
			isCode: true,
			res: &hput.PutResult{
				Input:   hput.Binary,
				Message: "Saved Binary at /pth",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			i := &TestInterpreter{ReturnIsCode: test.isCode}
			s := Service{
				Saver:       &TestSaver{},
				Interpreter: i,
				Logger:      &TestLogger{},
			}
			r, err := s.Put(test.req)
			assert.NoError(t, err)
			assert.Equal(t, test.res, r)
		})
	}
}

// TestRun tests that the service can accept requests to run paths
func TestRun(t *testing.T) {
	tt := []struct {
		name     string
		req      *http.Request
		runnable hput.Runnable
		dumpText string
	}{
		{
			name: "Get Text",
			req: &http.Request{
				URL: &url.URL{Path: "/pth"},
			},
			runnable: hput.Runnable{
				Path: "/pth",
				Type: hput.Text,
				Text: "aText",
			},
		},
		{
			name: "Get Binary",
			req: &http.Request{
				URL: &url.URL{Path: "/pth"},
			},
			runnable: hput.Runnable{
				Path:   "/pth",
				Type:   hput.Binary,
				Binary: []byte{200, 200, 200, 0, 1},
			},
		},
		{
			name: "Run Code",
			req: &http.Request{
				URL: &url.URL{Path: "/pth"},
			},
			runnable: hput.Runnable{
				Path: "/pth",
				Type: hput.Js,
				Text: "var a = 1;",
			},
		},
		{
			name: "Get Dump",
			req: &http.Request{
				URL: &url.URL{Path: "/dump"},
			},
			dumpText: "//Dumping creation instructions v0.1\nvar xhr = new XMLHttpRequest();\nxhr.withCredentials = true;\nxhr.open(\"PUT\", \"http://localhost/pth\");\nxhr.send(`aText`);\n",
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			s := Service{
				Saver: &TestSaver{
					GiveRunnable: test.runnable,
				},
				Interpreter: &TestInterpreter{},
				Logger:      &TestLogger{},
			}
			responseRecorder := httptest.NewRecorder()
			err := s.Run(responseRecorder, test.req)
			assert.NoError(t, err)
			assert.Equal(t, responseRecorder.Code, http.StatusOK)
			switch test.runnable.Type {
			case hput.Text:
				assert.Equal(t, test.runnable.Text, responseRecorder.Body.String())
			case hput.Js:
				assert.Equal(t, fmt.Sprintf("Interpreter Ran %s", test.runnable.Text), responseRecorder.Body.String())
			case hput.Binary:
				assert.Equal(t, test.runnable.Binary, responseRecorder.Body.Bytes())
			default:
				assert.Equal(t, test.dumpText, responseRecorder.Body.String())
			}
		})
	}
}