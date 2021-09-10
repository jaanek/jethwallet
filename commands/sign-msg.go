package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
)

func SignMsg(term ui.Screen, flag *flags.Flags) error {
	if flag.FlagFrom == "" {
		return errors.New("Missing --from address")
	}
	if flag.FlagInput == "" {
		return errors.New("Missing --data")
	}
	fromAddr := common.HexToAddress(flag.FlagFrom)
	data := []byte(flag.FlagInput)
	if strings.HasPrefix(flag.FlagInput, "0x") {
		data = hexutil.MustDecode(flag.FlagInput)
	}
	var msg []byte = data
	if flag.FlagAddEthPrefix {
		msg = []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data))
	}

	// sign message
	var signature []byte
	if flag.UseTrezor || flag.UseLedger {
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
		addr, sig, err := hww.SignMessage(path, msg)
		if err != nil {
			return err
		}
		if addr != acc.Address {
			return errors.New("Signed message sender address != provided derivation path address!")
		}
		signature = sig
	} else if flag.KeystorePath != "" {
		ks := keystore.NewKeyStore(term, flag.KeystorePath)

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
