package mapsaver

import (
	"hput"
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
	Type  input
	val   string
	bytes []byte
}

var texts = make(map[string]runnable)

// Logger logs out.
type Logger interface {
	Debugf(msg string, args ...interface{})
}

type MapSaver struct {
	Logger Logger
}

func (m *MapSaver) SaveText(s string, p url.URL, r *hput.PutResult) error {
	m.Logger.Debugf("processing SaveText with string: %s and path: %s", s, p)
	_, ok := texts[p.Path]
	if ok {
		m.Logger.Debugf("Found something where saving text")
		r.Overwrote = true
	}
	texts[p.Path] = runnable{Type: text, val: s}
	return nil
}

func (m *MapSaver) SaveCode(s string, p url.URL, r *hput.PutResult) error {
	m.Logger.Debugf("processing SaveCode with string: %s and path: %s", s, p.String())
	_, ok := texts[p.Path]
	if ok {
		m.Logger.Debugf("Found something where saving code")
		r.Overwrote = true
	}
	texts[p.Path] = runnable{Type: js, val: s}
	return nil
}

func (m *MapSaver) SaveBinary(b []byte, p url.URL, r *hput.PutResult) error {
	m.Logger.Debugf("processing SaveBinary with length %d and path: %s", len(b), p.String())
	_, ok := texts[p.Path]
	if ok {
		m.Logger.Debugf("Found something where saving binary")
		r.Overwrote = true
	}
	texts[p.Path] = runnable{Type: binary, bytes: b}
	return nil
}

func (m *MapSaver) GetRunnable(p url.URL) (hput.Runnable, error) {
	m.Logger.Debugf("retrieving text at path %s", p.Path)
	r, ok := texts[p.Path]
	if !ok {
		return hput.Runnable{}, nil
	}
	return hput.Runnable{
		Type:   hput.Input(r.Type),
		Text:   r.val,
		Binary: r.bytes,
	}, nil
}

func (m *MapSaver) SendRunnables(p string, runnables chan<- hput.Runnable, done chan<- bool) error {
	for key, runnable := range texts {
		if strings.HasPrefix(key, p) {
			m.Logger.Debugf("Printing key: %s, prefix: %s", key, p)
			r := hput.Runnable{
				Path: key,
				Type: hput.Input(runnable.Type),
				Text: runnable.val,
			}
			runnables <- r
		} else {
			m.Logger.Debugf("Not printing key: %s, prefix: %s", key, p)
		}
	}
	done <- true
	return nil
}
