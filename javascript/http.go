package javascript

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	v8 "rogchap.com/v8go"
)

// express attaches express-compatible objects to an isolated context
type express struct {
	Logger Logger      // used to log out
	RunVM  *v8.Isolate // Parent runtime VM
	ctx    *v8.Context // isolated context where process will run
}

// getRequest attaches a compatible express 4.x `request` to global of context
func (e *express) attachRequest(r *http.Request) error {
	req := v8.NewObjectTemplate(e.RunVM)
	getFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("Provide exactly 1 argument")
		}
		key := info.Args()[0].String()
		h := r.Header[key]
		if len(h) == 0 {
			return v8.Undefined(e.RunVM)
		}
		if len(h) == 1 {
			val, err := v8.NewValue(e.RunVM, h[0])
			if err != nil {
				panic(err)
			}
			return val
		}
		arr, err := parseToValue(e.RunVM, e.ctx, h)
		if err != nil {
			panic(err)
		}
		return arr
	})
	req.Set("get", getFn)
	req.Set("header", getFn)
	reqObj, err := req.NewInstance(e.ctx)
	if err != nil {
		e.Logger.Errorf("javascript.attachRequest(): failure creating request object: %+v", err)
		return fmt.Errorf("failure creating request object: %w", err)
	}
	uriComponents := parseComponents(r)
	if len(uriComponents) > 0 {
		reqObj.Set("baseUrl", uriComponents[0])
	}
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			e.Logger.Errorf("javascript.attachRequest(): error obtaining the incoming body: %+v", err)
			return fmt.Errorf("could not get incoming body: %w", err)
		}
		reqObj.Set("body", string(bodyBytes))
	}
	cVal, err := cookiesToValue(e.RunVM, e.ctx, r.Cookies())
	if err != nil {
		e.Logger.Errorf("javascript.attachRequest(): error obtaining the incoming cookies: %+v", err)
		return fmt.Errorf("could not get incoming cookies: %w", err)
	}
	reqObj.Set("cookies", cVal)
	if r.Header != nil {
		hVal, err := parseToValue(e.RunVM, e.ctx, r.Header)
		if err != nil {
			e.Logger.Errorf("javascript.attachRequest(): error obtaining the incoming headers: %+v", err)
			return fmt.Errorf("could not get incoming headers: %w", err)
		}
		reqObj.Set("headers", hVal)
	}
	reqObj.Set("hostname", r.Host)
	reqObj.Set("ip", r.RemoteAddr)
	reqObj.Set("method", r.Method)
	reqObj.Set("path", r.URL.Path)
	reqObj.Set("protocol", strings.ToLower(r.Proto))
	q := r.URL.Query()
	if q != nil {
		qVal, err := parseToValue(e.RunVM, e.ctx, q)
		if err != nil {
			e.Logger.Errorf("javascript.attachRequest(): error obtaining the incoming query: %+v", err)
			return fmt.Errorf("could not get incoming query: %w", err)
		}
		reqObj.Set("query", qVal)
	}
	global := e.ctx.Global()
	global.Set("request", reqObj)
	return nil
}

// parseComponents of a URL string
func parseComponents(r *http.Request) []string {
	//The URL that the user queried.
	path := r.URL.Path
	path = strings.TrimSpace(path)
	//Cut off the leading and trailing forward slashes, if they exist.
	//This cuts off the leading forward slash.
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	//This cuts off the trailing forward slash.
	if strings.HasSuffix(path, "/") {
		cut_off_last_char_len := len(path) - 1
		path = path[:cut_off_last_char_len]
	}
	//We need to isolate the individual components of the path.
	components := strings.Split(path, "/")
	return components
}

