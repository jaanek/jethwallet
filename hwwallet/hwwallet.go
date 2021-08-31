package hwwallet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

type HWWallet interface {
	Status() string
	Label() string
	Derive(path accounts.DerivationPath) (common.Address, error)
}

func GetAccount(wallet HWWallet, hdpath string) (accounts.Account, error) {
	// fmt.Printf("derive account from hd-path: %s\n", hdpath)
	path, _ := accounts.ParseDerivationPath(hdpath)
	da, err := wallet.Derive(path)
	if err != nil {
		return accounts.Account{}, err
	} else {
		return accounts.Account{
			Address: da,
			URL: accounts.URL{
				Scheme: "trezor",
				Path:   hdpath,
			},
		}, nil
	}
}

func GetAccounts(wallet HWWallet, defaultHDPaths []string, max int) ([]accounts.Account, error) {
	accs := []accounts.Account{}
	for i := range defaultHDPaths {
		for j := 0; j <= max; j++ {
			pathstr := fmt.Sprintf(defaultHDPaths[i], j)
			acc, err := GetAccount(wallet, pathstr)
			if err != nil {
				return nil, err
			}
			accs = append(accs, acc)
		}
	}
	return accs, nil
}