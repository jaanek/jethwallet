package ui

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type Screen interface {
	ReadPassword() ([]byte, error)
	Print(msg string)
	Output(msg string)
	Log(msg interface{})
	Logf(msg string, args ...interface{})
	Error(msg interface{})
	Errorf(msg string, args ...interface{})
}

type term struct {
	quiet bool
}

func NewTerminal(quiet bool) Screen {
	return &term{quiet: quiet}
}

func (t *term) ReadPassword() ([]byte, error) {
	return terminal.ReadPassword(syscall.Stdin)
}

func (t *term) Print(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func (t *term) Output(msg string) {
	fmt.Fprint(os.Stdout, msg)
}

func (t *term) Logf(msg string, args ...interface{}) {
	if t.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, fmt.Sprintf(msg, args...))
}

func (t *term) Log(msg interface{}) {
	if t.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}

func (t *term) Errorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(msg, args...))
}

func (t *term) Error(msg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}
