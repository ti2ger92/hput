# Plan: Migrate from rogchap/v8go to tommie/v8go

## Goal

Replace the dead `rogchap.com/v8go v0.9.0` dependency with the actively maintained
`github.com/tommie/v8go v0.34.0` fork, which bundles V8 13.6 (vs V8 11.1 currently).
Replace the incompatible `go.kuoruan.net/v8go-polyfills` with fetch/timers code lifted
from `github.com/cryguy/hostedat`, which is already built and tested against tommie v0.34.0.

---

## MIT License Compliance

Both `github.com/tommie/v8go` and `github.com/cryguy/hostedat` are MIT licensed.
We are permitted to copy, modify, and use the code freely, including commercially.

**Required action before shipping:** Add the original copyright notices for both
projects to a `THIRD_PARTY_LICENSES` file or similar in the repo root. This is the
only obligation MIT imposes.

---

## Steps

### 1. Update `go.mod`

- Remove `rogchap.com/v8go v0.9.0`
- Remove `go.kuoruan.net/v8go-polyfills v0.5.0`
- Add `github.com/tommie/v8go v0.34.0`

### 2. Swap import paths

In all Go files (`v0.1/javascript/js.go`, `v0.1/javascript/http.go`,
`javascript/js.go`, `javascript/http.go`):

- Replace `rogchap.com/v8go` → `github.com/tommie/v8go`
- Remove imports of `go.kuoruan.net/v8go-polyfills/fetch` and
  `go.kuoruan.net/v8go-polyfills/timers`

### 3. Lift fetch + timers from cryguy/hostedat

Read and extract the relevant code from `github.com/cryguy/hostedat/internal/worker`:

- `fetch.go` — full `fetch()` implementation using `NewFunctionTemplate` +
  `NewPromiseResolver` + goroutine-based async HTTP
- `timers.go` — `setTimeout`, `clearTimeout`, `setInterval`, `clearInterval`
  backed by an event loop
- `eventloop.go` — the event loop that drains timers and promise microtasks

Place the extracted code into a new `internal/polyfills/` package in this repo.
Strip any Cloudflare Workers-specific logic (SSRF protection, rate limiting,
`unenv` npm polyfills) that doesn't apply here. Keep only what replaces
`kuoruan/v8go-polyfills`.

### 4. Wire up the new polyfills

In `js.go`, replace the calls to `fetch.InjectTo(i, j.Global)` and
`timers.InjectTo(i, j.Global)` with calls to the new `internal/polyfills` package.

The API surface should remain the same from the JavaScript side:
- `fetch(url, options)` returns a Promise
- `setTimeout(fn, ms)` / `clearTimeout(id)`
- `setInterval(fn, ms)` / `clearInterval(id)`

### 5. Add THIRD_PARTY_LICENSES file

Create `THIRD_PARTY_LICENSES` in the repo root with the full MIT license text and
copyright notices for:
- `github.com/tommie/v8go`
- `github.com/cryguy/hostedat`

### 6. Build and test

- `go build ./...` — verify no compile errors
- `go test ./...` — run existing test suite
- Manual smoke test: start server, PUT a JS function that uses `fetch()` and
  `setTimeout()`, verify both work

### 7. Remove the compiled `hput` binary from the repo

The repo currently tracks a 52 MB compiled binary (`hput`) that triggered a GitHub
large-file warning on the last push. Add `hput` to `.gitignore` and remove it from
tracking with `git rm --cached hput`.

---

## What We Gain

| Feature | Before | After |
|---|---|---|
| V8 version | 11.1 (2022) | 13.6 (2025) |
| Maintenance status | Dead | Active |
| Memory limits per isolate | No | Yes |
| Console output capture | Manual | Built-in inspector |
| Getter/setter on objects | No | Yes |
| Go error → JS exception | No | Yes |

---

## What We Are Not Doing

- Not adding ES module (`import`/`export`) support — still not available in any v8go fork
- Not implementing the `hput` JS global for shared storage — that is a separate effort
  tracked in VISION.md
