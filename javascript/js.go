package javascript

import (
	"fmt"
	"net/http"

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
	Logger Logger      // used to log out
	TestVM *v8.Isolate // VM used only for testing
	RunVM  *v8.Isolate // VM used to run everyone's code
}

// New creates a new javascript interpreter
func New(l Logger) Javascript {
	runVM := v8.NewIsolate()
	return Javascript{
		Logger: l,
		TestVM: v8.NewIsolate(),
		RunVM:  runVM,
	}
}

// IsCode tells whether the string is valid javascript code and returns a message why it is not
func (j *Javascript) IsCode(s string) (bool, string) {
	// attempt to compile for testing only, incoming source
	j.Logger.Debugf("testing code: %s", s)
	if _, err := j.TestVM.CompileUnboundScript(s, "your_input", v8.CompileOptions{}); err != nil {
		msg := "I think this is not javascript, so we'll treat it as text.\n"
		msg = msg + fmt.Sprintf("If this were javascript, the error would be: %+v", err)
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
	ctx := v8.NewContext(j.RunVM)
	defer ctx.Close()
	exp := express{
		Logger: j.Logger,
		RunVM:  j.RunVM,
		ctx:    ctx,
	}
	err := exp.attachRequest(r)
	if err != nil {
		j.Logger.Errorf("Could not add a request object to the context %+v", err)
		return fmt.Errorf("could not set the script request object: %w", err)
	}
	err = exp.attachResponse(w)
	if err != nil {
		j.Logger.Errorf("Could not attach a response to the object")
		return fmt.Errorf("could not set the script response object: %w", err)
	}
	// Add a console.log capability
	console := v8.NewObjectTemplate(j.RunVM)
	logFn := v8.NewFunctionTemplate(j.RunVM, func(info *v8.FunctionCallbackInfo) *v8.Value {
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
		j.Logger.Errorf("Got an error running the script")
		return fmt.Errorf("got an error running the script: %w", err)
	}
	if val.IsObject() {
		bytes, err := val.MarshalJSON()
		if err != nil {
			j.Logger.Errorf("Got an error outputting the string of a json response")
			return fmt.Errorf("got an error outputting the string of a json response: %w", err)
		}
		w.Write(bytes)
	} else if val.IsString() || val.IsObject() {
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
