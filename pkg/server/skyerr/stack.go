// This source is obtained from https://github.com/pkg/errors.
package skyerr

import (
	"runtime"

	"github.com/pkg/errors"
)

type stack []uintptr

func (s *stack) StackTrace() errors.StackTrace {
	f := make([]errors.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = errors.Frame((*s)[i])
	}
	return f
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}
