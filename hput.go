// Package hput describes a web server that can be programmed via http PUT
package hput

import (
	"errors"
)

var (
	ErrCannotReadPostPayload = errors.New("cannot read POST Payload")
)

type Input string

const (
	Text   Input = "Text"
	Js           = "Javascript"
	Binary       = "Binary"
)

// PutResult shares the result of a save
type PutResult struct {
	Input     Input // type of input passed to the function
	Overwrote bool
	Message   string
}

// Runnable describes a path that can be run
type Runnable struct {
	Path   string // exact location of resource on this server
	Type   Input  // details specific type of runnable
	Text   string // to be returned to the runner
	Binary []byte // raw bytes
}
