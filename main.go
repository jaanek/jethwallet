package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	keystores      []string
	trezor         bool
	ledger         bool
	hdpath         string
	max            int
	defaultHDPaths = []string{
		"m/44'/60'/%d'/0/0", // aka "ledger live"
		"m/44'/60'/0'/%d",   // aka "ledger legacy"
	}
)

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&keystores, "key-store", "k", []string{}, "An array of key-store paths")
	rootCmd.PersistentFlags().BoolVarP(&trezor, "trezor", "t", false, "Use trezor wallet")
	rootCmd.PersistentFlags().BoolVarP(&ledger, "ledger", "l", false, "Use ledger wallet")
	rootCmd.PersistentFlags().IntVarP(&max, "max", "n", 2, "max hd-paths to derive from")
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
		wallets := getWallets(keystores, trezor, ledger)
		fmt.Printf("Found %d wallets\n", len(wallets))
		for _, w := range wallets {
			accs, err := getWalletAccounts(w)
			if err != nil {
				return err
			}
			for _, wa := range accs {
				fmt.Printf("%s hd-path-%s\n", wa.account.Address.Hex(), wa.path.String())
			}
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
