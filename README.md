# hput
A standalone http server that you can program easily via http without knowing linux.

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
// Boiler plate code to setup a request in javascript
var xhr = new XMLHttpRequest();
xhr.withCredentials = true;

xhr.addEventListener("readystatechange", function() {
  if(this.readyState === 4) {
    console.log(this.responseText);
  }
});

xhr.open("PUT", "http://localhost/test");
xhr.setRequestHeader("Content-Type", "text/plain");

xhr.send("Hello hput");
```

Then, in your browser, input the URL: `http://localhost/test`

## Start flags
| flag | description | default | other options |
| - | - | - | - |
| `-port` | which port to listen on | `80` | any int |
| `-nonlocal` | whether to allow traffic from a source other than localhost | `false` | `true` |
| `-storage` | which storage to use, currently supported: local disc and memory, which will be deleted whenever the server is shut down | `local` | `memory` |
| `-filename` | if using local storage, name of the database file to create and use. | `hput.db` | any valid file path |


## Potential use cases
- Mock server
- Database with functional capabilities
- Rapid prototyping
- Hackathons

## Projects that inspired me
I haven't connected with authors of these projects and they don't endorse this project, I just dig their ideas.
- https://glitch.com/
- https://github.com/aol/micro-server
- https://popcode.org/

## More about the project
What would it look like if an http application server grew with your abilities? It would start with zero config, then you could add static text, json, files or logical programs with single http calls. You'd program this http server the same way you use it: via http commands.

Web projects should be easy to start and keep the focus on your idea. Instead, web creators often have to learn to work with git, linux, apache, application servers, and language-specific frameworks before their server starts. Even cloud services and FaaS frameworks have their own concepts to master before you can start a project.

Unlike a web UI, HTTP programming protocols can scale to production needs, with an ability to backup, store, and redeploy functionality and data. Therefore an interesting addition would be a server that's simple to start, but doesn't require a web UI.

## Fast follow features 0.1:
1. S3 saver
1. Output whatever you replaced if you replace something
1. Ability to force a type of input
1. Add logs http output