package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ledger"
	"github.com/jaanek/jethwallet/trezor"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

var (
	keystorePath   string
	useTrezor      bool
	useLedger      bool
	hdpath         string
	max            int
	open           bool
	defaultHDPaths = []string{
		"m/44'/60'/%d'/0/0", // aka "ledger live"
		// "m/44'/60'/0'/%d",   // aka "ledger legacy"
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&keystorePath, "keystore", "", "An array of key-store paths")
	rootCmd.PersistentFlags().BoolVar(&useTrezor, "trezor", false, "Use trezor wallet")
	rootCmd.PersistentFlags().BoolVar(&useLedger, "ledger", false, "Use ledger wallet")
	rootCmd.PersistentFlags().BoolVar(&open, "open", false, "Force to open wallet")
	rootCmd.PersistentFlags().IntVarP(&max, "max", "n", 2, "max hd-paths to derive from")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if keystorePath == "" && !useTrezor && !useLedger {
			return errors.New("Specify wallet type to connect to: --keystore, --trezor or --ledger")
		}
		return nil
	}
	listAccountsCmd.Flags().StringVar(&hdpath, "hd", "", "hd derivation path")

	rootCmd.AddCommand(listAccountsCmd)
	rootCmd.AddCommand(newAccountsCmd)
}

var rootCmd = &cobra.Command{
	Use:   "jethwallet",
	Short: "Run jeth wallet command",
}

var listAccountsCmd = &cobra.Command{
	Use:     "accounts",
	Aliases: []string{"ls"},
	Short:   "List accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		screen := ui.NewTerminal()
		if useTrezor || useLedger {
			var wallets []wallet.HWWallet
			var err error
			if useTrezor {
				wallets, err = trezor.Wallets(screen)
			} else if useLedger {
				wallets, err = ledger.Wallets(screen)
			}
			if err != nil {
				return err
			}
			fmt.Printf("Found %d wallet(s)\n", len(wallets))
			for _, w := range wallets {
				fmt.Printf("Wallet status: %s\n", w.Status())
				if hdpath != "" {
					acc, err := wallet.GetHWWalletAccount(w, hdpath)
					if err != nil {
						return err
					}
					fmt.Printf("%s %s", acc.Address.Hex(), acc.URL.Path)
					break
				}
				accs, err := wallet.GetHWWalletAccounts(w, defaultHDPaths, max)
				if err != nil {
					return err
				}
				for _, acc := range accs {
					fmt.Printf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path)
				}
			}
		} else if keystorePath != "" {
			ks := keystore.NewKeyStore(screen, keystorePath)
			accounts, err := ks.Accounts()
			if err != nil {
				return err
			}
			fmt.Printf("Found %d account(s)\n", len(accounts))
			for _, acc := range accounts {
				fmt.Printf("Account address: %s, path: %s\n", acc.Address, acc.URL.Path)
			}
		}
		return nil
	},
}

var newAccountsCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new account in keystore",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal()
		if keystorePath == "" {
			return errors.New("Only supports creating new accounts in keystore!")
		}
		ks := keystore.NewKeyStore(term, keystorePath)
		term.Print("*** Enter passphrase (not echoed)...")
		passphrase, err := term.ReadPassword(nil)
		if err != nil {
			return err
		}
		acc, err := ks.NewAccount(string(passphrase))
		if err != nil {
			return err
		}
		fmt.Printf("New account created! Address: %s, path: %v\n", acc.Address, acc.URL.Path)
		return nil
	},
}

func main() {
	ctx, cancel := RootContext()
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func RootContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		ch := make(chan os.Signal, 1)
		defer close(ch)

		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(ch)

		select {
		case sig := <-ch:
			fmt.Printf("Got interrupt %v, shutting down...", sig)
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
