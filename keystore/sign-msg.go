package keystore

import (
	"fmt"

	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/common"
)

func SignMsg(term ui.Screen, keystorePath string, fromAddr common.Address, msg []byte) ([]byte, error) {
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
	sig, err := ks.SignData(key.PrivateKey, msg)
	if err != nil {
		return nil, err
	}
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig, nil
}
