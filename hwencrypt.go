package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

func hwEncrypt(term ui.Screen, cmd *cobra.Command, args []string) error {
	if flagFrom == "" {
		return errors.New("Missing --from address")
	}
	if flagKey == "" {
		return errors.New("Missing --key")
	}
	if flagInput == "" {
		return errors.New("Missing --data")
	}
	fromAddr := common.HexToAddress(flagFrom)
	key := []byte(flagKey)
	if strings.HasPrefix(flagKey, "0x") {
		key = hexutil.MustDecode(flagKey)
	}
	data := []byte(flagInput)
	if strings.HasPrefix(flagInput, "0x") {
		data = hexutil.MustDecode(flagInput)
	}

	// encrypt data
	if !useTrezor {
		return errors.New("Only support for trezor now!")
	}
	wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
	if err != nil {
		return err
	}
	hww, acc, _ := hwwallet.FindOneFromWallets(term, wallets, fromAddr, hwwallet.DefaultHDPaths, max)
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
