package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaanek/jethwallet/trezor"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

var (
	keystorePaths  []string
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
	rootCmd.PersistentFlags().StringSliceVar(&keystorePaths, "keystore", []string{}, "An array of key-store paths")
	rootCmd.PersistentFlags().BoolVar(&useTrezor, "trezor", false, "Use trezor wallet")
	rootCmd.PersistentFlags().BoolVar(&useLedger, "ledger", false, "Use ledger wallet")
	rootCmd.PersistentFlags().BoolVar(&open, "open", false, "Force to open wallet")
	rootCmd.PersistentFlags().IntVarP(&max, "max", "n", 2, "max hd-paths to derive from")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if len(keystorePaths) == 0 && !useTrezor && !useLedger {
			return errors.New("Specify wallet type to connect to: --keystore, --trezor or --ledger")
		}
		return nil
	}
	rootCmd.AddCommand(listAccountsCmd)

	listAccountsCmd.Flags().StringVar(&hdpath, "hd", "", "hd derivation path")
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
		if useTrezor {
			wallets, err := trezor.Wallets()
			if err != nil {
				return err
			}
			fmt.Printf("Found %d wallets\n", len(wallets))
			for _, w := range wallets {
				if hdpath != "" {
					acc, err := wallet.GetHWWalletAccount(screen, w, hdpath)
					if err != nil {
						return err
					}
					fmt.Printf("%s %s", acc.Address.Hex(), acc.URL.Path)
					break
				}
				accs, err := wallet.GetHWWalletAccounts(screen, w, defaultHDPaths, max)
				if err != nil {
					return err
				}
				for _, acc := range accs {
					fmt.Printf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path)
				}
			}
		} else {
			return errors.New("Other wallets not supported at the moment!")
		}
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