// attachResponse attaches a response object to the context
// you can use this to send responses.
func (e *express) attachResponse(w http.ResponseWriter) error {
	res := v8.NewObjectTemplate(e.RunVM)
	var resObj *v8.Object
	appendFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) < 2 {
			panic("too few arguments provided, must provide at least 2")
		}
		key := info.Args()[0].String()
		value := info.Args()[1].String()
		w.Header().Add(key, value)
		return resObj.Value
	})
	res.Set("append", appendFn)
	cookieFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) < 2 {
			panic("too few arguments provided, must provide at least 2")
		}
		name := info.Args()[0].String()
		value := info.Args()[1].String()
		c := &http.Cookie{
			Name:  name,
			Value: value,
		}
		if len(info.Args()) > 2 {
			obj, err := info.Args()[2].AsObject()
			if err != nil {
				panic(fmt.Sprintf("options passed but could not understand them : %+v", err))
			}
			e.parseCookieOpts(obj, c)
		}
		http.SetCookie(w, c)
		return resObj.Value
	})
	res.Set("cookie", cookieFn)

	jsonGoFn := func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("should have 1 argument")
		}
		j, err := info.Args()[0].MarshalJSON()
		if err != nil {
			panic("Invalid argument for json")
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(j)
		return resObj.Value
	}
	jsonFn := v8.NewFunctionTemplate(e.RunVM, jsonGoFn)
	res.Set("json", jsonFn)
	locationFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("provide exactly 1 argument")
		}
		value := info.Args()[0].String()
		w.Header().Add("Location", value)
		return nil
	})
	res.Set("location", locationFn)
	redirectFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) < 1 || len(info.Args()) > 2 {
			panic("provide 1 or 2 arguments")
		}
		if len(info.Args()) == 2 {
			c := info.Args()[0].Integer()
			v := info.Args()[1].String()
			w.WriteHeader(int(c))
			w.Write([]byte(v))
		}
		v := info.Args()[0].String()
		w.WriteHeader(http.StatusFound)
		w.Write([]byte(v))
		return resObj.Value
	})
	res.Set("redirect", redirectFn)
	sendFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("Provide 1 parameter")
		}
		if info.Args()[0].IsObject() {
			return jsonGoFn(info)
		}
		// TODO: support more types
		w.Write([]byte(info.Args()[0].String()))
		return resObj.Value
	})
	res.Set("send", sendFn)
	sendStatusFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("Provide 1 parameter")
		}
		if !info.Args()[0].IsInt32() {
			panic("Provide an integer")
		}
		w.WriteHeader(int(info.Args()[0].Int32()))
		return resObj.Value
	})
	res.Set("sendStatus", sendStatusFn)
	setFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) < 1 || len(info.Args()) > 2 {
			panic("Provide 1 or 2 parameters")
		}
		k := info.Args()[0].String()
		if len(info.Args()) == 1 {
			w.Header().Set(k, "")
			return nil
		}
		v := info.Args()[1].String()
		w.Header().Set(k, v)
		return resObj.Value
	})
	res.Set("set", setFn)
	statusFn := v8.NewFunctionTemplate(e.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("Provide 1 parameter")
		}
		s := info.Args()[0].Int32()
		w.WriteHeader(int(s))
		return resObj.Value
	})
	res.Set("status", statusFn)
	resObj, err := res.NewInstance(e.ctx)
	if err != nil {
		e.Logger.Errorf("javascript.attachResponse(): failure creating request object: %+v", err)
		return fmt.Errorf("failure creating request object: %w", err)
	}
	global := e.ctx.Global()
	global.Set("response", resObj)
	return nil
}

// parseCookieOpts parse a cookie option as a v8.Object to a cookie
func (e *express) parseCookieOpts(obj *v8.Object, c *http.Cookie) {
	v, err := obj.Get("httpOnly")
	if err != nil {
		e.Logger.Debugf("could not parse HttpOnly from cookie request: %+v", err)
	}
	if v.IsBoolean() {
		c.HttpOnly = v.Boolean()
	}

	v, err = obj.Get("domain")
	if err != nil {
		e.Logger.Debugf("could not parse domain from cookie request: %+v", err)
	}
	if v.IsString() {
		c.Domain = v.String()
	}

	v, err = obj.Get("maxAge")
	if err != nil {
		e.Logger.Debugf("could not parse maxAge from cookie request: %+v", err)
	}
	if v.IsInt32() {
		c.MaxAge = int(v.Int32())
	}

	v, err = obj.Get("expires")
	if err != nil {
		e.Logger.Debugf("could not parse expires from cookie request: %+v", err)
	}
	if v.IsDate() {
		dbytes, err := v.MarshalJSON()
		if err != nil {
			e.Logger.Debugf("could not parse expires from cookie request: %+v", err)
		}
		s := strings.Replace(string(dbytes), `"`, "", -1)
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			e.Logger.Debugf("could not parse expires from cookie request: %+v", err)
		}
		c.Expires = t
	}

	v, err = obj.Get("path")
	if err != nil {
		e.Logger.Debugf("could not parse domain from path request: %+v", err)
	}
	if v.IsString() {
		c.Path = v.String()
	}

	v, err = obj.Get("secure")
	if err != nil {
		e.Logger.Debugf("could not parse HttpOnly from cookie request: %+v", err)
	}
	if v.IsBoolean() {
		c.Secure = v.Boolean()
	}
}
