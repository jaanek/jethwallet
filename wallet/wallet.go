package wallet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jaanek/jethwallet/ui"
)

const (
	KeyStore = iota
	Ledger
	Trezor
)

type HWWallet interface {
	Label() string
	Derive(ui ui.Screen, path accounts.DerivationPath) (common.Address, error)
}

type Wallet struct {
	accounts.Wallet
}

type WalletAccount struct {
	Wallet  Wallet
	Address common.Address
	Path    *accounts.DerivationPath
}

func GetKSWallets(kspaths []string) []Wallet {
	fmt.Println("Trying to open keystore wallets ...")
	wallets := []Wallet{}
	for _, path := range kspaths {
		ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
		for _, w := range ks.Wallets() {
			wallets = append(wallets, Wallet{w})
		}
	}
	return wallets
}

// func getKSWalletAccounts(wallet Wallet) ([]WalletAccount, error) {
// 	fmt.Printf("Wallet: %s\n", wallet.URL())
// 	accs := []WalletAccount{}
// 	for _, acc := range wallet.Accounts() {
// 		accs = append(accs, WalletAccount{wallet, acc, nil})
// 	}
// 	return accs, nil
// }

// func getWallets(kspaths []string, trezor, ledger bool) []Wallet {
// 	wallets := []Wallet{}

// 	if ledger {
// 		fmt.Println("Trying to open ledger wallets ...")
// 		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
// 			fmt.Fprintf(os.Stderr, "failed to look for USB Ledgers")
// 		} else {
// 			for _, w := range ledgerhub.Wallets() {
// 				wallets = append(wallets, Wallet{w, Ledger})
// 			}
// 		}
// 	}
// 	return wallets
// }

func GetHWWalletAccount(ui ui.Screen, wallet HWWallet, hdpath string) (accounts.Account, error) {
	// fmt.Printf("derive account from hd-path: %s\n", hdpath)
	path, _ := accounts.ParseDerivationPath(hdpath)
	da, err := wallet.Derive(ui, path)
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

func GetHWWalletAccounts(ui ui.Screen, wallet HWWallet, defaultHDPaths []string, max int) ([]accounts.Account, error) {
	accs := []accounts.Account{}
	for i := range defaultHDPaths {
		for j := 0; j <= max; j++ {
			pathstr := fmt.Sprintf(defaultHDPaths[i], j)
			path, _ := accounts.ParseDerivationPath(pathstr)
			// fmt.Printf("derive account from hd-path: %s\n", pathstr)
			da, err := wallet.Derive(ui, path)
			if err != nil {
				return nil, err
			} else {
				accs = append(accs, accounts.Account{
					Address: da,
					URL: accounts.URL{
						Scheme: "trezor",
						Path:   pathstr,
					},
				})
			}
		}
	}
	return accs, nil
}
