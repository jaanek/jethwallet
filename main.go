package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/jaanek/jethwallet/wallet"
	"github.com/spf13/cobra"
)

var (
	keystorePath string
	useTrezor    bool
	useLedger    bool
	hdpath       string
	max          int
	open         bool

	// tx params
	flagNonce     string
	flagFrom      string
	flagTo        string
	flagGasLimit  string
	flagGasPrice  string
	flagGasTip    string
	flagGasFeeCap string
	flagValue     string
	flagChainID   string
	flagInput     string
	flagSig       bool
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
	txCmd.Flags().StringVar(&flagNonce, "nonce", "", "")
	txCmd.Flags().StringVar(&flagFrom, "from", "", "")
	txCmd.Flags().StringVar(&flagTo, "to", "", "")
	txCmd.Flags().StringVar(&flagGasLimit, "gas-limit", "", "")
	txCmd.Flags().StringVar(&flagGasPrice, "gas-price", "", "")
	txCmd.Flags().StringVar(&flagGasTip, "gas-tip", "", "")
	txCmd.Flags().StringVar(&flagGasFeeCap, "gas-maxfee", "", "")
	txCmd.Flags().StringVar(&flagValue, "value", "", "")
	txCmd.Flags().StringVar(&flagChainID, "chain-id", "", "")
	txCmd.Flags().StringVar(&flagInput, "input", "", "")
	txCmd.Flags().BoolVar(&flagSig, "sig", false, "")

	rootCmd.AddCommand(listAccountsCmd)
	rootCmd.AddCommand(newAccountsCmd)
	rootCmd.AddCommand(txCmd)
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
		term := ui.NewTerminal()
		if useTrezor || useLedger {
			wallets, err := wallet.GetHWWallets(term, useTrezor, useLedger)
			if err != nil {
				return err
			}
			term.Logf("Found %d wallet(s)\n", len(wallets))
			for _, w := range wallets {
				term.Logf("Wallet status: %s\n", w.Status())
				if hdpath != "" {
					acc, err := hwwallet.Account(w, hdpath)
					if err != nil {
						return err
					}
					term.Logf("%s %s", acc.Address.Hex(), acc.URL.Path)
					break
				}
				accs, err := hwwallet.Accounts(w, hwwallet.DefaultHDPaths, max)
				if err != nil {
					return err
				}
				for _, acc := range accs {
					term.Logf("%s hd-path-%s\n", acc.Address.Hex(), acc.URL.Path)
				}
			}
		} else if keystorePath != "" {
			ks := keystore.NewKeyStore(term, keystorePath)
			accounts, err := ks.Accounts()
			if err != nil {
				return err
			}
			term.Logf("Found %d account(s)\n", len(accounts))
			for _, acc := range accounts {
				term.Logf("Account address: %s, path: %s\n", acc.Address, acc.URL.Path)
			}
		}
		return nil
	},
}

var newAccountsCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new account in keystore",
	RunE: func(cmd *cobra.Command, args []string) error {
		if keystorePath == "" {
			return errors.New("Only supports creating new accounts in keystore!")
		}
		term := ui.NewTerminal()
		ks := keystore.NewKeyStore(term, keystorePath)
		term.Print("*** Enter passphrase (not echoed)...")
		passphrase, err := term.ReadPassword()
		if err != nil {
			return err
		}
		acc, err := ks.NewAccount(string(passphrase))
		if err != nil {
			return err
		}
		term.Logf("New account created! Address: %s, path: %v\n", acc.Address, acc.URL.Path)
		return nil
	},
}

var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Sign a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal()
		err := signTx(term, cmd, args)
		if err != nil {
			term.Error(err)
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
