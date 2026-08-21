// Package procrun runs external commands under a controlled environment.
//
// Layer:   Platform
// Concern: Give a child process exactly the environment it should have —
// nothing inherited by accident — and keep its output out of a pipe.
package procrun
