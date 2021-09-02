package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jaanek/jethwallet/ui"
	"github.com/spf13/cobra"
)

func recoverAddress(term ui.Screen, cmd *cobra.Command, args []string) error {
	if flagInput == "" {
		return errors.New("Missing --data")
	}
	if flagSignature == "" {
		return errors.New("Missing --sig")
	}
	signature := hexutil.MustDecode(flagSignature)
	data := []byte(flagInput)
	if strings.HasPrefix(flagInput, "0x") {
		data = hexutil.MustDecode(flagInput)
	}
	var msg []byte = data
	if flagAddEthPrefix {
		msg = []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data))
	}

	// recover address from signature
	addr, err := EcRecover(msg, signature)
	if err != nil {
		return err
	}
	term.Output(hexutil.Encode(addr[:]))
	return nil
}

// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
func EcRecover(data []byte, sig hexutil.Bytes) (common.Address, error) {
	if len(sig) != crypto.SignatureLength {
		return common.Address{}, fmt.Errorf("signature must be %d bytes long", crypto.SignatureLength)
	}
	if sig[crypto.RecoveryIDOffset] != 27 && sig[crypto.RecoveryIDOffset] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := crypto.SigToPub(crypto.Keccak256(data), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}
