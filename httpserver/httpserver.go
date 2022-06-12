package httpserver

import (
	"context"
	"fmt"
	"hput"
	"net/http"
	"strings"
)

// Httpserver accepts http requests and responds to them.
type Httpserver struct {
	Port     int     // number of port to listen to. Required.
	Service  Service // Handler functions for activities performed
	Logger   Logger
	NonLocal bool // Reject any traffic that doesn't come from local traffic
	Locked   bool // Pass all requests to run and don't put any paths
}

// Logger logs out.
type Logger interface {
	Debugf(msg string, args ...interface{})
	Debug(msg string)
	Warnf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
}

// Service an engine to accept sanitized requests.
// It should save items into from Put and output
// results to the http.ResponseWriter on Run.
type Service interface {
	Put(ctx context.Context, r *http.Request) (*hput.PutResult, error)
	Run(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

// Serve starts the http server and it starts listening.
func (s *Httpserver) Serve() {
	s.Logger.Debugf("establishing handlers")
	http.HandleFunc("/", s.handle)
	s.Logger.Infof("serving at port %v", s.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", s.Port), nil); err != nil {
		s.Logger.Errorf("Could not serve because: %+v", err)
	}
}

// handle will accept a request and write outputs to the http.ResponseWriter
func (s *Httpserver) handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.Logger.Infof("request: %#v", r)
	// only allow local traffic unless NonLocal is allowed
	if !s.NonLocal {
		caller := strings.Split(r.RemoteAddr, ":")[0]
		if caller != "[" && caller != "localhost" && caller != "127.0.0.1" {
			s.Logger.Warnf("invalid caller: %s tried to call but was rejected because only local traffic allowed", r.RemoteAddr)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("This can only be called from local"))
			return
		}
	}
	s.Logger.Debugf("Handling request with method %s", r.Method)
	switch r.Method {
	case "OPTIONS":
		s.Logger.Debug("Handling OPTIONS request")
		s.options(w, r)
	case "PUT":
		s.Logger.Debug("Handling PUT request")
		if s.Locked {
			s.run(ctx, w, r)
			return
		}
		s.put(ctx, w, r)
	default:
		s.Logger.Debugf("Handling other request request")
		s.run(ctx, w, r)
	}
}

// options returns the allowed methods to every endpoint. Filling this
// in was required to allow xhr to PUT requests.
func (s *Httpserver) options(w http.ResponseWriter, r *http.Request) {
	vlsOrigin := r.Header["Origin"]
	if len(vlsOrigin) == 1 {
		w.Header().Add("Access-Control-Allow-Origin", vlsOrigin[0])
	}
	w.Header().Add("Access-Control-Allow-Methods", http.MethodPut)
	w.Header().Add("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Add("Access-Control-Max-Age", "1728000")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(http.StatusOK)
}

// put handles all put requests. It sanitizes them and passes them on to
// the Service.
func (s *Httpserver) put(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	s.Logger.Debugf("processing PUT request")
	putResult, err := s.Service.Put(ctx, r)
	if err != nil {
		s.Logger.Warnf("error PUT request, %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Unexpected input: %s", err.Error())))
		return
	}
	s.Logger.Debugf("processing PUT success")
	vlsOrigin := r.Header["Origin"]
	if len(vlsOrigin) == 1 {
		w.Header().Add("Access-Control-Allow-Origin", vlsOrigin[0])
	}
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(http.StatusAccepted)
	// Enhancement idea: convert to html with GET link
	w.Write([]byte(fmt.Sprintf("Saved input of type: %s", putResult.Input)))
}

// run handles any requests that aren't PUT or OPTIONS (for example, get)
// it sanitizes the request and passes it to the Service.
func (s *Httpserver) run(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	s.Logger.Debugf("processing RUN")
	if err := s.Service.Run(ctx, w, r); err != nil {
		s.Logger.Debugf("processing RUN error, %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error Unexpected error"))
	}
}
