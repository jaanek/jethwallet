package main

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/spf13/cobra"
)

func signTx(cmd *cobra.Command, args []string) error {
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
	if flagGasTipCap != "" {
		gasTipCap = math.MustParseBig256(flagGasTipCap)
	}
	if flagGasFeeCap != "" {
		gasFeeCap = math.MustParseBig256(flagGasFeeCap)
	}
	if gasPrice == nil && (gasTipCap == nil || gasFeeCap == nil) {
		return errors.New("Either --gas-price or (--gas-tip and --gas-maxfee) must be provided")
	}
	// if gasTipCap != nil && gasFeeCap == nil {
	// 	gasFeeCap = new(big.Int).Add(gasTipCap, new(big.Int).Mul(head.BaseFee, big.NewInt(2)))
	// }
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

	// Create the transaction to sign
	var rawTx *types.Transaction
	if gasPrice != nil {
		baseTx := &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			Value:    value,
			// Data:     input,
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
			// Data:      input,
		}
		if to != nil {
			baseTx.To = to
		}
		rawTx = types.NewTx(baseTx)
	}

	// sign tx
	term := ui.NewTerminal()
	if useTrezor || useLedger {
		// wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
		// if err != nil {
		// 	return err
		// }
	} else if keystorePath != "" {
		ks := keystore.NewKeyStore(term, keystorePath)

		// find the account by address
		acc, err := ks.FindOne(fromAddr)
		if err != nil {
			return err
		}
		term.Print(fmt.Sprintf("*** Enter account: %v passphrase (not echoed) ...", acc.Address))
		passphrase, err := term.ReadPassword()
		if err != nil {
			return err
		}
		key, err := ks.GetDecryptedKey(acc, string(passphrase))
		if err != nil {
			return err
		}
		signed, err := ks.SignTx(key.PrivateKey, rawTx, chainID)
		if err != nil {
			return err
		}
		if flagSig {
			v, r, s := signed.RawSignatureValues()
			fmt.Println(fmt.Sprintf("0x%064x%064x%02x", r, s, v))
		} else {
			encoded, _ := rlp.EncodeToBytes(signed)
			fmt.Println(hexutil.Encode(encoded[:]))
		}
	}
	return nil
}
