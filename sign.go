package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/hwwallet/hwcommon"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

type Output struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	RawTransaction string `json:"tx"`
	TransactionSig string `json:"txsig"`
}

func SignTx(term ui.Screen, flag *flags.Flags) error {
	// validate flags
	if flag.FlagNonce == "" {
		return errors.New("Missing --nonce")
	}
	if flag.FlagFrom == "" {
		return errors.New("Missing --from address")
	}
	var to *common.Address
	if flag.FlagTo != "" {
		t := common.HexToAddress(flag.FlagTo)
		to = &t
	}
	if flag.FlagGasLimit == "" {
		return errors.New("Missing --gas-limit")
	}
	nonce := math.MustParseUint64(flag.FlagNonce)
	fromAddr := common.HexToAddress(flag.FlagFrom)
	gasLimit := math.MustParseUint64(flag.FlagGasLimit)
	var gasPrice, gasTipCap, gasFeeCap *uint256.Int
	if flag.FlagGasPrice != "" {
		var err error
		gasPrice, err = uint256.FromHex(flag.FlagGasPrice)
		if err != nil {
			return err
		}
	}
	if flag.FlagGasTip != "" {
		var err error
		gasTipCap, err = uint256.FromHex(flag.FlagGasTip)
		if err != nil {
			return err
		}
	}
	if flag.FlagGasFeeCap != "" {
		var err error
		gasFeeCap, err = uint256.FromHex(flag.FlagGasFeeCap)
		if err != nil {
			return err
		}
	}
	if gasPrice == nil && (gasTipCap == nil || gasFeeCap == nil) {
		return errors.New("Either --gas-price or (--gas-tip and --gas-maxfee) must be provided")
	}
	var value *uint256.Int
	if flag.FlagValue != "" {
		var err error
		value, err = uint256.FromHex(flag.FlagValue)
		if err != nil {
			return errors.New(fmt.Sprintf("invalid 256 bit integer: " + flag.FlagValue))
		}
		if flag.FlagValueEth {
			value = new(uint256.Int).Mul(value, new(uint256.Int).SetUint64(params.Ether))
		} else if flag.FlagValueGwei {
			value = new(uint256.Int).Mul(value, new(uint256.Int).SetUint64(params.GWei))
		}
	} else {
		value = new(uint256.Int)
	}
	if flag.FlagChainID == "" {
		return errors.New("Missing --chain-id")
	}
	chainID, err := uint256.FromHex(flag.FlagChainID)
	if err != nil {
		return err
	}
	if to == nil && flag.FlagInput == "" {
		return errors.New("Either --to or --input must be provided")
	}
	var input = []byte{}
	if flag.FlagInput != "" {
		input = hexutil.MustDecode(flag.FlagInput)
	}

	// Create the transaction to sign
	tx, err := wallet.NewTransaction(*chainID, nonce, to, value, input, gasLimit, gasPrice, gasTipCap, gasFeeCap)

	// sign tx
	var signed types.Transaction
	if flag.KeystorePath != "" {
		signed, err = keystore.SignTx(term, flag.KeystorePath, fromAddr, tx)
	} else {
		hwWalletType := hwcommon.GetWalletTypeFromFlags(flag)
		signed, err = hwwallet.SignTx(term, hwWalletType, fromAddr, tx, flag.Max)
	}

	// output
	var encoded bytes.Buffer
	err = signed.MarshalBinary(&encoded)
	if err != nil {
		return err
	}
	encodedRawTx := hexutil.Encode(encoded.Bytes()[:])
	v, r, s := signed.RawSignatureValues()
	txSig := fmt.Sprintf("0x%064x%064x%02x", r, s, v)
	out := Output{
		RpcUrl:         flag.FlagRpcUrl,
		ChainId:        flag.FlagChainID,
		RawTransaction: encodedRawTx,
		TransactionSig: txSig,
	}
	outb, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(outb)))
	return nil
}
