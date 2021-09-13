package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/hwwallet/hwcommon"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
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
	var err error
	if flag.KeystorePath != "" {
		signature, err = keystore.SignMsg(term, flag.KeystorePath, fromAddr, msg)
	} else {
		hwWalletType := hwcommon.GetWalletTypeFromFlags(flag)
		signature, err = hwwallet.SignMsg(term, hwWalletType, fromAddr, msg, flag.Max)
	}
	if err != nil {
		return fmt.Errorf("Error while signing message: %w", err)
	}
	term.Output(hexutil.Encode(signature[:]))
	return nil
}
