// Inspired by github.com/cryguy/hostedat/internal/worker/fetch.go
// Copyright (c) cryguy/hostedat contributors. MIT License.
// See THIRD_PARTY_LICENSES for full license text.
//
// Simplified for hput: no SSRF protection, no rate limiting, no AbortSignal.
package polyfills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	v8 "github.com/tommie/v8go"
)

const maxFetchResponseBytes = 10 * 1024 * 1024 // 10 MB

// InjectFetch registers a global fetch() function into the context.
// The fetch blocks synchronously inside the Go callback and resolves
// the returned Promise immediately, so await works without a separate
// event loop pump for the fetch itself.
func InjectFetch(iso *v8.Isolate, ctx *v8.Context) error {
	fetchFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		resolver, _ := v8.NewPromiseResolver(ctx)
		args := info.Args()

		if len(args) == 0 {
			errVal, _ := v8.NewValue(iso, "fetch requires at least 1 argument")
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}

		// Pass args through JS to extract url/method/headers/body cleanly.
		_ = ctx.Global().Set("__fetch_a0", args[0])
		if len(args) > 1 {
			_ = ctx.Global().Set("__fetch_a1", args[1])
		} else {
			_ = ctx.Global().Set("__fetch_a1", v8.Undefined(iso))
		}

		extractVal, err := ctx.RunScript(`(function() {
			var a0 = globalThis.__fetch_a0;
			var a1 = globalThis.__fetch_a1;
			delete globalThis.__fetch_a0;
			delete globalThis.__fetch_a1;
			var url = typeof a0 === 'string' ? a0 : (a0 && a0.url) || '';
			var method = 'GET', headers = {}, body = null;
			if (a1 && typeof a1 === 'object') {
				if (a1.method) method = String(a1.method).toUpperCase();
				if (a1.headers) {
					var src = a1.headers;
					for (var k in src) { if (src.hasOwnProperty(k)) headers[k] = String(src[k]); }
				}
				if (a1.body != null) body = String(a1.body);
			}
			return JSON.stringify({url: url, method: method, headers: headers, body: body});
		})()`, "fetch_extract.js")
		if err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: extracting args: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}

		var fetchArgs struct {
			URL     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Body    *string           `json:"body"`
		}
		if err := json.Unmarshal([]byte(extractVal.String()), &fetchArgs); err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: parsing args: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}

		var bodyReader io.Reader
		if fetchArgs.Body != nil && *fetchArgs.Body != "" {
			bodyReader = strings.NewReader(*fetchArgs.Body)
		}

		req, err := http.NewRequest(fetchArgs.Method, fetchArgs.URL, bodyReader)
		if err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}
		for k, v := range fetchArgs.Headers {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchResponseBytes))
		if err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: reading body: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}

		respHeaders := make(map[string]string)
		for k, vals := range resp.Header {
			respHeaders[strings.ToLower(k)] = strings.Join(vals, ", ")
		}
		headersJSON, _ := json.Marshal(respHeaders)

		_ = ctx.Global().Set("__fetch_resp_status", int32(resp.StatusCode))
		_ = ctx.Global().Set("__fetch_resp_status_text", resp.Status)
		_ = ctx.Global().Set("__fetch_resp_body", string(respBody))
		_ = ctx.Global().Set("__fetch_resp_headers", string(headersJSON))

		jsResp, err := ctx.RunScript(`(function() {
			var status = globalThis.__fetch_resp_status;
			var statusText = globalThis.__fetch_resp_status_text;
			var bodyText = globalThis.__fetch_resp_body;
			var headers = JSON.parse(globalThis.__fetch_resp_headers);
			delete globalThis.__fetch_resp_status;
			delete globalThis.__fetch_resp_status_text;
			delete globalThis.__fetch_resp_body;
			delete globalThis.__fetch_resp_headers;
			return {
				ok: status >= 200 && status < 300,
				status: status,
				statusText: statusText,
				headers: headers,
				_bodyText: bodyText,
				json: function() { return Promise.resolve(JSON.parse(this._bodyText)); },
				text: function() { return Promise.resolve(this._bodyText); },
			};
		})()`, "fetch_response.js")
		if err != nil {
			errVal, _ := v8.NewValue(iso, fmt.Sprintf("fetch: building response: %s", err))
			resolver.Reject(errVal)
			return resolver.GetPromise().Value
		}

		resolver.Resolve(jsResp)
		return resolver.GetPromise().Value
	})

	return ctx.Global().Set("fetch", fetchFT.GetFunction(ctx))
}
