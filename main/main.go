package main

import (
	"flag"
	"fmt"
	"hput"
	"hput/discsaver"
	"hput/httpserver"
	"hput/javascript"
	"hput/logger"
	"hput/mapsaver"
	"hput/service"
	"net/url"
)

// Saver Saves stateful data for the service
type Saver interface {
	SaveText(s string, p url.URL, r *hput.PutResult) error
	GetRunnable(p url.URL) (hput.Runnable, error)
	SendRunnables(p string, runnables chan<- hput.Runnable, done chan<- bool) error
	SaveCode(s string, p url.URL, r *hput.PutResult) error
	SaveBinary(b []byte, p url.URL, r *hput.PutResult) error
}

func main() {
	l, err := logger.New()
	if err != nil {
		fmt.Printf("Unable to initialize logger, stopping, %+v", err)
	}

	portPtr := flag.Int("port", 80, "an int")
	allTrafficPtr := flag.Bool("nonlocal", false, "allow traffic which is not local")
	storagePtr := flag.String("storage", "local", "which storage to use, currently supported: local and memory")
	fileNamePtr := flag.String("filename", "hput.db", "if using local storage, name of the database file to create and use")
	flag.Parse()
	var saver Saver
	switch *storagePtr {
	case "local":
		saver, err = discsaver.New(&l, *fileNamePtr)
		if err != nil {
			l.Errorf("main.Main(): could not initialize discsaver: %v", err)
			return
		}
	case "memory":
		saver = &mapsaver.MapSaver{
			Logger: &l,
		}
	default:
		l.Errorf("main.Main(): incorrect storage parameter passed, use 'local' or 'memory'")
	}
	l.Debug("Initialized map saver")
	js := javascript.New(&l)
	l.Debug("Initialized javascript module")
	s := service.Service{
		Interpreter: &js,
		Saver:       saver,
		Logger:      &l,
	}
	l.Debug("Initialized service module")
	h := httpserver.Httpserver{
		Port:     *portPtr,
		Service:  &s,
		Logger:   &l,
		NonLocal: *allTrafficPtr,
	}
	if *allTrafficPtr {
		l.Debug("Allowing nonlocal traffic")
	}
	l.Debug("Initialized http server")
	h.Serve()
}
