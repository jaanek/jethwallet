package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaanek/jethwallet/ui"
	"github.com/spf13/cobra"
)

var (
	// general params
	keystorePath string
	useTrezor    bool
	useLedger    bool
	hdpath       string
	max          int
	flagQuiet    bool

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
	rootCmd.PersistentFlags().StringVar(&keystorePath, "keystore", "", "A key-store directory path")
	rootCmd.PersistentFlags().BoolVar(&useTrezor, "trezor", false, "Use trezor wallet")
	rootCmd.PersistentFlags().BoolVar(&useLedger, "ledger", false, "Use ledger wallet")
	rootCmd.PersistentFlags().IntVarP(&max, "max", "n", 2, "max hd-paths to derive from")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "be quiet when outputting results")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if keystorePath == "" && !useTrezor && !useLedger {
			return errors.New("Specify wallet type to connect to: --keystore, --trezor or --ledger")
		}
		return nil
	}
	listAccountsCmd.Flags().StringVar(&hdpath, "hd", "", "hd derivation path")
	txCmd.Flags().StringVar(&flagNonce, "nonce", "", "")
	txCmd.Flags().StringVar(&flagFrom, "from", "", "an account to send from")
	txCmd.Flags().StringVar(&flagTo, "to", "", "send to or if not provided then input required with contract data")
	txCmd.Flags().StringVar(&flagGasLimit, "gas-limit", "", "in wei")
	txCmd.Flags().StringVar(&flagGasPrice, "gas-price", "", "for legacy tx")
	txCmd.Flags().StringVar(&flagGasTip, "gas-tip", "", "for dynamic tx")
	txCmd.Flags().StringVar(&flagGasFeeCap, "gas-maxfee", "", "for dynamic tx")
	txCmd.Flags().StringVar(&flagValue, "value", "", "in wei")
	txCmd.Flags().StringVar(&flagChainID, "chain-id", "", "1: mainnet, 5: goerli")
	txCmd.Flags().StringVar(&flagInput, "input", "", "A hexadecimal input data for tx")
	txCmd.Flags().BoolVar(&flagSig, "sig", false, "output only signature parts(r,s,v) in hex")

	rootCmd.AddCommand(listAccountsCmd)
	rootCmd.AddCommand(newAccountCmd)
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
		term := ui.NewTerminal(flagQuiet)
		err := listAccounts(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var newAccountCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new account in keystore",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := newAccount(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Sign a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
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
