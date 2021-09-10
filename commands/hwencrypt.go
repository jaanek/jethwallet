package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jaanek/jethwallet/accounts"
	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
)

func HwEncrypt(term ui.Screen, flag *flags.Flags) error {
	if flag.FlagFrom == "" {
		return errors.New("Missing --from address")
	}
	if flag.FlagKey == "" {
		return errors.New("Missing --key")
	}
	if flag.FlagInput == "" {
		return errors.New("Missing --data")
	}
	fromAddr := common.HexToAddress(flag.FlagFrom)
	key := []byte(flag.FlagKey)
	if strings.HasPrefix(flag.FlagKey, "0x") {
		key = hexutil.MustDecode(flag.FlagKey)
	}
	data := []byte(flag.FlagInput)
	if strings.HasPrefix(flag.FlagInput, "0x") {
		data = hexutil.MustDecode(flag.FlagInput)
	}

	// encrypt data
	if !flag.UseTrezor {
		return errors.New("Only support for trezor now!")
	}
	wallets, err := wallet.GetHWWallets(term, flag.UseTrezor, flag.UseLedger)
	if err != nil {
		return err
	}
	hww, acc, _ := hwwallet.FindOneFromWallets(term, wallets, fromAddr, hwwallet.DefaultHDPaths, flag.Max)
	if acc == (accounts.Account{}) {
		return errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
	}
	term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
	path, err := accounts.ParseDerivationPath(acc.URL.Path)
	if err != nil {
		return err
	}
	encrypted, err := hww.Encrypt(path, string(key), data, true, true)
	if err != nil {
		return err
	}
	term.Output(hexutil.Encode(encrypted[:]))
	return nil
}
