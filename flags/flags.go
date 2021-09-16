package flags

type Flags struct {
	// general params
	KeystorePath string
	UseTrezor    bool
	UseLedger    bool
	Hdpath       string
	Max          int
	FlagVerbose  bool

	// sign tx params
	FlagNonce         string
	FlagFrom          string
	FlagTo            string
	FlagGasLimit      string
	FlagGasPrice      string
	FlagGasTip        string
	FlagGasFeeCap     string
	FlagValue         string
	FlagValueGwei     bool
	FlagGasPriceGwei  bool
	FlagGasTipGwei    bool
	FlagGasFeeCapGwei bool
	FlagValueEth      bool
	FlagRpcUrl        string
	FlagChainID       string
	FlagInput         string
	FlagInputMethod   string
	FlagSig           bool
	Plain             bool

	// sign msg, recover params
	FlagAddEthPrefix bool
	FlagSignature    string

	// encrypt, decrypt params
	FlagKey string
}
