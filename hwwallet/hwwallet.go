package hwwallet

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jaanek/jethwallet/ui"
)

var (
	DefaultHDPaths = []string{
		"m/44'/60'/%d'/0/0", // aka "ledger live"
		// "m/44'/60'/0'/%d",   // aka "ledger legacy"
	}
)

type HWWallet interface {
	Scheme() string
	Status() string
	Label() string
	Derive(path accounts.DerivationPath) (common.Address, error)
	SignTx(path accounts.DerivationPath, tx *types.Transaction, chainID big.Int) (common.Address, *types.Transaction, error)
	SignMessage(path accounts.DerivationPath, msg []byte) (common.Address, []byte, error)
	Encrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error)
	Decrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error)
}

func FindOneFromWallets(term ui.Screen, wallets []HWWallet, fromAddr common.Address, defaultHDPaths []string, max int) (HWWallet, accounts.Account, error) {
	var (
		acc accounts.Account
		hww HWWallet
		err error
	)
	for _, w := range wallets {
		acc, err = FindOne(w, fromAddr, defaultHDPaths, max)
		if err != nil {
			// log out that we did not found from wallet or there was multiple
			term.Error(err.Error())
			continue
		}
		hww = w
		break
	}
	return hww, acc, nil
}

// find an address from max paths provided
func Find(wallet HWWallet, a common.Address, defaultHDPaths []string, max int) ([]accounts.Account, error) {
	accs, err := Accounts(wallet, defaultHDPaths, max)
	if err != nil {
		return nil, err
	}
	found := []accounts.Account{}
	for _, acc := range accs {
		if acc.Address == a {
			found = append(found, acc)
		}
	}
	return found, nil
}

func FindOne(wallet HWWallet, a common.Address, defaultHDPaths []string, max int) (accounts.Account, error) {
	accs, err := Find(wallet, a, defaultHDPaths, max)
	if err != nil {
		return accounts.Account{}, err
	}
	if len(accs) == 0 {
		return accounts.Account{}, errors.New(fmt.Sprintf("%s: No accounts found for address: %v", wallet.Scheme(), a))
	}
	if len(accs) > 1 {
		return accounts.Account{}, errors.New(fmt.Sprintf("%s: Found %d accounts for address: %v", wallet.Scheme(), len(accs), a))
	}
	return accs[0], nil
}

func Account(wallet HWWallet, hdpath string) (accounts.Account, error) {
	path, err := accounts.ParseDerivationPath(hdpath)
	if err != nil {
		return accounts.Account{}, err
	}
	da, err := wallet.Derive(path)
	if err != nil {
		return accounts.Account{}, err
	} else {
		return accounts.Account{
			Address: da,
			URL: accounts.URL{
				Scheme: wallet.Scheme(),
				Path:   hdpath,
			},
		}, nil
	}
}

func Accounts(wallet HWWallet, defaultHDPaths []string, max int) ([]accounts.Account, error) {
	accs := []accounts.Account{}
	for i := range defaultHDPaths {
		for j := 0; j <= max; j++ {
			pathstr := fmt.Sprintf(defaultHDPaths[i], j)
			acc, err := Account(wallet, pathstr)
			if err != nil {
				return nil, err
			}
			accs = append(accs, acc)
		}
	}
	return accs, nil
}
