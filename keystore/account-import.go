package keystore

import (
	"errors"
	"fmt"

	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/crypto"
)

func ImportKey(term ui.Screen, keystorePath string) error {
	if keystorePath == "" {
		return errors.New("keystore path required")
	}
	ks := NewKeyStore(term, keystorePath)
	term.Print("*** Enter private key as 64 hexadecimal digits (not echoed): ")
	keyBytes, err := term.ReadPassword()
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}
	privatekey, err := crypto.HexToECDSA(string(keyBytes))
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}
	term.Print("*** Choose a passphrase for the account (not echoed): ")
	passphrase, err := term.ReadPassword()
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}
	acc, err := ks.ImportECDSA(privatekey, string(passphrase))
	if err != nil {
		return fmt.Errorf("failed to import private key to keystore: %w", err)
	}
	term.Logf("New account created! Address: %s, path: %v\n", acc.Address, acc.URL.Path)
	return nil
}
