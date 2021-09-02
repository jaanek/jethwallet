package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

func signMsg(term ui.Screen, cmd *cobra.Command, args []string) error {
	if flagFrom == "" {
		return errors.New("Missing --from address")
	}
	if flagInput == "" {
		return errors.New("Missing --data")
	}
	fromAddr := common.HexToAddress(flagFrom)
	data := []byte(flagInput)
	if strings.HasPrefix(flagInput, "0x") {
		data = hexutil.MustDecode(flagInput)
	}
	var msg []byte = data
	if flagAddEthPrefix {
		msg = []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data))
	}

	// sign message
	var signature []byte
	if useTrezor || useLedger {
		wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
		if err != nil {
			return err
		}
		hww, acc, _ := hwwallet.FindOneFromWallets(term, wallets, fromAddr, hwwallet.DefaultHDPaths, max)
		if acc == (accounts.Account{}) {
			return errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
		}
		term.Logf("Found account: %v, path: %s ...", acc.Address, acc.URL.Path)
		path, err := accounts.ParseDerivationPath(acc.URL.Path)
		if err != nil {
			return err
		}
		addr, sig, err := hww.SignMessage(path, msg)
		if err != nil {
			return err
		}
		if addr != acc.Address {
			return errors.New("Signed message sender address != provided derivation path address!")
		}
		signature = sig
	} else if keystorePath != "" {
		ks := keystore.NewKeyStore(term, keystorePath)

		// find the account by address
		acc, err := ks.FindOne(fromAddr)
		if err != nil {
			return err
		}
		term.Print(fmt.Sprintf("*** Enter passphrase (not echoed) account: %v ...", acc.Address))
		passphrase, err := term.ReadPassword()
		if err != nil {
			return err
		}
		key, err := ks.GetDecryptedKey(acc, string(passphrase))
		if err != nil {
			return err
		}
		defer keystore.ZeroKey(key.PrivateKey)
		sig, err := ks.SignData(key.PrivateKey, msg)
		if err != nil {
			return err
		}
		sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
		signature = sig
	}
	term.Output(hexutil.Encode(signature[:]))
	return nil
}
