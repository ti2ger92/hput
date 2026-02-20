package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"hput"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestLogger struct{}

func (t *TestLogger) Debugf(msg string, args ...interface{}) {}

func (t *TestLogger) Debug(msg string) {}

func (t *TestLogger) Warnf(msg string, args ...interface{}) {}

func (t *TestLogger) Errorf(msg string, args ...interface{}) {}

func (t *TestLogger) Infof(msg string, args ...interface{}) {}

type TestService struct{}

func (t *TestService) Put(ctx context.Context, w http.ResponseWriter, r *http.Request) (*hput.PutResult, error) {
	p, err := io.ReadAll(r.Body)
	if err != nil {
		panic(fmt.Sprintf("error reading incoming payload: %v", err))
	}
	return &hput.PutResult{
		Input:   hput.Text,
		Message: fmt.Sprintf("passed request with path %s and payload %s to Put", r.URL.Path, string(p)),
	}, nil
}

func (t *TestService) Run(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(fmt.Sprintf("passed request with path %s to Run", r.URL.Path)))
	return nil
}

// Test_Serve verify the server will start and respond to requests
func Test_Serve(t *testing.T) {
	h := Httpserver{
		Port:    8080,
		Logger:  &TestLogger{},
		Service: &TestService{},
	}
	go func() {
		h.Serve()
	}()
	for i := 0; i < 3; i++ {
		time.Sleep(250 * time.Microsecond)
		c := http.Client{}
		r, err := c.Get("http://localhost:8080/ping")
		if r == nil {
			fmt.Print("Warning: server is launching slowly")
		}
		assert.NoError(t, err)
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "passed request with path /ping to Run", string(b))
		return
	}
	panic("Server never started")
}

// Test_handle verifies service can handle basic input types
func Test_handle(t *testing.T) {
	h := Httpserver{
		Port:     8080,
		Logger:   &TestLogger{},
		Service:  &TestService{},
		NonLocal: true,
	}
	tt := []struct {
		name       string
		method     string
		path       string
		reqPayload io.Reader
		resPayload []byte
		statusCode int
		resHeader  http.Header
	}{
		{
			name:       "GET path",
			method:     http.MethodGet,
			path:       "/testGet",
			reqPayload: nil,
			resPayload: []byte("passed request with path /testGet to Run"),
			statusCode: http.StatusOK,
		},
		{
			name:       "PUT path",
			method:     http.MethodPut,
			path:       "/testPut",
			statusCode: http.StatusAccepted,
			reqPayload: bytes.NewBufferString("aPayload"),
			resPayload: []byte("Saved input of type: Text"),
		},
		{
			name:       "OPTIONS",
			method:     http.MethodOptions,
			path:       "/testPut",
			statusCode: http.StatusOK,
			resHeader: http.Header{
				"Access-Control-Allow-Methods":     []string{http.MethodPut},
				"Access-Control-Allow-Headers":     []string{"accept, content-type"},
				"Access-Control-Max-Age":           []string{"1728000"},
				"Access-Control-Allow-Credentials": []string{"true"},
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, test.reqPayload)
			responseRecorder := httptest.NewRecorder()
			h.handle(responseRecorder, request)
			assert.Equal(t, test.statusCode, responseRecorder.Code)
			if len(test.resPayload) > 0 {
				assert.Equal(t, test.resPayload, responseRecorder.Body.Bytes())
			}
			if len(test.resHeader) > 0 {
				assert.Equal(t, test.resHeader, responseRecorder.HeaderMap)
			}
		})
	}
}
