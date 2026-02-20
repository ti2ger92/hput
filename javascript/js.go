package javascript

import (
	"errors"
	"fmt"
	"net/http"

	"go.kuoruan.net/v8go-polyfills/fetch"
	"go.kuoruan.net/v8go-polyfills/timers"
	v8 "rogchap.com/v8go"
)

// Logger logs out.
type Logger interface {
	Debugf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
}

// Javascript runs javascript.
type Javascript struct {
	Logger Logger             // used to log out
	Global *v8.ObjectTemplate // Template injected with add-ons
}

var (
	ErrCreateIsolateRun = errors.New("Error creating an isolate while running code")
	ErrPolyfillsInject  = errors.New("injecting a polyfill")
	ErrFetchInject      = errors.New("injecting fetch into the context")
	ErrTimersInject     = errors.New("injecting timers into the context")
)

// New creates a new javascript interpreter
func New(l Logger) (Javascript, error) {
	global := v8.NewObjectTemplate(v8.NewIsolate())
	v := Javascript{
		Logger: l,
		Global: global,
	}
	return v, nil
}

// isolate return an isolated runtime with needed values filled in
func (j *Javascript) isolate() (*v8.Isolate, error) {
	i := v8.NewIsolate()
	if err := fetch.InjectTo(i, j.Global); err != nil {
		j.Logger.Errorf("Unable to inject the fetch polyfill to runVM: %v", err)
		return nil, fmt.Errorf("%w: %w: %w", ErrPolyfillsInject, ErrFetchInject, err)
	}
	if err := timers.InjectTo(i, j.Global); err != nil {
		j.Logger.Errorf("Unable to inject the timers polyfill to runVM: %v", err)
		return nil, fmt.Errorf("%w: %w, %w", ErrPolyfillsInject, nil, err)
	}
	return i, nil
}

// IsCode tells whether the string is valid javascript code and returns a message why it is not
func (j *Javascript) IsCode(s string) (bool, string) {
	// attempt to compile for testing only, incoming source
	j.Logger.Debugf("testing code: %s", s)
	iso, err := j.isolate()
	if err != nil {
		return false, fmt.Sprintf("Could not create an isolated runtime for this process: %v", err)
	}
	ctx := v8.NewContext(iso, j.Global)
	exp := express{
		Logger: j.Logger,
		RunVM:  iso,
		ctx:    ctx,
		Global: j.Global,
	}
	if err = exp.attachRequest(new(http.Request)); err != nil {
		return false, fmt.Sprintf("Could not add a request object to the context %v", err)
	}
	err = exp.attachResponse(nil)
	if err != nil {
		j.Logger.Errorf("Could not attach a response to the object")
		return false, fmt.Sprintf("could not set the script response object: %v", err)
	}
	if _, err := ctx.Isolate().CompileUnboundScript(s, "your_input", v8.CompileOptions{}); err != nil {
		msg := "I think this is not javascript, so I'll treat it as text.\n"
		msg = msg + fmt.Sprintf("If this were javascript, the error would be: %v", err)
		return false, msg
	}
	return true, ""
}

// Run runs the javascript at a location and writes results to the response.
// Adds objects to the global context
// console.log logs out at INFO level
// request: has express fields for: body, cookies, hostname, ip, method, path, protocol, query
// response: has express functions for: append, cookie, json, location, redirect, sendStatus, set, status
func (j *Javascript) Run(c string, r *http.Request, w http.ResponseWriter) error {
	j.Logger.Debugf("Running code: %s", c)
	iso, err := j.isolate()
	if err != nil {
		return fmt.Errorf("%w, %w", ErrCreateIsolateRun, err)
	}
	ctx := v8.NewContext(iso, j.Global)
	defer ctx.Close()
	exp := express{
		Logger: j.Logger,
		RunVM:  iso,
		ctx:    ctx,
		Global: j.Global,
	}
	if err = exp.attachRequest(r); err != nil {
		j.Logger.Errorf("Could not add a request object to the context %+v", err)
		return fmt.Errorf("could not set the script request object: %w", err)
	}
	if err = exp.attachResponse(w); err != nil {
		j.Logger.Errorf("Could not attach a response to the object")
		return fmt.Errorf("could not set the script response object: %w", err)
	}
	// Add a console.log capability
	// FIXME: move this to new()
	console := v8.NewObjectTemplate(iso)
	logFn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("Provide exactly 1 argument")
		}
		value := info.Args()[0].String()
		j.Logger.Infof(value)
		return nil
	})
	console.Set("log", logFn)
	consoleObj, err := console.NewInstance(ctx)
	if err != nil {
		j.Logger.Errorf("javascript.Run(): failure creating console object: %+v", err)
		return fmt.Errorf("failure creating console object: %w", err)
	}
	global := ctx.Global()
	global.Set("console", consoleObj)
	val, err := ctx.RunScript(c, "your_function")
	if err != nil {
		j.Logger.Errorf("Got an error running the script: %+v", err)
		return fmt.Errorf("got an error running the script: %w", err)
	}
	if val.IsObject() {
		j.Logger.Debugf("response was object")
		bytes, err := val.MarshalJSON()
		if err != nil {
			j.Logger.Errorf("Got an error outputting the string of a json response")
			return fmt.Errorf("got an error outputting the string of a json response: %w", err)
		}
		w.Write(bytes)
	} else if val.IsString() || val.IsInt32() || val.IsBigInt() || val.IsBoolean() {
		j.Logger.Debugf("response was primitive")
		w.Write([]byte(val.String()))
	}
	return nil
}

// cookiesToValue convert an incoming cookie to the expected express cookie
func cookiesToValue(vm *v8.Isolate, ctx *v8.Context, cs []*http.Cookie) (*v8.Value, error) {
	mapTmp := v8.NewObjectTemplate(vm)
	for _, c := range cs {
		mapTmp.Set(c.Name, c.Value)
	}
	obj, err := mapTmp.NewInstance(ctx)
	return obj.Value, err
}

// valuesMapObject convert a map[string][]string to a v8go object
func valuesMapObject(vm *v8.Isolate, ctx *v8.Context, aMap map[string][]string) (*v8.Object, error) {
	tmpl := v8.NewObjectTemplate(vm)
	obj, err := tmpl.NewInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't create string object from template for map: %w", err)
	}
	for key, val := range aMap {
		strArr, err := strArrayObject(ctx, val)
		if err != nil {
			return nil, fmt.Errorf("could not create string array for map: %w", err)
		}
		obj.Set(key, strArr.Value)
	}

	return obj, nil
}

// strArrayObject create a string array from object
func strArrayObject(ctx *v8.Context, arr []string) (*v8.Object, error) {
	// TODO: compile this
	arrayValue, err := ctx.RunScript("[]", "new_array")
	if err != nil {
		return nil, fmt.Errorf("could not create initial array value %w", err)
	}
	obj := arrayValue.Object()

	err = obj.Set("length", int32(len(arr)))
	if err != nil {
		return nil, fmt.Errorf("could not set length of array object %w", err)
	}

	for i, str := range arr {
		obj.SetIdx(uint32(i), str)
	}

	return obj, nil
}
