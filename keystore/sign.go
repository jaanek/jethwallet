package keystore

import (
	"fmt"

	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
)

func SignTx(term ui.Screen, keystorePath string, fromAddr common.Address, tx types.Transaction) (types.Transaction, error) {
	var signed types.Transaction
	ks := NewKeyStore(term, keystorePath)

	// find the account by address
	acc, err := ks.FindOne(fromAddr)
	if err != nil {
		return nil, err
	}
	term.Print(fmt.Sprintf("*** Enter passphrase (not echoed) account: %v ...", acc.Address))
	passphrase, err := term.ReadPassword()
	if err != nil {
		return nil, err
	}
	key, err := ks.GetDecryptedKey(acc, string(passphrase))
	if err != nil {
		return nil, err
	}
	defer ZeroKey(key.PrivateKey)
	signed, err = ks.SignTx(key.PrivateKey, tx, tx.GetChainID())
	if err != nil {
		return nil, err
	}
	return signed, nil
}
