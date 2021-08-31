package wallet

import (
	"errors"

	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/ledger"
	"github.com/jaanek/jethwallet/trezor"
	"github.com/jaanek/jethwallet/ui"
)

const (
	KeyStore = iota
	Ledger
	Trezor
)

func GetHWWallets(ui ui.Screen, useTrezor, useLedger bool) ([]hwwallet.HWWallet, error) {
	var wallets []hwwallet.HWWallet
	var err error
	if useTrezor {
		wallets, err = trezor.Wallets(ui)
	} else if useLedger {
		wallets, err = ledger.Wallets(ui)
	} else {
		return nil, errors.New("Unsupported hw wallet type")
	}
	return wallets, err
}
