package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/hwwallet/hwcommon"
	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
)

func HwDecrypt(term ui.Screen, flag *flags.Flags) error {
	if flag.FlagFrom == "" {
		return errors.New("Missing --from address")
	}
	if flag.FlagKey == "" {
		return errors.New("Missing --key")
	}
	if flag.FlagInput == "" {
		return errors.New("Missing --data")
	}
	fromAddr := common.HexToAddress(flag.FlagFrom)
	key := []byte(flag.FlagKey)
	if strings.HasPrefix(flag.FlagKey, "0x") {
		key = hexutil.MustDecode(flag.FlagKey)
	}
	data := []byte(flag.FlagInput)
	if strings.HasPrefix(flag.FlagInput, "0x") {
		data = hexutil.MustDecode(flag.FlagInput)
	}

	// encrypt data
	walletType := hwcommon.GetWalletTypeFromFlags(flag)
	decrypted, err := hwwallet.Decrypt(term, walletType, fromAddr, key, data, flag.Max)
	if err != nil {
		return fmt.Errorf("error while decrypting: %w", err)
	}
	term.Output(string(decrypted))
	return nil
}
