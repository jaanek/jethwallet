package keystore

import (
	"fmt"

	"github.com/jaanek/jethwallet/ui"
)

func ListAccounts(term ui.Screen, keystorePath string, verbose bool) error {
	ks := NewKeyStore(term, keystorePath)
	accounts, err := ks.Accounts()
	if err != nil {
		return err
	}
	term.Logf("Found %d account(s)\n", len(accounts))
	for _, acc := range accounts {
		if verbose {
			term.Output(fmt.Sprintf("%s path: %s\n", acc.Address, acc.URL.Path))
		} else {
			term.Output(fmt.Sprintf("%s\n", acc.Address))
		}
	}
	return nil
}
