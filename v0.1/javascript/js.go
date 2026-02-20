package javascript

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"hput/internal/polyfills"

	v8 "github.com/tommie/v8go"
)

// Logger logs out.
type Logger interface {
	Debugf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
}

// Javascript runs javascript.
type Javascript struct {
	Logger Logger
}

var (
	ErrCreateIsolateRun = errors.New("Error creating an isolate while running code")
	ErrPolyfillsInject  = errors.New("injecting a polyfill")
	ErrFetchInject      = errors.New("injecting fetch into the context")
	ErrTimersInject     = errors.New("injecting timers into the context")
)

// New creates a new javascript interpreter
func New(l Logger) (Javascript, error) {
	return Javascript{Logger: l}, nil
}

// newContext creates a fresh isolate, context, and event loop with polyfills injected.
func (j *Javascript) newContext() (*v8.Isolate, *v8.Context, *polyfills.EventLoop, error) {
	iso := v8.NewIsolate()
	ctx := v8.NewContext(iso)
	el := polyfills.NewEventLoop()

	if err := polyfills.InjectFetch(iso, ctx); err != nil {
		ctx.Close()
		return nil, nil, nil, fmt.Errorf("%w: %w: %w", ErrPolyfillsInject, ErrFetchInject, err)
	}
	if err := polyfills.InjectTimers(iso, ctx, el); err != nil {
		ctx.Close()
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrPolyfillsInject, ErrTimersInject)
	}
	return iso, ctx, el, nil
}

// IsCode tells whether the string is valid javascript code and returns a message why it is not
func (j *Javascript) IsCode(s string) (bool, string) {
	j.Logger.Debugf("testing code: %s", s)
	iso := v8.NewIsolate()
	defer iso.Dispose()
	if _, err := iso.CompileUnboundScript(s, "your_input", v8.CompileOptions{}); err != nil {
		msg := "I think this is not javascript, so I'll treat it as text.\n"
		msg = msg + fmt.Sprintf("If this were javascript, the error would be: %v", err)
		return false, msg
	}
	return true, ""
}

// Run runs the javascript at a location and writes results to the response.
// Adds objects to the global context:
// console.log logs out at INFO level
// request: has express fields for: body, cookies, hostname, ip, method, path, protocol, query
// response: has express functions for: append, cookie, json, location, redirect, sendStatus, set, status
// fetch: standard fetch API
// setTimeout/setInterval/clearTimeout/clearInterval: timer APIs
func (j *Javascript) Run(c string, r *http.Request, w http.ResponseWriter) error {
	j.Logger.Debugf("Running code: %s", c)

	iso, ctx, el, err := j.newContext()
	if err != nil {
		return fmt.Errorf("%w, %w", ErrCreateIsolateRun, err)
	}
	defer ctx.Close()

	exp := express{
		Logger: j.Logger,
		RunVM:  iso,
		ctx:    ctx,
	}
	if err = exp.attachRequest(r); err != nil {
		j.Logger.Errorf("Could not add a request object to the context %+v", err)
		return fmt.Errorf("could not set the script request object: %w", err)
	}
	if err = exp.attachResponse(w); err != nil {
		j.Logger.Errorf("Could not attach a response to the object")
		return fmt.Errorf("could not set the script response object: %w", err)
	}

	console := v8.NewObjectTemplate(iso)
	logFn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.String()
		}
		j.Logger.Infof("%s", joinStrings(parts))
		return nil
	})
	console.Set("log", logFn)
	consoleObj, err := console.NewInstance(ctx)
	if err != nil {
		return fmt.Errorf("failure creating console object: %w", err)
	}
	ctx.Global().Set("console", consoleObj)

	val, err := ctx.RunScript(c, "your_function")
	if err != nil {
		j.Logger.Errorf("Got an error running the script: %+v", err)
		return fmt.Errorf("got an error running the script: %w", err)
	}

	// Drain microtasks so any awaited Promises settle.
	ctx.PerformMicrotaskCheckpoint()

	// If the script returned a Promise, wait for it to resolve.
	if val != nil && val.IsPromise() {
		promise, _ := val.AsPromise()
		deadline := time.Now().Add(30 * time.Second)
		for promise.State() == v8.Pending && time.Now().Before(deadline) {
			el.Drain(iso, ctx, deadline)
			ctx.PerformMicrotaskCheckpoint()
		}
		if promise.State() == v8.Rejected {
			return fmt.Errorf("script promise rejected: %s", promise.Result().String())
		}
		val = promise.Result()
	}

	// Drain any remaining timers.
	el.Drain(iso, ctx, time.Now().Add(30*time.Second))

	if val == nil {
		return nil
	}
	if val.IsObject() {
		j.Logger.Debugf("response was object")
		bytes, err := val.MarshalJSON()
		if err != nil {
			return fmt.Errorf("got an error outputting the string of a json response: %w", err)
		}
		w.Write(bytes)
	} else if val.IsString() || val.IsInt32() || val.IsBigInt() || val.IsBoolean() {
		j.Logger.Debugf("response was primitive")
		w.Write([]byte(val.String()))
	}
	return nil
}

func joinStrings(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
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
	arrayValue, err := ctx.RunScript("[]", "new_array")
	if err != nil {
		return nil, fmt.Errorf("could not create initial array value %w", err)
	}
	obj := arrayValue.Object()
	if err = obj.Set("length", int32(len(arr))); err != nil {
		return nil, fmt.Errorf("could not set length of array object %w", err)
	}
	for i, str := range arr {
		obj.SetIdx(uint32(i), str)
	}
	return obj, nil
}
