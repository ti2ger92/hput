package javascript

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "rogchap.com/v8go"
)

type TestLogger struct{}

func (t *TestLogger) Debugf(msg string, args ...interface{}) {}

func (t *TestLogger) Errorf(msg string, args ...interface{}) {}

func (t *TestLogger) Infof(msg string, args ...interface{}) {}

// Test_IsCode verifies that IsCode runs as expected
func Test_IsCode(t *testing.T) {
	tt := []struct {
		name        string
		code        string
		isCode      bool
		msgIncludes string
	}{
		{
			name:        "basic false",
			code:        "const invalid = 'unclosed single quote",
			msgIncludes: "SyntaxError: Invalid or unexpected token",
		},
		{
			name:   "basic true",
			code:   "const valid = 'closed var'",
			isCode: true,
		},
		{
			name: "mutiline true",
			code: `
const a = 1;
const b = 2;
a + b;
`,
			isCode: true,
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			js := New(&TestLogger{})
			isCode, msg := js.IsCode(test.code)
			assert.Equal(t, test.isCode, isCode)
			assert.Contains(t, msg, test.msgIncludes)
		})
	}
}

// Test_IsCode verifies that IsCode runs as expected
func Test_Run(t *testing.T) {
	tt := []struct {
		name        string
		code        string
		r           *http.Request
		msgIncludes []string
		respHeader  http.Header
		respCode    int
	}{
		// {
		// 	name: "basic simple",
		// 	code: "const valid = 'closed var'; valid;",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	msgIncludes: []string{"closed var"},
		// },
		{
			name: "return request as json",
			code: "request",
			r: &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{Path: "/pth", RawQuery: "key1=val1&key2=val2"},
				Body:   io.NopCloser(bytes.NewBufferString("payload")),
				Host:   "host",
				Header: http.Header{
					"Cookie":     []string{"cookieKey1=cookieValue1; cookieKey2=cookieValue2"},
					"HeaderKey1": []string{`HeaderVal1 with a " quote`},
				},
				RemoteAddr: "ip",
				Proto:      "Proto",
			},
			msgIncludes: []string{
				"\"baseUrl\":\"pth\"",
				"\"body\":\"payload\"",
				"\"hostname\":\"host\"",
				"\"method\":\"POST\"",
				"\"path\":\"/pth\"",
				`"cookieKey2":"cookieValue2"`,
				`"HeaderKey1":["HeaderVal1 with a \" quote"]`,
				`"ip":"ip"`,
				`"protocol":"proto"`,
				`"query":{"key1":["val1"],"key2":["val2"]}`,
			},
		},
		// {
		// 	name: "append 1 item",
		// 	code: "response.append('Key', 'value')",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	respHeader: http.Header{"Key": []string{"value"}},
		// },
		// {
		// 	name: "set cookie",
		// 	code: "response.cookie('cookie', 'cookieValue', {expires: new Date(0), httpOnly: true, domain: 'domain', maxAge: 1, path: '/pth', secure: true})",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	respHeader: http.Header{"Set-Cookie": []string{"cookie=cookieValue; Path=/pth; Domain=domain; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=1; HttpOnly; Secure"}},
		// },
		// {
		// 	name: "output some json",
		// 	code: "response.json({'a':'b'})",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	msgIncludes: []string{`{"a":"b"}`},
		// },
		// {
		// 	name: "send a redirect with code",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	code:        "response.redirect(311, 'redirected')",
		// 	msgIncludes: []string{"redirected"},
		// 	respCode:    311,
		// },
		// {
		// 	name: "send a redirect",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	code:        "response.redirect('redirected')",
		// 	msgIncludes: []string{"redirected"},
		// 	respCode:    302,
		// },
		// {
		// 	name: "output via send",
		// 	code: "response.send({'a':'b'})",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	msgIncludes: []string{`{"a":"b"}`},
		// },
		// {
		// 	name: "send status",
		// 	code: "response.sendStatus(100)",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	respCode: 100,
		// },
		// {
		// 	name: "set a header",
		// 	code: "response.set('hdrKey', 'hdrVal')\nresponse.set('hdrKey')\nresponse.set('hdrKey2', 'hdrVal2')",
		// 	r: &http.Request{
		// 		Method: http.MethodGet,
		// 		URL:    &url.URL{Path: "/pth"},
		// 	},
		// 	respHeader: http.Header{
		// 		"Hdrkey":  []string{},
		// 		"Hdrkey2": []string{"hdrVal2"},
		// 	},
		// },
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			js := New(&TestLogger{})
			responseRecorder := httptest.NewRecorder()
			err := js.Run(test.code, test.r, responseRecorder)
			assert.NoError(t, err)
			for _, msg := range test.msgIncludes {
				assert.Contains(t, responseRecorder.Body.String(), msg)
			}
			for k, hdr := range test.respHeader {
				assert.NotNil(t, responseRecorder.Header()[k])
				for i, s := range hdr {
					assert.Equal(t, s, responseRecorder.Header()[k][i])
				}
			}
			if test.respCode > 0 {
				assert.Equal(t, test.respCode, responseRecorder.Code)
			}
		})
	}
}

func Test_parseToValue(t *testing.T) {
	runVM := v8.NewIsolate()
	ctx := v8.NewContext(runVM)
	tt := []struct {
		name   string
		header http.Header
	}{
		{
			name:   "one header",
			header: http.Header{"a": []string{"b", "c"}},
		},
		{
			name: "some headers",
			header: http.Header{
				"a": []string{"b", "c"},
				"b": []string{"val"},
				"c": []string{"x", "y", "z"},
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			val, err := parseToValue(runVM, ctx, test.header)
			assert.NoError(t, err)
			global := ctx.Global()
			global.Set("testValue", val)
			for k, h := range test.header {
				for i, s := range h {
					res, err := ctx.RunScript(fmt.Sprintf(`testValue['%s'][%d]`, k, i), "test_script")
					assert.NoError(t, err)
					assert.Equal(t, s, res.String())
				}
			}
		})
	}
}
