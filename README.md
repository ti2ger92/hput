# hput
A standalone http server that you can program easily via http without knowing linux.
- http `PUT` web pages, text, pictures 🖼️ 🎵🎞️ or any file, then request `GET`, `POST`, `PATCH`, or `DELETE` to the same path path to get them back ✨.
- http `PUT` javascript to a path then call that path with any other verb to run your code
  - The last line of your javascript is returned as payload.
  - node-express style `request` and `response` objects can be used to run web programs.

## Start

### Launch with Go
You can launch the service directly with Go

#### Prerequisite:
Go v1.18

#### Start:
```
go mod download # gathers all dependencies
go run main/main.go -port 80 # runs the program as an http server, 80 is the default port number
```

## Start flags
| flag | description | default | other options |
| - | - | - | - |
| `-port` | which port to listen on | `80` | any int |
| `-nonlocal` | whether to allow traffic from a source other than localhost | `false` | `true` |
| `-storage` | which storage to use, currently supported: local disc and memory, which will be deleted whenever the server is shut down | `local` | `memory`, `s3` |
| `-log` | which level to set logging at | `info` | `debug`, `warn`, `error`|
| `-filename` | if using local storage, name of the database file to create and use. | `hput.db` | any valid file path |
| `-locked` | do not accept any http `PUT` commands to write, pass all commands to runnables | `false` | `true` |
| `-bucket` | if using s3, bucket name to read and write items to | N/A | any valid bucket |
| `-prefix` | If using S3, a prefix (folder) to use with object keys | N/A | any |

#### Stop:
hold `control` and hit `c` on your keyboard

### Launch with docker-compose
#### Prerequisite:
Docker/Docker desktop and Docker Compose (usually comes installed with Docker)

Either a local version of all containers needed, or a docker login
```
docker login
```

#### Start:
```
docker-compose up -d
```

#### Stop:
```
docker-compose down
```

## Use
You can send a PUT request to your server to save something.

Here is some Javascript that you can copy/paste. You can run javascript in browsers like Chrome [Chrome instructions](https://developer.chrome.com/docs/devtools/console/javascript/).
```
var xhr = new XMLHttpRequest()
xhr.open("PUT", "http://localhost/test")
xhr.send("Hello hput")
```

Then, in your browser, input the URL: `http://localhost/test`

### Save your work
You can dump any subpath by a call to your <path>`/dump` to get code which will recreate your work on another hput server. Using the above as an example, visit `http://localhost/test/dump` or `http://localhost/dump` to output the same result, then you can paste that result into any javascript console to reprogram the same thing.

### Example payloads
Send any of these as an HTTP PUT payload, and see how the server responds
#### html loads as a webpage
basic hello world:
```
<html>
<h1>hello hput</h1>
here is some html
```
html with in browser javascript:
```
<html>
<h1>hello html</h1>
<script>
    document.write(`The date on your computer is ${new Date()}`);
</script>
```

#### server side javascript
Get the time 24 hours from now:
```
`tomorrow it will be: ${new Date(new Date().getTime() + 1*24*60*60*1000)}`
```


#### Combine both
1. Return the date, try putting this at `/time`
```
new Date()
```

2. Try putting this at `index.html`. If you've setup the server side javascript, you'll see both responses
```
<html>
<h1>browser and server</h1>
<div id="browser"></div><br>
<div id="server"></div><br>
The times should be the same. The first was calculated by your browser, the second by the hput server.
<script>
    document.getElementById("browser").innerHTML = `The date on your browser is ${new Date()}`
    fetch("/time")
      .then(response => response.text())
      .then((response) => {
        document.getElementById("server").innerHTML = `The date on your server is ${response}`
      })
</script>
```

## Javascript code
Thanks to [v8go](https://github.com/rogchap/v8go), you can http `PUT` Javascript to your server.
1. You can state a value as the last line of your program to return it via http.
2. You can use `console.log` to log something at level `INFO`.
3. You'll find a global variable with the name `request` with these elements from node-express v4 ([express api](https://expressjs.com/en/api.html#req)):

| element | type | description |
| - | - | - |
| get | function | retrieves a header from the request |
| cookies | object | describes the cookies in the request |
| hostname | string | name of the host making the request |
| ip | string | ip address making the request |
| method | string | http verb, like GET or POST |
| path | string | path of the request, or the url after the domain |
| protocol | string | whether it's http or https |
| query | object | map of query elements sent in the request |

4. You'll find a global variable with the name `response` with these elements from node-express v4 ([express api](https://expressjs.com/en/api.html#res)):

| element | type | description |
| - | - | - |
| append | function | adds a header, or if the header already exists, adds another value to the header |
| cookie | function | adds a cookie |
| json | function | writes out json in the response |
| location | function | adds a value to the Location header |
| redirect | function | responds with an http redirect |
| send | function | currently the same as json, will be expanded to auto-detect type and respond with correct type |
| sendStatus | function | sends an http status |
| set | function | overwrites a header with a value. You can make that value `''` to delete it |
| status | function | sets the http status |

5. For example, you can set a path with PUT like this: `response.json(request)`, then visit it in your browser to get a pretty good look at the request object.

## Potential use cases
- Mock server
- Database with functional capabilities
- Rapid prototyping
- Hackathons
- Education
- Revolutionize the planet with your awesome idea

## Projects that inspired me
I haven't connected with authors of these projects and they don't endorse this project, I just dig their ideas.
- https://glitch.com/
- https://github.com/aol/micro-server
- https://popcode.org/

## More about the project
What would a web server that is as simple to program as it is to visit look like? It would start with zero config, then accept text, json, files, or code via simple http calls. You'd program this http server the same way you use it: via http commands.

Hput limits the activities when programming for the web to http requests and responses. This way a developer doesn't need to know about the operating system, nor access the system directly to build on the server. Components of the server can be tried in real time as they are added, and developers can troubleshoot via the web. It's a server built for the cloud from the ground up.

Web projects should be easy to start and keep the focus on your idea. Instead, web creators often have to learn to work with git, linux, apache, application servers, and language-specific frameworks before they ever visit `index.html`. Even cloud services and FaaS frameworks have their own concepts to master before you can start a project, and these become barriers of entry for people who want to create for the web that don't have much to do with creating for the web.

## Fast follow features 0.2:
1. Ability to force a type of input
1. Add logs http output


## Roadmap:
1. Javascript calls back to the same hput are handled like function calls, without any actual http call
1. Unit Tests