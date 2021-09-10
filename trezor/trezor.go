package trezor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/holiman/uint256"
	"github.com/jaanek/jethwallet/accounts"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/trezor/trezorproto"
	"github.com/jaanek/jethwallet/ui"
	"github.com/karalabe/usb"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
)

const (
	// USB vendor identifier used for device discovery
	vendorID = 0x1209
)

var (
	// USB product identifiers used for device discovery
	productIDs = [...]uint16{
		0x0001, // Trezor HID
		0x53c1, // Trezor WebUSB
	}
	// USB usage page identifier used for macOS device discovery
	usageID uint16 = 0xff00
	// USB endpoint identifier used for non-macOS device discovery
	endpointID = 0
)

const PIN_MATRIX = `
Use the numeric keypad or lowercase letters to describe number positions.
The layout is:
    7 8 9        e r t
    4 5 6  -or-  d f g
    1 2 3        c v b
`

type trezorWallet struct {
	ui       ui.Screen
	device   usb.Device // USB device advertising itself as a hardware wallet
	features *trezorproto.Features
}

func Wallets(term ui.Screen) ([]hwwallet.HWWallet, error) {
	var infos []usb.DeviceInfo
	allInfos, err := usb.Enumerate(vendorID, 0)
	if err != nil {
		return nil, err
	}
	for _, info := range allInfos {
		for _, id := range productIDs {
			// Windows and Macos use UsageID matching, Linux uses Interface matching
			if info.ProductID == id && (info.UsagePage == usageID || info.Interface == endpointID) {
				infos = append(infos, info)
				break
			}
		}
	}
	wallets := make([]hwwallet.HWWallet, 0, len(infos))
	for _, info := range infos {
		device, err := info.Open()
		if err != nil {
			term.Errorf("Cannot open trezor device: %v\n", info)
			continue
		}
		// init device
		wallet := &trezorWallet{
			ui:     term,
			device: device,
		}
		err = wallet.init()
		if err != nil {
			term.Errorf("Cannot initialize trezor device: %v\n", err)
			continue
		}
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

// https://github.com/trezor/trezor-firmware/blob/eb34c0850e8bc74852b5f8aca5c3ab78dc863796/python/src/trezorlib/client.py#L263
func (w *trezorWallet) init() error {
	kind, reply, err := w.rawCall(&trezorproto.Initialize{SessionId: nil})
	if err != nil {
		return err
	}
	if kind != trezorproto.MessageType_MessageType_Features {
		return fmt.Errorf("trezor: expected reply type %s, got %s", MessageName(trezorproto.MessageType_MessageType_Features), MessageName(kind))
	}
	features := new(trezorproto.Features)
	err = proto.Unmarshal(reply, features)
	if err != nil {
		return err
	}
	w.features = features
	w.ui.Logf("Initialized trezor device: %s\n", w.Label())
	return nil
}

func (w *trezorWallet) Scheme() string {
	return "trezor"
}

func (w *trezorWallet) Status() string {
	if w.device == nil || w.features == nil {
		return "Closed"
	}
	return fmt.Sprintf("Trezor v%s '%s' online", w.Version(), w.Label())
}

func (w *trezorWallet) Version() string {
	var version = [3]uint32{w.features.GetMajorVersion(), w.features.GetMinorVersion(), w.features.GetPatchVersion()}
	return fmt.Sprintf("%d.%d.%d", version[0], version[1], version[2])
}

func (w *trezorWallet) Label() string {
	if w.features == nil {
		return ""
	}
	return w.features.GetLabel()
}

// https://github.com/trezor/trezor-firmware/blob/master/python/src/trezorlib/misc.py#L63
func (w *trezorWallet) Encrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error) {
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	var err error
	data, err = pkcs7pad(data, 16)
	if err != nil {
		return nil, err
	}
	var t bool = true
	var request = &trezorproto.CipherKeyValue{
		AddressN:     []uint32(path),
		Key:          &key,
		Value:        data,
		Encrypt:      &t,
		AskOnEncrypt: &askOnEncrypt,
		AskOnDecrypt: &askOnDecrypt,
		Iv:           []byte{},
	}
	response := new(trezorproto.CipheredKeyValue)
	if err := w.Call(request, response); err != nil {
		return nil, err
	}
	return response.Value, nil
}

// https://github.com/trezor/trezor-firmware/blob/master/python/src/trezorlib/misc.py#L87
func (w *trezorWallet) Decrypt(path accounts.DerivationPath, key string, data []byte, askOnEncrypt, askOnDecrypt bool) ([]byte, error) {
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	var t bool = false
	var request = &trezorproto.CipherKeyValue{
		AddressN:     []uint32(path),
		Key:          &key,
		Value:        data,
		Encrypt:      &t,
		AskOnEncrypt: &askOnEncrypt,
		AskOnDecrypt: &askOnDecrypt,
		Iv:           []byte{},
	}
	response := new(trezorproto.CipheredKeyValue)
	if err := w.Call(request, response); err != nil {
		return nil, err
	}
	data, err := pkcs7strip(response.Value, 16)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// https://github.com/trezor/trezor-firmware/blob/master/python/src/trezorlib/ethereum.py#L127
func (w *trezorWallet) SignMessage(path accounts.DerivationPath, msg []byte) (common.Address, []byte, error) {
	if w.device == nil {
		return common.Address{}, nil, accounts.ErrWalletClosed
	}
	var request = &trezorproto.EthereumSignMessage{
		AddressN: []uint32(path),
		Message:  msg,
	}
	response := new(trezorproto.EthereumMessageSignature)
	if err := w.Call(request, response); err != nil {
		return common.Address{}, nil, err
	}
	return common.HexToAddress(*response.Address), response.Signature, nil
}

// SignTx sends the transaction to the Trezor and
// waits for the user to confirm or deny the transaction.
func (w *trezorWallet) SignTx(path accounts.DerivationPath, tx types.Transaction, chainID *uint256.Int) (common.Address, types.Transaction, error) {
	if w.device == nil {
		return common.Address{}, nil, accounts.ErrWalletClosed
	}
	chainId := uint32(chainID.ToBig().Int64()) // EIP-155 transaction, set chain ID explicitly (only 32 bit is supported!?)
	var toAddr *string
	if to := tx.GetTo(); to != nil {
		// Non contract deploy, set recipient explicitly
		hex := to.Hex()
		toAddr = &hex
	}

	// data chunk setup
	data := tx.GetData()
	length := uint32(len(data))
	var dataInitialChunk []byte
	if length > 1024 { // Send the data chunked if that was requested
		dataInitialChunk, data = data[:1024], data[1024:]
	} else {
		dataInitialChunk, data = data, nil
	}

	// build trezor tx
	var req proto.Message
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		var request = &trezorproto.EthereumSignTx{
			AddressN:         []uint32(path),
			To:               toAddr,
			Nonce:            new(big.Int).SetUint64(tx.GetNonce()).Bytes(),
			GasPrice:         tx.GetPrice().Bytes(),
			GasLimit:         new(big.Int).SetUint64(tx.GetGas()).Bytes(),
			Value:            tx.GetValue().Bytes(),
			DataInitialChunk: dataInitialChunk,
			DataLength:       &length,
			ChainId:          &chainId,
		}
		req = request
	case types.DynamicFeeTxType:
		var request = &trezorproto.EthereumSignTxEIP1559{
			AddressN:         []uint32(path),
			To:               toAddr,
			Nonce:            new(big.Int).SetUint64(tx.GetNonce()).Bytes(),
			MaxGasFee:        tx.GetFeeCap().Bytes(),
			MaxPriorityFee:   tx.GetTip().Bytes(),
			GasLimit:         new(big.Int).SetUint64(tx.GetGas()).Bytes(),
			Value:            tx.GetValue().Bytes(),
			DataInitialChunk: dataInitialChunk,
			DataLength:       &length,
			ChainId:          &chainId,
		}
		req = request
	default:
		return common.Address{}, nil, fmt.Errorf("unsupported tx type %d", tx.Type())
	}
	// Send the initiation message and stream content until a signature is returned
	return w.sendTx(req, tx, chainID, data)
}

func (w *trezorWallet) sendTx(req proto.Message, tx types.Transaction, chainID *uint256.Int, data []byte) (common.Address, types.Transaction, error) {
	response := new(trezorproto.EthereumTxRequest)
	if err := w.Call(req, response); err != nil {
		return common.Address{}, nil, err
	}
	for response.DataLength != nil && int(*response.DataLength) <= len(data) {
		var chunk []byte
		dataLen := *response.DataLength
		chunk, data = data[:dataLen], data[dataLen:]

		if err := w.Call(&trezorproto.EthereumTxAck{DataChunk: chunk}, response); err != nil {
			return common.Address{}, nil, err
		}
	}
	// Extract the Ethereum signature and do a sanity validation
	if len(response.GetSignatureR()) == 0 || len(response.GetSignatureS()) == 0 || response.GetSignatureV() == 0 {
		return common.Address{}, nil, errors.New("reply lacks signature")
	}
	signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

	// Create the correct signer and signature transform based on the chain ID
	// signer := types.NewLondonSigner(&chainID)
	signer := types.LatestSignerForChainID(chainID.ToBig())
	signature[64] -= byte(chainID.Uint64()*2 + 35)

	// Inject the final signature into the transaction and sanity check the sender
	signed, err := tx.WithSignature(*signer, signature)
	if err != nil {
		return common.Address{}, nil, err
	}
	// sender, err := types.Sender(signer, signed)
	sender, err := signed.Sender(*signer)
	if err != nil {
		return common.Address{}, nil, err
	}
	return sender, signed, nil
}

func (w *trezorWallet) Derive(path accounts.DerivationPath) (common.Address, error) {
	address := new(trezorproto.EthereumAddress)
	if err := w.Call(&trezorproto.EthereumGetAddress{AddressN: []uint32(path)}, address); err != nil {
		return common.Address{}, err
	}
	if addr := address.GetXOldAddress(); len(addr) > 0 { // Older firmwares use binary formats
		return common.BytesToAddress(addr), nil
	}
	if addr := address.GetAddress(); len(addr) > 0 { // Newer firmwares use hexadecimal formats
		return common.HexToAddress(addr), nil
	}
	return common.Address{}, errors.New("missing derived address")
}

// https://github.com/trezor/trezor-firmware/blob/master/python/src/trezorlib/client.py#L216
func (w *trezorWallet) Call(req proto.Message, result proto.Message) error {
	kind, reply, err := w.rawCall(req)
	if err != nil {
		return err
	}
	for {
		// fmt.Printf("for loop new call. kind: %s ...\n", MessageName(kind))
		switch kind {
		case trezorproto.MessageType_MessageType_PinMatrixRequest:
			{
				w.ui.Print("*** NB! Enter PIN (not echoed)...")
				w.ui.Print(PIN_MATRIX)
				pin, err := w.ui.ReadPassword()
				if err != nil {
					kind, reply, _ = w.rawCall(&trezorproto.Cancel{})
					return err
				}
				// check if pin is valid
				pinStr := string(pin)
				for _, d := range pinStr {
					if !strings.ContainsRune("123456789", d) || len(pin) < 1 {
						kind, reply, _ = w.rawCall(&trezorproto.Cancel{})
						return errors.New("Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = w.rawCall(&trezorproto.PinMatrixAck{Pin: &pinStr})
				if err != nil {
					return err
				}
				w.ui.Logf("Trezor pin success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_PassphraseRequest:
			{
				w.ui.Print("*** NB! Enter Passphrase ...")
				pass, err := w.ui.ReadPassword()
				if err != nil {
					kind, reply, _ = w.rawCall(&trezorproto.Cancel{})
					return err
				}
				passStr := string(pass)
				// send it
				kind, reply, err = w.rawCall(&trezorproto.PassphraseAck{Passphrase: &passStr})
				if err != nil {
					return err
				}
				w.ui.Logf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_ButtonRequest:
			{
				w.ui.Print("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = w.rawCall(&trezorproto.ButtonAck{})
				if err != nil {
					return err
				}
				w.ui.Logf("Trezor button success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_Failure:
			{
				// Trezor returned a failure, extract and return the message
				failure := new(trezorproto.Failure)
				if err := proto.Unmarshal(reply, failure); err != nil {
					return err
				}
				// fmt.Printf("Trezor failure success. kind: %s\n", MessageName(kind))
				return errors.New("trezor: " + failure.GetMessage())
			}
		default:
			{
				resultKind := MessageType(result)
				if resultKind != kind {
					return fmt.Errorf("trezor: expected reply type %s, got %s", MessageName(resultKind), MessageName(kind))
				}
				return proto.Unmarshal(reply, result)
			}
		}
	}
}

// Type returns the protocol buffer type number of a specific message. If the
// message is nil, this method panics!
func MessageType(msg proto.Message) trezorproto.MessageType {
	return trezorproto.MessageType(trezorproto.MessageType_value["MessageType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type number.
func MessageName(kind trezorproto.MessageType) string {
	name := trezorproto.MessageType_name[int32(kind)]
	if len(name) < 12 {
		return name
	}
	return name[12:]
}

//
// Shameless copy (with little modifications) from go-ethereum project
//
// rawCall performs a data exchange with the Trezor wallet, sending it a
// message and retrieving the raw response.
func (w *trezorWallet) rawCall(req proto.Message) (trezorproto.MessageType, []byte, error) {
	// Construct the original message payload to chunk up
	data, err := proto.Marshal(req)
	if err != nil {
		return 0, nil, err
	}
	payload := make([]byte, 8+len(data))
	copy(payload, []byte{0x23, 0x23})
	binary.BigEndian.PutUint16(payload[2:], uint16(MessageType(req)))
	binary.BigEndian.PutUint32(payload[4:], uint32(len(data)))
	copy(payload[8:], data)

	// Stream all the chunks to the device
	chunk := make([]byte, 64)
	chunk[0] = 0x3f // Report ID magic number

	for len(payload) > 0 {
		// Construct the new message to stream, padding with zeroes if needed
		if len(payload) > 63 {
			copy(chunk[1:], payload[:63])
			payload = payload[63:]
		} else {
			copy(chunk[1:], payload)
			copy(chunk[1+len(payload):], make([]byte, 63-len(payload)))
			payload = nil
		}
		// Send over to the device
		// fmt.Printf("Data chunk sent to the Trezor: %v\n", hexutil.Bytes(chunk))
		if _, err := w.device.Write(chunk); err != nil {
			return 0, nil, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {
		// Read the next chunk from the Trezor wallet
		if _, err := io.ReadFull(w.device, chunk); err != nil {
			return 0, nil, err
		}

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
			return 0, nil, errTrezorReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the reply message type and total message length
		var payload []byte

		if len(reply) == 0 {
			kind = binary.BigEndian.Uint16(chunk[3:5])
			reply = make([]byte, 0, int(binary.BigEndian.Uint32(chunk[5:9])))
			payload = chunk[9:]
		} else {
			payload = chunk[1:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	return trezorproto.MessageType(kind), reply, nil
}

// errTrezorReplyInvalidHeader is the error message returned by a Trezor data exchange
// if the device replies with a mismatching header. This usually means the device
// is in browser mode.
var errTrezorReplyInvalidHeader = errors.New("trezor: invalid reply header")

// pkcs7strip remove pkcs7 padding
func pkcs7strip(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("pkcs7: Data is empty")
	}
	if length%blockSize != 0 {
		return nil, errors.New("pkcs7: Data is not block-aligned")
	}
	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen)
	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errors.New("pkcs7: Invalid padding")
	}
	return data[:length-padLen], nil
}

// pkcs7pad add pkcs7 padding
func pkcs7pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 0 || blockSize > 256 {
		return nil, fmt.Errorf("pkcs7: Invalid block size %d", blockSize)
	} else {
		padLen := blockSize - len(data)%blockSize
		padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
		return append(data, padding...), nil
	}
}
