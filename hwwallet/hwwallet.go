package hwwallet

import (
	"errors"
	"fmt"

	"github.com/jaanek/jethwallet/accounts"
	"github.com/jaanek/jethwallet/hwwallet/hwcommon"
	"github.com/jaanek/jethwallet/ledger"
	"github.com/jaanek/jethwallet/trezor"
	"github.com/jaanek/jethwallet/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
)

var (
	DefaultHDPaths = []string{
		"m/44'/60'/%d'/0/0", // aka "ledger live"
		// "m/44'/60'/0'/%d",   // aka "ledger legacy"
	}
)

func GetWallets(ui ui.Screen, walletType hwcommon.WalletType) ([]hwcommon.HWWallet, error) {
	var wallets []hwcommon.HWWallet
	var err error
	switch walletType {
	case hwcommon.Trezor:
		wallets, err = trezor.Wallets(ui)
	case hwcommon.Ledger:
		wallets, err = ledger.Wallets(ui)
	default:
		return nil, errors.New("Unsupported hw wallet type")
	}
	return wallets, err
}

func ListAccounts(term ui.Screen, walletType hwcommon.WalletType, hdpath string, max int, verbose bool) error {
	switch walletType {
	case hwcommon.Trezor, hwcommon.Ledger:
		wallets, err := GetWallets(term, walletType)
		if err != nil {
			return err
		}
		term.Logf("Found %d wallet(s)\n", len(wallets))
		for _, w := range wallets {
			term.Logf("Wallet status: %s\n", w.Status())
			if hdpath != "" {
				acc, err := Account(w, hdpath)
				if err != nil {
					return err
				}
				term.Logf("%s %s", acc.Address.Hex(), acc.URL.Path)
				break
			}
			accs, err := Accounts(w, DefaultHDPaths, max)
			if err != nil {
				return err
			}
			for _, acc := range accs {
				if verbose {
					term.Output(fmt.Sprintf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path))
				} else {
					term.Output(fmt.Sprintf("%s\n", acc.Address.Hex()))
				}
			}
		}
	default:
		return errors.New("Unknown hardware wallet type!")
	}
	return nil
}

func SignTx(term ui.Screen, walletType hwcommon.WalletType, fromAddr common.Address, tx types.Transaction, max int) (types.Transaction, error) {
	var signed types.Transaction
	wallets, err := GetWallets(term, walletType)
	if err != nil {
		return nil, err
	}
	hww, acc, _ := FindOneFromWallets(term, wallets, fromAddr, DefaultHDPaths, max)
	if acc == (accounts.Account{}) {
		return nil, errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
	}
	term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
	path, err := accounts.ParseDerivationPath(acc.URL.Path)
	if err != nil {
		return nil, err
	}
	var addr common.Address
	addr, signed, err = hww.SignTx(path, tx, tx.GetChainID())
	if err != nil {
		return nil, err
	}
	if addr != acc.Address {
		return nil, errors.New("Signed tx sender address != provided derivation path address!")
	}
	return signed, nil
}

func SignMsg(term ui.Screen, walletType hwcommon.WalletType, fromAddr common.Address, msg []byte, max int) ([]byte, error) {
	wallets, err := GetWallets(term, walletType)
	if err != nil {
		return nil, err
	}
	hww, acc, _ := FindOneFromWallets(term, wallets, fromAddr, DefaultHDPaths, max)
	if acc == (accounts.Account{}) {
		return nil, errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
	}
	term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
	path, err := accounts.ParseDerivationPath(acc.URL.Path)
	if err != nil {
		return nil, err
	}
	addr, sig, err := hww.SignMessage(path, msg)
	if err != nil {
		return nil, err
	}
	if addr != acc.Address {
		return nil, errors.New("Signed message sender address != provided derivation path address!")
	}
	return sig, nil
}

func Encrypt(term ui.Screen, walletType hwcommon.WalletType, fromAddr common.Address, key []byte, data []byte, max int) ([]byte, error) {
	wallets, err := GetWallets(term, walletType)
	if err != nil {
		return nil, err
	}
	hww, acc, _ := FindOneFromWallets(term, wallets, fromAddr, DefaultHDPaths, max)
	if acc == (accounts.Account{}) {
		return nil, errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
	}
	term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
	path, err := accounts.ParseDerivationPath(acc.URL.Path)
	if err != nil {
		return nil, err
	}
	encrypted, err := hww.Encrypt(path, string(key), data, true, true)
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}

func Decrypt(term ui.Screen, walletType hwcommon.WalletType, fromAddr common.Address, key []byte, data []byte, max int) ([]byte, error) {
	wallets, err := GetWallets(term, walletType)
	if err != nil {
		return nil, err
	}
	hww, acc, _ := FindOneFromWallets(term, wallets, fromAddr, DefaultHDPaths, max)
	if acc == (accounts.Account{}) {
		return nil, errors.New(fmt.Sprintf("No account found for address: %s\n", fromAddr))
	}
	term.Logf("Found account: %v, path: %s ...\n", acc.Address, acc.URL.Path)
	path, err := accounts.ParseDerivationPath(acc.URL.Path)
	if err != nil {
		return nil, err
	}
	decrypted, err := hww.Decrypt(path, string(key), data, true, true)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}

func FindOneFromWallets(term ui.Screen, wallets []hwcommon.HWWallet, fromAddr common.Address, defaultHDPaths []string, max int) (hwcommon.HWWallet, accounts.Account, error) {
	var (
		acc accounts.Account
		hww hwcommon.HWWallet
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
func Find(wallet hwcommon.HWWallet, a common.Address, defaultHDPaths []string, max int) ([]accounts.Account, error) {
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

func FindOne(wallet hwcommon.HWWallet, a common.Address, defaultHDPaths []string, max int) (accounts.Account, error) {
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

func Account(wallet hwcommon.HWWallet, hdpath string) (accounts.Account, error) {
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

func Accounts(wallet hwcommon.HWWallet, defaultHDPaths []string, max int) ([]accounts.Account, error) {
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
