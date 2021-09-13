package wallet

import (
	"errors"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
)

type WrappedLegacyTx struct {
	types.LegacyTx
	ChainID *uint256.Int
}

func (tx WrappedLegacyTx) GetChainID() *uint256.Int {
	return tx.ChainID
}

func NewTransaction(chainID uint256.Int, nonce uint64, to *common.Address, value *uint256.Int, input []byte, gasLimit uint64, gasPrice, gasTip, gasFeeCap *uint256.Int) (types.Transaction, error) {
	var rawTx types.Transaction
	if gasTip != nil {
		if gasFeeCap == nil {
			gasFeeCap = gasPrice
		}
		tx := &types.DynamicFeeTransaction{
			CommonTx: types.CommonTx{
				Nonce: nonce,
				Data:  input,
				Gas:   gasLimit,
				Value: value,
			},
			ChainID: &chainID,
			Tip:     gasTip,
			FeeCap:  gasFeeCap,
		}
		if to != nil {
			tx.To = to
		}
		rawTx = tx
	} else if gasPrice != nil {
		tx := &WrappedLegacyTx{
			LegacyTx: types.LegacyTx{
				CommonTx: types.CommonTx{
					Nonce: nonce,
					Data:  input,
					Gas:   gasLimit,
					Value: value,
				},
				GasPrice: gasPrice,
			},
			ChainID: &chainID,
		}
		if to != nil {
			tx.To = to
		}
		rawTx = tx
	} else {
		return nil, errors.New("Either --gas-price or (--gas-tip and --gas-price) must be specified")
	}
	return rawTx, nil
}
