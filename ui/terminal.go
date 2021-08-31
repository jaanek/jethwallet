package ui

import (
	"fmt"
	"os"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts"
	"golang.org/x/crypto/ssh/terminal"
)

type Screen interface {
	ReadPassword(acc *accounts.Account) ([]byte, error)
	Print(msg string)
}

type term struct{}

func NewTerminal() Screen {
	return &term{}
}

func (t *term) ReadPassword(acc *accounts.Account) ([]byte, error) {
	return terminal.ReadPassword(syscall.Stdin)
}

func (t *term) Print(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}
