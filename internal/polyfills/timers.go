// Adapted from github.com/cryguy/hostedat/internal/worker/timers.go
// Copyright (c) cryguy/hostedat contributors. MIT License.
// See THIRD_PARTY_LICENSES for full license text.
package polyfills

import (
	"time"

	v8 "github.com/tommie/v8go"
)

// InjectTimers registers setTimeout/clearTimeout/setInterval/clearInterval
// into the context's global object.
func InjectTimers(iso *v8.Isolate, ctx *v8.Context, el *EventLoop) error {
	setTimeoutFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) < 1 || !args[0].IsFunction() {
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}
		fn, err := args[0].AsFunction()
		if err != nil {
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}
		var delay time.Duration
		if len(args) > 1 {
			delay = time.Duration(args[1].Int32()) * time.Millisecond
		}
		id := el.setTimeout(fn, delay)
		val, _ := v8.NewValue(iso, int32(id))
		return val
	})
	if err := ctx.Global().Set("setTimeout", setTimeoutFT.GetFunction(ctx)); err != nil {
		return err
	}

	clearTimeoutFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			el.clearTimer(int(args[0].Int32()))
		}
		return v8.Undefined(iso)
	})
	if err := ctx.Global().Set("clearTimeout", clearTimeoutFT.GetFunction(ctx)); err != nil {
		return err
	}

	setIntervalFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) < 1 || !args[0].IsFunction() {
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}
		fn, err := args[0].AsFunction()
		if err != nil {
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}
		interval := 10 * time.Millisecond
		if len(args) > 1 && args[1].Int32() > 0 {
			interval = time.Duration(args[1].Int32()) * time.Millisecond
		}
		id := el.setInterval(fn, interval)
		val, _ := v8.NewValue(iso, int32(id))
		return val
	})
	if err := ctx.Global().Set("setInterval", setIntervalFT.GetFunction(ctx)); err != nil {
		return err
	}

	clearIntervalFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			el.clearTimer(int(args[0].Int32()))
		}
		return v8.Undefined(iso)
	})
	return ctx.Global().Set("clearInterval", clearIntervalFT.GetFunction(ctx))
}
