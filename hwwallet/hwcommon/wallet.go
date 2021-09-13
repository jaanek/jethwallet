package hwcommon

import (
	"github.com/holiman/uint256"
	"github.com/jaanek/jethwallet/accounts"
	"github.com/jaanek/jethwallet/flags"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
)

type WalletType int

const (
	Ledger WalletType = iota
	Trezor
)

type HWWallet interface {
	Scheme() string
	Status() string
	Label() string
	Derive(path accounts.DerivationPath) (common.Address, error)
	SignTx(path accounts.DerivationPath, tx types.Transaction, chainID *uint256.Int) (common.Address, types.Transaction, error)
	SignMessage(path accounts.DerivationPath, msg []byte) (common.Address, []byte, error)
	Encrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error)
	Decrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error)
}

func GetWalletTypeFromFlags(flag *flags.Flags) WalletType {
	if flag.UseTrezor {
		return Trezor
	} else if flag.UseLedger {
		return Ledger
	}
	return -1
}
