// Package hput describes a web server that can be programmed via http PUT
package hput

import (
	"errors"
)

var (
	// ErrCannotReadPostPayload A payload was sent via POST, but it cannot be read
	ErrCannotReadPostPayload = errors.New("cannot read POST Payload")
)

// Input describes the type of input which was sent or retrieved
type Input string

const (
	// Text means the input was plain text
	Text Input = "Text"
	// Js means the input can be compiled as Javascript
	Js = "Javascript"
	// Binary means the input is not text-like, for example an image
	Binary = "Binary"
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
