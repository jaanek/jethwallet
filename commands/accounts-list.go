package commands

import (
	"fmt"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
)

func ListAccounts(term ui.Screen, flag *flags.Flags) error {
	if flag.UseTrezor || flag.UseLedger {
		wallets, err := wallet.GetHWWallets(term, flag.UseTrezor, flag.UseLedger)
		if err != nil {
			return err
		}
		term.Logf("Found %d wallet(s)\n", len(wallets))
		for _, w := range wallets {
			term.Logf("Wallet status: %s\n", w.Status())
			if flag.Hdpath != "" {
				acc, err := hwwallet.Account(w, flag.Hdpath)
				if err != nil {
					return err
				}
				term.Logf("%s %s", acc.Address.Hex(), acc.URL.Path)
				break
			}
			accs, err := hwwallet.Accounts(w, hwwallet.DefaultHDPaths, flag.Max)
			if err != nil {
				return err
			}
			for _, acc := range accs {
				if flag.FlagVerbose {
					term.Output(fmt.Sprintf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path))
				} else {
					term.Output(fmt.Sprintf("%s\n", acc.Address.Hex()))
				}
			}
		}
	} else if flag.KeystorePath != "" {
		ks := keystore.NewKeyStore(term, flag.KeystorePath)
		accounts, err := ks.Accounts()
		if err != nil {
			return err
		}
		term.Logf("Found %d account(s)\n", len(accounts))
		for _, acc := range accounts {
			if flag.FlagVerbose {
				term.Output(fmt.Sprintf("%s path: %s\n", acc.Address, acc.URL.Path))
			} else {
				term.Output(fmt.Sprintf("%s\n", acc.Address))
			}
		}
	}
	return nil
}
