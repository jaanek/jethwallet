package keystore

import (
	"errors"

	"github.com/jaanek/jethwallet/ui"
)

func NewAccount(term ui.Screen, keystorePath string) error {
	if keystorePath == "" {
		return errors.New("Only supports creating new accounts in keystore!")
	}
	ks := NewKeyStore(term, keystorePath)
	term.Print("*** Enter passphrase (not echoed)...")
	passphrase, err := term.ReadPassword()
	if err != nil {
		return err
	}
	acc, err := ks.NewAccount(string(passphrase))
	if err != nil {
		return err
	}
	term.Logf("New account created! Address: %s, path: %v\n", acc.Address, acc.URL.Path)
	return nil
}
