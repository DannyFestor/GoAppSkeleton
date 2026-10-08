package main

import "io"

// environment is everything run takes from the process, so tests can supply their own.
type environment struct {
	stdout io.Writer
	stderr io.Writer
	getenv func(string) string
}
