package main

import (
	"errors"
	"strings"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/ledgerwatch/erigon/common/hexutil"
)

func RecoverAddress(term ui.Screen, flag *flags.Flags) error {
	if flag.FlagInput == "" {
		return errors.New("Missing --data")
	}
	if flag.FlagSignature == "" {
		return errors.New("Missing --sig")
	}
	signature := hexutil.MustDecode(flag.FlagSignature)
	data := []byte(flag.FlagInput)
	if strings.HasPrefix(flag.FlagInput, "0x") {
		data = hexutil.MustDecode(flag.FlagInput)
	}
	var msg []byte = data
	if flag.FlagAddEthPrefix {
		msg = wallet.MessageWithEthPrefix(data)
	}

	// recover address from signature
	addr, err := wallet.EcRecover(msg, signature)
	if err != nil {
		return err
	}
	term.Output(hexutil.Encode(addr[:]))
	return nil
}
