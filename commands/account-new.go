package commands

import (
	"errors"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
)

func NewAccount(term ui.Screen, flag *flags.Flags) error {
	if flag.KeystorePath == "" {
		return errors.New("Only supports creating new accounts in keystore!")
	}
	ks := keystore.NewKeyStore(term, flag.KeystorePath)
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
