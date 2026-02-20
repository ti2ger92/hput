package main

import (
	"context"
	"flag"
	"fmt"
	"hput/discsaver"
	"hput/httpserver"
	"hput/javascript"
	"hput/logger"
	"hput/mapsaver"
	"hput/s3saver"
	"hput/service"
)

func main() {
	ctx := context.Background()
	portPtr := flag.Int("port", 80, "an int")
	allTrafficPtr := flag.Bool("nonlocal", false, "allow traffic which is not local")
	storagePtr := flag.String("storage", "local", "which storage to use, currently supported: local and memory")
	fileNamePtr := flag.String("filename", "hput.db", "if using local storage, name of the database file to create and use")
	lockedPtr := flag.Bool("locked", false, "pass all requests to run, do not store any paths")
	logLvlPtr := flag.String("log", "info", "which log level to use, options are: debug, info, warn, error")
	bucketPtr := flag.String("bucket", "", "if using s3 storage, the bucket to use")
	prefixPtr := flag.String("prefix", "", "if using s3 storage, the prefix to use")
	flag.Parse()

	l, err := logger.New(*logLvlPtr)
	if err != nil {
		fmt.Printf("Unable to initialize logger, stopping, %+v", err)
	}

	var saver service.Saver
	switch *storagePtr {
	case "local":
		saver, err = discsaver.New(&l, *fileNamePtr)
		if err != nil {
			l.Errorf("main.Main(): could not initialize discsaver: %v", err)
			return
		}
		l.Debug("Initialized local saver")
	case "memory":
		saver = &mapsaver.MapSaver{
			Logger: &l,
		}
		l.Debug("Initialized map saver")
	case "s3":
		saver, err = s3saver.New(ctx, &l, *bucketPtr, *&s3saver.PrefixOption{Prefix: *prefixPtr})
		if err != nil {
			l.Errorf("Unable to initialize s3saver: %v", err)
		}
	default:
		l.Errorf("main.Main(): incorrect storage parameter passed, use 'local' or 'memory'")
	}
	js, err := javascript.New(&l)
	if err != nil {
		l.Errorf("Unable to initialize Javascript: %v", err)
		return
	}
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
		Locked:   *lockedPtr,
	}
	if *allTrafficPtr {
		l.Debug("Allowing nonlocal traffic")
	}
	l.Debug("Initialized http server")
	h.Serve()
}
