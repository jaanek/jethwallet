package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

type Output struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	RawTransaction string `json:"tx"`
	TransactionSig string `json:"txsig"`
}

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
		var ok bool
		value, ok = math.ParseBig256(flagValue)
		if !ok {
			return errors.New(fmt.Sprintf("invalid 256 bit integer: " + flagValue))
		}
		if flagValueEth {
			value = new(big.Int).Mul(value, new(big.Int).SetInt64(params.Ether))
		} else if flagValueGwei {
			value = new(big.Int).Mul(value, new(big.Int).SetInt64(params.GWei))
		}
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
		hww, acc, _ := hwwallet.FindOneFromWallets(term, wallets, fromAddr, hwwallet.DefaultHDPaths, max)
		if acc == (accounts.Account{}) {
			return errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
		}
		term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
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
		defer keystore.ZeroKey(key.PrivateKey)
		signed, err = ks.SignTx(key.PrivateKey, rawTx, chainID)
		if err != nil {
			return err
		}
	}

	// output
	encoded, _ := rlp.EncodeToBytes(signed)
	encodedRawTx := hexutil.Encode(encoded[:])
	v, r, s := signed.RawSignatureValues()
	txSig := fmt.Sprintf("0x%064x%064x%02x", r, s, v)
	out := Output{
		RpcUrl:         flagRpcUrl,
		ChainId:        flagChainID,
		RawTransaction: encodedRawTx,
		TransactionSig: txSig,
	}
	outb, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(outb)))
	// if flagSig {
	// 	v, r, s := signed.RawSignatureValues()
	// 	term.Output(fmt.Sprintf("0x%064x%064x%02x", r, s, v))
	// } else {
	// 	encoded, _ := rlp.EncodeToBytes(signed)
	// 	term.Output(hexutil.Encode(encoded[:]))
	// }
	return nil
}
