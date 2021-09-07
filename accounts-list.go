package main

import (
	"fmt"

	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

func listAccounts(term ui.Screen, cmd *cobra.Command, args []string) error {
	if useTrezor || useLedger {
		wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
		if err != nil {
			return err
		}
		term.Logf("Found %d wallet(s)\n", len(wallets))
		for _, w := range wallets {
			term.Logf("Wallet status: %s\n", w.Status())
			if hdpath != "" {
				acc, err := hwwallet.Account(w, hdpath)
				if err != nil {
					return err
				}
				term.Logf("%s %s", acc.Address.Hex(), acc.URL.Path)
				break
			}
			accs, err := hwwallet.Accounts(w, hwwallet.DefaultHDPaths, max)
			if err != nil {
				return err
			}
			for _, acc := range accs {
				if flagVerbose {
					term.Output(fmt.Sprintf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path))
				} else {
					term.Output(fmt.Sprintf("%s\n", acc.Address.Hex()))
				}
			}
		}
	} else if keystorePath != "" {
		ks := keystore.NewKeyStore(term, keystorePath)
		accounts, err := ks.Accounts()
		if err != nil {
			return err
		}
		term.Logf("Found %d account(s)\n", len(accounts))
		for _, acc := range accounts {
			if flagVerbose {
				term.Output(fmt.Sprintf("%s path: %s\n", acc.Address, acc.URL.Path))
			} else {
				term.Output(fmt.Sprintf("%s\n", acc.Address))
			}
		}
	}
	return nil
}
