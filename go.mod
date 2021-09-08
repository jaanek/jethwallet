module github.com/jaanek/jethwallet

go 1.17

replace github.com/ethereum/go-ethereum => ../../3party/go-ethereum-jaanek

require (
	github.com/ethereum/go-ethereum v1.10.8
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.5
	github.com/karalabe/usb v0.0.0-20190919080040-51dc0efba356
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	google.golang.org/protobuf v1.27.1
)

require (
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/deckarep/golang-set v0.0.0-20180603214616-504e848d77ea // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/rjeczalik/notify v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.0.0-20210816183151-1e6c022a8912 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
)
