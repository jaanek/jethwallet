package main

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

func signTx(term ui.Screen, cmd *cobra.Command, args []string) error {
	// validate flags
	if flagNonce == "" {
		return errors.New("Missing --nonce")
	}
	if flagFrom == "" {
		return errors.New("Missing --from address")
	}
	var to *common.Address
	if flagTo != "" {
		t := common.HexToAddress(flagTo)
		to = &t
	}
	if flagGasLimit == "" {
		return errors.New("Missing --gas-limit")
	}
	nonce := math.MustParseUint64(flagNonce)
	fromAddr := common.HexToAddress(flagFrom)
	gasLimit := math.MustParseUint64(flagGasLimit)
	var gasPrice, gasTipCap, gasFeeCap *big.Int
	if flagGasPrice != "" {
		gasPrice = math.MustParseBig256(flagGasPrice)
	}
	if flagGasTip != "" {
		gasTipCap = math.MustParseBig256(flagGasTip)
	}
	if flagGasFeeCap != "" {
		gasFeeCap = math.MustParseBig256(flagGasFeeCap)
	}
	if gasPrice == nil && (gasTipCap == nil || gasFeeCap == nil) {
		return errors.New("Either --gas-price or (--gas-tip and --gas-maxfee) must be provided")
	}
	var value *big.Int
	if flagValue != "" {
		value = math.MustParseBig256(flagValue)
	} else {
		value = new(big.Int)
	}
	if flagChainID == "" {
		return errors.New("Missing --chain-id")
	}
	chainID := math.MustParseBig256(flagChainID)
	if to == nil && flagInput == "" {
		return errors.New("Either --to or --input must be provided")
	}
	var input = []byte{}
	if flagInput != "" {
		input = hexutil.MustDecode(flagInput)
	}

	// Create the transaction to sign
	var rawTx *types.Transaction
	if gasPrice != nil {
		baseTx := &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			Value:    value,
			Data:     input,
		}
		if to != nil {
			baseTx.To = to
		}
		rawTx = types.NewTx(baseTx)
	} else {
		baseTx := &types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			Value:     value,
			Data:      input,
		}
		if to != nil {
			baseTx.To = to
		}
		rawTx = types.NewTx(baseTx)
	}

	// sign tx
	var (
		signed *types.Transaction
	)
	if useTrezor || useLedger {
		wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
		if err != nil {
			return err
		}
		var acc accounts.Account
		var hww hwwallet.HWWallet
		for _, w := range wallets {
			acc, err = hwwallet.FindOne(w, fromAddr, hwwallet.DefaultHDPaths, max)
			if err != nil {
				// log out that we did not found from wallet or there was multiple
				term.Print(err.Error())
				continue
			}
			hww = w
			break
		}
		if acc == (accounts.Account{}) {
			return errors.New(fmt.Sprintf("No accounts found for address: %s\n", fromAddr))
		}
		term.Print(fmt.Sprintf("Found account: %v, path: %s ...", acc.Address, acc.URL.Path))
		path, err := accounts.ParseDerivationPath(acc.URL.Path)
		if err != nil {
			return err
		}
		var addr common.Address
		addr, signed, err = hww.SignTx(path, rawTx, *chainID)
		if err != nil {
			return err
		}
		if addr != acc.Address {
			return errors.New("Signed tx sender address != provided derivation path address!")
		}
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
		signed, err = ks.SignTx(key.PrivateKey, rawTx, chainID)
		if err != nil {
			return err
		}
	}
	if flagSig {
		v, r, s := signed.RawSignatureValues()
		term.Output(fmt.Sprintf("0x%064x%064x%02x", r, s, v))
	} else {
		encoded, _ := rlp.EncodeToBytes(signed)
		term.Output(hexutil.Encode(encoded[:]))
	}
	return nil
}
