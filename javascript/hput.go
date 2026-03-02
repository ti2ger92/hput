package javascript

import (
	"context"
	"encoding/json"
	"fmt"
	"hput/kv"

	v8 "github.com/tommie/v8go"
)

// attachHput injects the `hput` global object into the V8 context.
// All KV operations are scoped to path — the JS caller never specifies
// which path they belong to.
func attachHput(ctx context.Context, iso *v8.Isolate, v8ctx *v8.Context, path string, store kv.KV) error {
	hputTmpl := v8.NewObjectTemplate(iso)

	// hput.get(key) → value | null
	hputTmpl.Set("get", v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("hput.get requires exactly 1 argument")
		}
		key := info.Args()[0].String()
		val, err := store.Get(ctx, path, key)
		if err != nil {
			panic(fmt.Sprintf("hput.get: %s", err))
		}
		if val == nil {
			return v8.Null(iso)
		}
		parsed, err := v8ctx.RunScript("JSON.parse("+jsonStringLiteral(string(val))+")", "hput_get")
		if err != nil {
			panic(fmt.Sprintf("hput.get: parsing stored value: %s", err))
		}
		return parsed
	}))

	// hput.put(key, value) → undefined
	hputTmpl.Set("put", v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 2 {
			panic("hput.put requires exactly 2 arguments")
		}
		key := info.Args()[0].String()
		valueJSON, err := info.Args()[1].MarshalJSON()
		if err != nil {
			panic(fmt.Sprintf("hput.put: serializing value: %s", err))
		}
		if err := store.Put(ctx, path, key, valueJSON); err != nil {
			panic(fmt.Sprintf("hput.put: %s", err))
		}
		return v8.Undefined(iso)
	}))

	// hput.delete(key) → undefined
	hputTmpl.Set("delete", v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		if len(info.Args()) != 1 {
			panic("hput.delete requires exactly 1 argument")
		}
		key := info.Args()[0].String()
		if err := store.Delete(ctx, path, key); err != nil {
			panic(fmt.Sprintf("hput.delete: %s", err))
		}
		return v8.Undefined(iso)
	}))

	// hput.list(opts?) → { keys: string[], cursor: string }
	// opts: { prefix?: string, limit?: number, cursor?: string }
	hputTmpl.Set("list", v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		opts := kv.ListOptions{}
		if len(info.Args()) > 0 && info.Args()[0].IsObject() {
			obj, err := info.Args()[0].AsObject()
			if err == nil {
				if v, err := obj.Get("prefix"); err == nil && v.IsString() {
					opts.Prefix = v.String()
				}
				if v, err := obj.Get("limit"); err == nil && v.IsInt32() {
					opts.Limit = int(v.Int32())
				}
				if v, err := obj.Get("cursor"); err == nil && v.IsString() {
					opts.Cursor = v.String()
				}
			}
		}

		result, err := store.List(ctx, path, opts)
		if err != nil {
			panic(fmt.Sprintf("hput.list: %s", err))
		}

		// Build { keys: [...], cursor: "..." } as a JSON string then parse it.
		keysJSON, _ := json.Marshal(result.Keys)
		cursorJSON, _ := json.Marshal(result.Cursor)
		script := fmt.Sprintf("({keys:%s,cursor:%s})", keysJSON, cursorJSON)
		val, err := v8ctx.RunScript(script, "hput_list")
		if err != nil {
			panic(fmt.Sprintf("hput.list: building result: %s", err))
		}
		return val
	}))

	hputObj, err := hputTmpl.NewInstance(v8ctx)
	if err != nil {
		return fmt.Errorf("creating hput object: %w", err)
	}
	return v8ctx.Global().Set("hput", hputObj)
}

// jsonStringLiteral returns a JavaScript string literal containing s,
// safe to embed directly in a script (e.g. JSON.parse(<result>)).
func jsonStringLiteral(s string) string {
	b, _ := json.Marshal(s) // json.Marshal of a string produces a valid JS string literal
	return string(b)
}
