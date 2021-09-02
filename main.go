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

	// sign tx params
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

	// sign msg, recover params
	flagAddEthPrefix bool
	flagSignature    string

	// encrypt, decrypt params
	flagKey string
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

	// list cmd flags
	listAccountsCmd.Flags().StringVar(&hdpath, "hd", "", "hd derivation path")

	// sign tx flags
	signCmd.Flags().StringVar(&flagNonce, "nonce", "", "")
	signCmd.Flags().StringVar(&flagFrom, "from", "", "an account to send from")
	signCmd.Flags().StringVar(&flagTo, "to", "", "send to or if not provided then input required with contract data")
	signCmd.Flags().StringVar(&flagGasLimit, "gas-limit", "", "in wei")
	signCmd.Flags().StringVar(&flagGasPrice, "gas-price", "", "for legacy tx")
	signCmd.Flags().StringVar(&flagGasTip, "gas-tip", "", "for dynamic tx")
	signCmd.Flags().StringVar(&flagGasFeeCap, "gas-maxfee", "", "for dynamic tx")
	signCmd.Flags().StringVar(&flagValue, "value", "", "in wei")
	signCmd.Flags().StringVar(&flagChainID, "chain-id", "", "1: mainnet, 5: goerli")
	signCmd.Flags().StringVar(&flagInput, "input", "", "A hexadecimal input data for tx")
	signCmd.Flags().BoolVar(&flagSig, "sig", false, "output only signature parts(r,s,v) in hex")

	// sign msg flags
	signMsgCmd.Flags().StringVar(&flagFrom, "from", "", "an account to use to sign")
	signMsgCmd.Flags().StringVar(&flagInput, "data", "", "input data to sign. If prefixed with 0x then interpreted as hexidecimal data, otherwise as plain text")
	signMsgCmd.Flags().BoolVar(&flagAddEthPrefix, "with-eth-prefix", false, "Add Ethereum signature prefix: 'x19Ethereum Signed Message:' in front of input data")

	// recover address flags
	recoverCmd.Flags().StringVar(&flagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) that was used to generate a signature")
	recoverCmd.Flags().StringVar(&flagSignature, "sig", "", "a signature of input data. Used to derive an ethereum address form it")
	recoverCmd.Flags().BoolVar(&flagAddEthPrefix, "with-eth-prefix", false, "Add Ethereum signature prefix to data before hashing: 'x19Ethereum Signed Message:' in front of input data")

	// encrypt
	hwEncryptCmd.Flags().StringVar(&flagFrom, "from", "", "an account to use to encrypt")
	hwEncryptCmd.Flags().StringVar(&flagKey, "key", "", "a key used to encrypt (with 0x prefix means hexadecimal data, otherwise plain text)")
	hwEncryptCmd.Flags().StringVar(&flagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) to encrypt")

	// decrypt
	hwDecryptCmd.Flags().StringVar(&flagFrom, "from", "", "an account to use to decrypt")
	hwDecryptCmd.Flags().StringVar(&flagKey, "key", "", "a key used to decrypt (with 0x prefix means hexadecimal data, otherwise plain text)")
	hwDecryptCmd.Flags().StringVar(&flagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) to decrypt")

	rootCmd.AddCommand(listAccountsCmd)
	rootCmd.AddCommand(newAccountCmd)
	rootCmd.AddCommand(signCmd)
	rootCmd.AddCommand(signMsgCmd)
	rootCmd.AddCommand(recoverCmd)
	rootCmd.AddCommand(hwEncryptCmd)
	rootCmd.AddCommand(hwDecryptCmd)
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

var signCmd = &cobra.Command{
	Use:     "sign",
	Aliases: []string{"tx"},
	Short:   "Sign a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := signTx(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var signMsgCmd = &cobra.Command{
	Use:     "sign-msg",
	Aliases: []string{"msg"},
	Short:   "Sign a message",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := signMsg(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover an address from signature",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := recoverAddress(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var hwEncryptCmd = &cobra.Command{
	Use:     "hwencrypt",
	Aliases: []string{"hwe"},
	Short:   "Encrypt on Trezor wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := hwEncrypt(term, cmd, args)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var hwDecryptCmd = &cobra.Command{
	Use:     "hwdecrypt",
	Aliases: []string{"hwd"},
	Short:   "Decrypt on Trezor wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flagQuiet)
		err := hwDecrypt(term, cmd, args)
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
