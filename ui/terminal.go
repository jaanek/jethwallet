package ui

import (
	"fmt"
	"os"
	"syscall"

	"github.com/pkg/errors"
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
	verbose bool
}

func NewTerminal(verbose bool) Screen {
	return &term{verbose: verbose}
}

func (t *term) ReadPassword() ([]byte, error) {
	// return terminal.ReadPassword(syscall.Stdin)
	return readPassword()
}

func (t *term) Print(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func (t *term) Output(msg string) {
	fmt.Fprint(os.Stdout, msg)
}

func (t *term) Logf(msg string, args ...interface{}) {
	if t.verbose {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(msg, args...))
	}
}

func (t *term) Log(msg interface{}) {
	if t.verbose {
		fmt.Fprintf(os.Stderr, "%v\n", msg)
	}
}

func (t *term) Errorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(msg, args...))
}

func (t *term) Error(msg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}

func readPassword() ([]byte, error) {
	var fd int
	if terminal.IsTerminal(syscall.Stdin) {
		fd = syscall.Stdin
	} else {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return nil, errors.Wrap(err, "error allocating terminal")
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}
	pass, err := terminal.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	return pass, err
}
