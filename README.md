An http server you can program via http — no console access needed.
- `PUT` web pages, text, pictures, or any file, then `GET` (or `POST`, `PATCH`, `DELETE`) the same path to get it back.
- `PUT` javascript to a path, then call that path with any verb to run your code.
  - Express-style `request` and `response` objects are available in your code.
  - `fetch()`, `setTimeout`, and `setInterval` are available too.
  - Your code can read and write private key-value storage via `hput.get/put/list/delete`.

# hput

## Start

```
go mod download
go run cmd/hput/main.go
```

Default port is `80`. Stop with `ctrl-c`.

## Start flags
| flag | default | description |
| - | - | - |
| `-port` | `80` | port to listen on |
| `-nonlocal` | `false` | allow traffic from outside localhost |
| `-storage` | `local` | `local`, `memory`, or `s3` |
| `-filename` | `hput.db` | file to use for local storage |
| `-kv-backend` | `bbolt` | KV backend for JS private storage (`bbolt`) |
| `-kv-file` | `hput-kv.db` | file to use for bbolt KV storage |
| `-locked` | `false` | disable PUT — serve existing content only |
| `-log` | `info` | `debug`, `warn`, or `error` |
| `-bucket` | | S3 bucket name |
| `-prefix` | | S3 key prefix |

### Docker
```
docker-compose up -d
docker-compose down
```

## Use

PUT something to your server. You can run this in a browser console:
```javascript
var xhr = new XMLHttpRequest()
xhr.open("PUT", "http://localhost/hello")
xhr.send("Hello hput")
```

Then visit `http://localhost/hello`.

### Save your work
Visit `http://localhost/dump` to get javascript that will recreate everything on another hput server. You can also dump a subpath: `http://localhost/hello/dump`.

## Example payloads

#### HTML
```html
<html>
<h1>hello hput</h1>
```

#### HTML with browser javascript
```html
<html>
<script>
    document.write(`It is ${new Date()} on your computer`)
</script>
```

#### Server-side javascript
```javascript
`tomorrow it will be: ${new Date(new Date().getTime() + 86400000)}`
```

#### Fetch from another path
Put this at `/time`:
```javascript
new Date()
```

Put this at `/index.html`:
```html
<html>
<div id="t"></div>
<script>
    fetch("/time").then(r => r.text()).then(t => document.getElementById("t").innerHTML = t)
</script>
```

## Javascript

Your server-side javascript runs in V8. A few things to know:

- Wrap async code in an async IIFE: `(async () => { ... })()`
- Return a value on the last line, or use `response.send()` / `response.json()`
- `console.log` logs at INFO level

#### `request`
| field | type | description |
| - | - | - |
| `get` | function | retrieve a request header |
| `body` | string | request body |
| `cookies` | object | cookies on the request |
| `hostname` | string | host making the request |
| `ip` | string | IP address making the request |
| `method` | string | HTTP verb |
| `path` | string | URL path |
| `protocol` | string | `http` or `https` |
| `query` | object | query string parameters |
| `headers` | object | all request headers |

#### `response`
| function | description |
| - | - |
| `send(value)` | send a response body |
| `json(value)` | send JSON |
| `status(code)` | set HTTP status code |
| `sendStatus(code)` | send a status with no body |
| `set(key, value)` | set a response header |
| `append(key, value)` | append to a response header |
| `cookie(name, value)` | set a cookie |
| `location(url)` | set the Location header |
| `redirect(url)` | send an HTTP redirect |

#### `hput` — private key-value storage

Each path gets its own isolated KV store. JS at `/users` cannot read `/orders`'s data.

```javascript
(async () => {
    // store anything JSON-serializable
    await hput.put('count', 42)
    await hput.put('session', { user: 'alice', role: 'admin' })

    // get returns null if the key doesn't exist
    const count = await hput.get('count')

    // list all keys
    const all = await hput.list()
    // all.keys   → ['count', 'session']
    // all.cursor → '' (no more pages)

    // list with prefix and pagination
    const page = await hput.list({ prefix: 'sess', limit: 100 })
    const next = await hput.list({ prefix: 'sess', limit: 100, cursor: page.cursor })

    await hput.delete('count')

    response.json({ count, all: all.keys })
})()
```

#### Counter example
PUT this to `/counter`:
```javascript
(async () => {
    const n = await hput.get('n')
    const count = n === null ? 1 : n + 1
    await hput.put('n', count)
    response.json({ count })
})()
```

Each request to `GET /counter` increments and returns the count.

## Projects that make this work
- https://github.com/tommie/v8go
- https://github.com/etcd-io/bbolt

## Projects that inspired this
- https://glitch.com/
- https://github.com/aol/micro-server
