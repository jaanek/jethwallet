package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/karalabe/usb"
)

type WalletAccount struct {
	account accounts.Account
	path    *accounts.DerivationPath
}

func getWallets(kspaths []string, trezor, ledger bool) []accounts.Wallet {
	wallets := []accounts.Wallet{}

	if len(kspaths) > 0 {
		fmt.Println("Trying to open keystore wallets ...")
		for _, path := range kspaths {
			ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
			wallets = append(wallets, ks.Wallets()...)
		}
	}
	if ledger {
		fmt.Println("Trying to open ledger wallets ...")
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to look for USB Ledgers")
		} else {
			wallets = append(wallets, ledgerhub.Wallets()...)
		}
	}
	if trezor {
		fmt.Println("Trying to open trezor wallets ...")

		// debug
		vendorID := 0x1209
		infos, err := usb.Enumerate(uint16(vendorID), 0)
		if err != nil {
			fmt.Printf("Failed to enumerate USB devices. Vendor ID: %d, err: %v\n", vendorID, err)
		}
		fmt.Printf("Found %d trezor devices\n", len(infos))
		for _, device := range infos {
			fmt.Printf("Device, vendor id: %d, product id: %d, path: %s\n", device.VendorID, device.ProductID, device.Path)
		}
		// for _, info := range infos {
		// 	for _, id := range hub.productIDs {
		// 		// Windows and Macos use UsageID matching, Linux uses Interface matching
		// 		if info.ProductID == id && (info.UsagePage == hub.usageID || info.Interface == hub.endpointID) {
		// 			devices = append(devices, info)
		// 			break
		// 		}
		// 	}
		// }

		//
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			fmt.Fprintf(os.Stderr, "ethsign: failed to look for USB Trezors (HID)")
		} else {
			hws := trezorhub.Wallets()
			fmt.Printf("Trezor wallets %d\n", len(hws))
			for _, hw := range hws {
				fmt.Printf("Opening trezor wallet ...")
				hw.Open("")
			}
			wallets = append(wallets, hws...)
		}
	}
	return wallets
}

func getWalletAccounts(wallet accounts.Wallet) ([]WalletAccount, error) {
	fmt.Printf("Wallet: %s", wallet.URL())
	accs := []WalletAccount{}
	if wallet.URL().Scheme == "keystore" {
		for _, acc := range wallet.Accounts() {
			accs = append(accs, WalletAccount{acc, nil})
		}
	} else if wallet.URL().Scheme == "ledger" || wallet.URL().Scheme == "trezor" {
		fmt.Println("Deriving accounts from ledger wallet")
		wallet.Open("")
		// if hd-path is given, check that only
		if hdpath != "" {
			path, _ := accounts.ParseDerivationPath(hdpath)
			da, err := wallet.Derive(path, false)
			if err != nil {
				return nil, err
			} else {
				accs = append(accs, WalletAccount{da, &path})
			}
			fmt.Printf("%s: ledger-%s\n", da.Address.Hex(), path.String())
		} else {
			// derive wallets from hd-paths
			for i := range defaultHDPaths {
				for j := 0; j <= max; j++ {
					pathstr := fmt.Sprintf(defaultHDPaths[i], j)
					path, _ := accounts.ParseDerivationPath(pathstr)
					da, err := wallet.Derive(path, false)
					if err != nil {
						return nil, err
					} else {
						accs = append(accs, WalletAccount{da, &path})
					}
					fmt.Printf("%s: ledger-%s\n", da.Address.Hex(), path.String())
				}
			}
		}
	}
	return accs, nil
}
