package main

import (
	"errors"

	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/spf13/cobra"
)

func newAccount(term ui.Screen, cmd *cobra.Command, args []string) error {
	if keystorePath == "" {
		return errors.New("Only supports creating new accounts in keystore!")
	}
	ks := keystore.NewKeyStore(term, keystorePath)
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
