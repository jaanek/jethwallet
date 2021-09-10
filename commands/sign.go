package commands

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
	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
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
	var gasPrice, gasTipCap, gasFeeCap *big.Int
	if flag.FlagGasPrice != "" {
		gasPrice = math.MustParseBig256(flag.FlagGasPrice)
	}
	if flag.FlagGasTip != "" {
		gasTipCap = math.MustParseBig256(flag.FlagGasTip)
	}
	if flag.FlagGasFeeCap != "" {
		gasFeeCap = math.MustParseBig256(flag.FlagGasFeeCap)
	}
	if gasPrice == nil && (gasTipCap == nil || gasFeeCap == nil) {
		return errors.New("Either --gas-price or (--gas-tip and --gas-maxfee) must be provided")
	}
	var value *big.Int
	if flag.FlagValue != "" {
		var ok bool
		value, ok = math.ParseBig256(flag.FlagValue)
		if !ok {
			return errors.New(fmt.Sprintf("invalid 256 bit integer: " + flag.FlagValue))
		}
		if flag.FlagValueEth {
			value = new(big.Int).Mul(value, new(big.Int).SetInt64(params.Ether))
		} else if flag.FlagValueGwei {
			value = new(big.Int).Mul(value, new(big.Int).SetInt64(params.GWei))
		}
	} else {
		value = new(big.Int)
	}
	if flag.FlagChainID == "" {
		return errors.New("Missing --chain-id")
	}
	chainID := math.MustParseBig256(flag.FlagChainID)
	if to == nil && flag.FlagInput == "" {
		return errors.New("Either --to or --input must be provided")
	}
	var input = []byte{}
	if flag.FlagInput != "" {
		input = hexutil.MustDecode(flag.FlagInput)
	}

	// Create the transaction to sign
	var rawTx *types.Transaction
	if gasTipCap != nil {
		if gasFeeCap == nil {
			gasFeeCap = gasPrice
		}
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
	} else if gasPrice != nil {
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
		return errors.New("No --gas-price nor --gas-tip found in args")
	}

	// sign tx
	var (
		signed *types.Transaction
	)
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
		var addr common.Address
		addr, signed, err = hww.SignTx(path, rawTx, *chainID)
		if err != nil {
			return err
		}
		if addr != acc.Address {
			return errors.New("Signed tx sender address != provided derivation path address!")
		}
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
		signed, err = ks.SignTx(key.PrivateKey, rawTx, chainID)
		if err != nil {
			return err
		}
	}

	// output
	encoded, err := signed.MarshalBinary()
	if err != nil {
		return err
	}
	encodedRawTx := hexutil.Encode(encoded[:])
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
