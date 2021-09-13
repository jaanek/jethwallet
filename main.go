package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jaanek/jethwallet/flags"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/hwwallet/hwcommon"
	"github.com/jaanek/jethwallet/keystore"
	"github.com/jaanek/jethwallet/ui"
	"github.com/spf13/cobra"
)

type StdInput struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	From           string `json:"from"`
	To             string `json:"to"`
	Value          string `json:"value"`
	Data           string `json:"data"`
	GasTip         string `json:"gasTip"`
	GasPrice       string `json:"gasPrice"`
	Gas            string `json:"gas"`
	TxCount        string `json:"txCount"`
	TxCountPending string `json:"txCountPending"`
	Balance        string `json:"balance"`
}

var flag = flags.Flags{}

func init() {
	rootCmd.PersistentFlags().StringVar(&flag.KeystorePath, "keystore", "", "A key-store directory path")
	rootCmd.PersistentFlags().BoolVar(&flag.UseTrezor, "trezor", false, "Use trezor wallet")
	rootCmd.PersistentFlags().BoolVar(&flag.UseLedger, "ledger", false, "Use ledger wallet")
	rootCmd.PersistentFlags().IntVarP(&flag.Max, "max", "n", 2, "max hd-paths to derive from")
	rootCmd.PersistentFlags().BoolVarP(&flag.FlagVerbose, "verbose", "v", false, "output debug info")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flag.KeystorePath == "" && !flag.UseTrezor && !flag.UseLedger {
			return errors.New("Specify wallet type to connect to: --keystore, --trezor or --ledger")
		}
		return nil
	}

	// list cmd flags
	listAccountsCmd.Flags().StringVar(&flag.Hdpath, "hd", "", "hd derivation path")

	// sign tx flags
	signCmd.Flags().StringVar(&flag.FlagNonce, "nonce", "", "")
	signCmd.Flags().StringVar(&flag.FlagFrom, "from", "", "an account to send from")
	signCmd.Flags().StringVar(&flag.FlagTo, "to", "", "send to or if not provided then input required with contract data")
	signCmd.Flags().StringVar(&flag.FlagGasLimit, "gas-limit", "", "in wei")
	signCmd.Flags().StringVar(&flag.FlagGasPrice, "gas-price", "", "for legacy tx")
	signCmd.Flags().StringVar(&flag.FlagGasTip, "gas-tip", "", "for dynamic tx")
	signCmd.Flags().StringVar(&flag.FlagGasFeeCap, "gas-maxfee", "", "for dynamic tx")
	signCmd.Flags().StringVar(&flag.FlagValue, "value", "", "in wei")
	signCmd.Flags().BoolVar(&flag.FlagValueGwei, "value-gwei", false, "indicate that provided --value is in gwei and not in wei")
	signCmd.Flags().BoolVar(&flag.FlagValueEth, "value-eth", false, "indicate that provided --value is in eth and not in wei")
	signCmd.Flags().StringVar(&flag.FlagChainID, "chain-id", "", "1: mainnet, 5: goerli, 250: Fantom, 137: Matic/Polygon")
	signCmd.Flags().StringVar(&flag.FlagInput, "input", "", "A hexadecimal input data for tx")
	signCmd.Flags().BoolVar(&flag.FlagSig, "sig", false, "output only signature parts(r,s,v) in hex")

	// sign msg flags
	signMsgCmd.Flags().StringVar(&flag.FlagFrom, "from", "", "an account to use to sign")
	signMsgCmd.Flags().StringVar(&flag.FlagInput, "data", "", "input data to sign. If prefixed with 0x then interpreted as hexidecimal data, otherwise as plain text")
	signMsgCmd.Flags().BoolVar(&flag.FlagAddEthPrefix, "with-eth-prefix", false, "Add Ethereum signature prefix: 'x19Ethereum Signed Message:' in front of input data")

	// recover address flags
	recoverCmd.Flags().StringVar(&flag.FlagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) that was used to generate a signature")
	recoverCmd.Flags().StringVar(&flag.FlagSignature, "sig", "", "a signature of input data. Used to derive an ethereum address form it")
	recoverCmd.Flags().BoolVar(&flag.FlagAddEthPrefix, "with-eth-prefix", false, "Add Ethereum signature prefix to data before hashing: 'x19Ethereum Signed Message:' in front of input data")

	// encrypt
	hwEncryptCmd.Flags().StringVar(&flag.FlagFrom, "from", "", "an account to use to encrypt")
	hwEncryptCmd.Flags().StringVar(&flag.FlagKey, "key", "", "a key used to encrypt (with 0x prefix means hexadecimal data, otherwise plain text)")
	hwEncryptCmd.Flags().StringVar(&flag.FlagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) to encrypt")

	// decrypt
	hwDecryptCmd.Flags().StringVar(&flag.FlagFrom, "from", "", "an account to use to decrypt")
	hwDecryptCmd.Flags().StringVar(&flag.FlagKey, "key", "", "a key used to decrypt (with 0x prefix means hexadecimal data, otherwise plain text)")
	hwDecryptCmd.Flags().StringVar(&flag.FlagInput, "data", "", "input data (with 0x prefix means hexadecimal data, otherwise plain text) to decrypt")

	rootCmd.AddCommand(listAccountsCmd)
	rootCmd.AddCommand(newAccountCmd)
	rootCmd.AddCommand(importKeyCmd)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		var err error
		if flag.KeystorePath != "" {
			err = keystore.ListAccounts(term, flag.KeystorePath, flag.FlagVerbose)
		} else {
			walletType := hwcommon.GetWalletTypeFromFlags(&flag)
			err = hwwallet.ListAccounts(term, walletType, flag.Hdpath, flag.Max, flag.FlagVerbose)
		}
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := keystore.NewAccount(term, flag.KeystorePath)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

var importKeyCmd = &cobra.Command{
	Use:   "import-key",
	Short: "import hexadecimal private key into keystore",
	RunE: func(cmd *cobra.Command, args []string) error {
		term := ui.NewTerminal(flag.FlagVerbose)
		err := keystore.ImportKey(term, flag.KeystorePath)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := SignTx(term, &flag)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := SignMsg(term, &flag)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := RecoverAddress(term, &flag)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := HwEncrypt(term, &flag)
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
		term := ui.NewTerminal(flag.FlagVerbose)
		err := HwDecrypt(term, &flag)
		if err != nil {
			term.Error(err)
		}
		return nil
	},
}

func main() {
	ctx, cancel := RootContext()
	defer cancel()

	// try to read command params from from std input json stream
	if isReadFromStdInArgSpecified(os.Args) {
		stdInStr := StdInReadAll()
		if len(stdInStr) > 0 {
			input := StdInput{}
			err := json.Unmarshal([]byte(stdInStr), &input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while parsing stdin json: %s\n", err)
			} else {
				flag.FlagChainID = input.ChainId
				flag.FlagRpcUrl = input.RpcUrl
				flag.FlagNonce = input.TxCount
				flag.FlagFrom = input.From
				flag.FlagTo = input.To
				flag.FlagGasLimit = input.Gas
				flag.FlagGasPrice = input.GasPrice
				flag.FlagGasTip = input.GasTip
				flag.FlagGasFeeCap = input.GasPrice
				flag.FlagValue = input.Value
				flag.FlagInput = input.Data
			}
		}
	}

	// run command
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

func isReadFromStdInArgSpecified(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return true
		}
	}
	return false
}

func StdInReadAll() string {
	arr := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		text := scanner.Text()
		if len(text) > 0 {
			arr = append(arr, text)
		} else {
			break
		}
	}
	return strings.Join(arr, "")
}
