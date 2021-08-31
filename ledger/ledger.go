package ledger

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jaanek/jethwallet/hwwallet"
	"github.com/jaanek/jethwallet/ui"
	"github.com/karalabe/usb"
)

const (
	// USB vendor identifier used for device discovery
	vendorID = 0x2c97
)

var (
	// USB product identifiers used for device discovery
	productIDs = [...]uint16{
		// Original product IDs
		0x0000, /* Ledger Blue */
		0x0001, /* Ledger Nano S */
		0x0004, /* Ledger Nano X */

		// Upcoming product IDs: https://www.ledger.com/2019/05/17/windows-10-update-sunsetting-u2f-tunnel-transport-for-ledger-devices/
		0x0015, /* HID + U2F + WebUSB Ledger Blue */
		0x1015, /* HID + U2F + WebUSB Ledger Nano S */
		0x4015, /* HID + U2F + WebUSB Ledger Nano X */
		0x0011, /* HID + WebUSB Ledger Blue */
		0x1011, /* HID + WebUSB Ledger Nano S */
		0x4011, /* HID + WebUSB Ledger Nano X */
	}
	// USB usage page identifier used for macOS device discovery
	usageID uint16 = 0xffa0
	// USB endpoint identifier used for non-macOS device discovery
	endpointID = 0
)

// ledgerOpcode is an enumeration encoding the supported Ledger opcodes.
type ledgerOpcode byte

// ledgerParam1 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam1 byte

// ledgerParam2 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam2 byte

const (
	ledgerOpRetrieveAddress  ledgerOpcode = 0x02 // Returns the public key and Ethereum address for a given BIP 32 path
	ledgerOpSignTransaction  ledgerOpcode = 0x04 // Signs an Ethereum transaction after having the user validate the parameters
	ledgerOpGetConfiguration ledgerOpcode = 0x06 // Returns specific wallet application configuration
	ledgerOpSignTypedMessage ledgerOpcode = 0x0c // Signs an Ethereum message following the EIP 712 specification

	ledgerP1DirectlyFetchAddress    ledgerParam1 = 0x00 // Return address directly from the wallet
	ledgerP1InitTypedMessageData    ledgerParam1 = 0x00 // First chunk of Typed Message data
	ledgerP1InitTransactionData     ledgerParam1 = 0x00 // First transaction data block for signing
	ledgerP1ContTransactionData     ledgerParam1 = 0x80 // Subsequent transaction data block for signing
	ledgerP2DiscardAddressChainCode ledgerParam2 = 0x00 // Do not return the chain code along with the address
)

// errLedgerReplyInvalidHeader is the error message returned by a Ledger data exchange
// if the device replies with a mismatching header. This usually means the device
// is in browser mode.
var errLedgerReplyInvalidHeader = errors.New("ledger: invalid reply header")

// errLedgerInvalidVersionReply is the error message returned by a Ledger version retrieval
// when a response does arrive, but it does not contain the expected data.
var errLedgerInvalidVersionReply = errors.New("ledger: invalid version reply")

type ledgerWallet struct {
	ui      ui.Screen
	device  usb.Device // USB device advertising itself as a hardware wallet
	browser bool
	version [3]byte
}

func Wallets(ui ui.Screen) ([]hwwallet.HWWallet, error) {
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
			fmt.Fprintf(os.Stderr, "Cannot open ledger device: %v\n", info)
			continue
		}
		// init device
		wallet := &ledgerWallet{
			ui:      ui,
			device:  device,
			browser: false,
		}
		err = wallet.init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot initialize ledger device: %v\n", err)
			continue
		}
		fmt.Printf("Initialized ledger device: %s\n", wallet.Label())
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

func (w *ledgerWallet) init() error {
	_, err := w.Derive(accounts.DefaultBaseDerivationPath)
	if err != nil {
		// Ethereum app is not running or in browser mode, nothing more to do, return
		if err == errLedgerReplyInvalidHeader {
			fmt.Printf("errLedgerReplyInvalidHeader\n")
			w.browser = true
		}
		return nil
	}
	// Try to resolve the Ethereum app's version, will fail prior to v1.0.2
	if w.version, err = w.ledgerVersion(); err != nil {
		w.version = [3]byte{1, 0, 0} // Assume worst case, can't verify if v1.0.0 or v1.0.1
	}
	fmt.Printf("ledger version: %x\n", w.version)
	return nil
}

func (w *ledgerWallet) Status() string {
	if w.device == nil {
		return "Closed"
	}
	if w.browser {
		return "Ledger Ethereum app in browser mode"
	}
	if w.offline() {
		return "Ledger Ethereum app offline"
	}
	return fmt.Sprintf("Ledger Ethereum app v%s online", w.Version())
}

func (w *ledgerWallet) Version() string {
	return fmt.Sprintf("%d.%d.%d", w.version[0], w.version[1], w.version[2])
}

// offline returns whether the wallet and the Ethereum app is offline or not.
//
// The method assumes that the state lock is held!
func (w *ledgerWallet) offline() bool {
	return w.version == [3]byte{0, 0, 0}
}

// ledgerVersion retrieves the current version of the Ethereum wallet app running
// on the Ledger wallet.
//
// The version retrieval protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc | Le
//   ----+-----+----+----+----+---
//    E0 | 06  | 00 | 00 | 00 | 04
//
// With no input data, and the output data being:
//
//   Description                                        | Length
//   ---------------------------------------------------+--------
//   Flags 01: arbitrary data signature enabled by user | 1 byte
//   Application major version                          | 1 byte
//   Application minor version                          | 1 byte
//   Application patch version                          | 1 byte
func (w *ledgerWallet) ledgerVersion() ([3]byte, error) {
	// Send the request and wait for the response
	reply, err := w.rawCall(ledgerOpGetConfiguration, 0, 0, nil)
	if err != nil {
		return [3]byte{}, err
	}
	if len(reply) != 4 {
		return [3]byte{}, errLedgerInvalidVersionReply
	}
	// Cache the version for future reference
	var version [3]byte
	copy(version[:], reply[1:])
	return version, nil
}

func (w *ledgerWallet) Label() string {
	if w.version == [3]byte{0, 0, 0} {
		return ""
	}
	return fmt.Sprintf("%x", w.version)
}

// Derive retrieves the currently active Ethereum address from a Ledger
// wallet at the specified derivation path.
//
// The address derivation protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc  | Le
//   ----+-----+----+----+-----+---
//    E0 | 02  | 00 return address
//               01 display address and confirm before returning
//                  | 00: do not return the chain code
//                  | 01: return the chain code
//                       | var | 00
//
// Where the input data is:
//
//   Description                                      | Length
//   -------------------------------------------------+--------
//   Number of BIP 32 derivations to perform (max 10) | 1 byte
//   First derivation index (big endian)              | 4 bytes
//   ...                                              | 4 bytes
//   Last derivation index (big endian)               | 4 bytes
//
// And the output data is:
//
//   Description             | Length
//   ------------------------+-------------------
//   Public Key length       | 1 byte
//   Uncompressed Public Key | arbitrary
//   Ethereum address length | 1 byte
//   Ethereum address        | 40 bytes hex ascii
//   Chain code if requested | 32 bytes
func (w *ledgerWallet) Derive(derivationPath accounts.DerivationPath) (common.Address, error) {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Send the request and wait for the response
	reply, err := w.rawCall(ledgerOpRetrieveAddress, ledgerP1DirectlyFetchAddress, ledgerP2DiscardAddressChainCode, path)
	if err != nil {
		return common.Address{}, err
	}
	// Discard the public key, we don't need that for now
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks public key entry")
	}
	reply = reply[1+int(reply[0]):]

	// Extract the Ethereum hex address string
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks address entry")
	}
	hexstr := reply[1 : 1+int(reply[0])]

	// Decode the hex string into an Ethereum address and return
	var address common.Address
	if _, err = hex.Decode(address[:], hexstr); err != nil {
		return common.Address{}, err
	}
	return address, nil
}

//
// Shameless copy (with little modifications) from go-ethereum project
//
// rawCall performs a data exchange with the Ledger wallet, sending it a
// message and retrieving the response.
//
// The common transport header is defined as follows:
//
//  Description                           | Length
//  --------------------------------------+----------
//  Communication channel ID (big endian) | 2 bytes
//  Command tag                           | 1 byte
//  Packet sequence index (big endian)    | 2 bytes
//  Payload                               | arbitrary
//
// The Communication channel ID allows commands multiplexing over the same
// physical link. It is not used for the time being, and should be set to 0101
// to avoid compatibility issues with implementations ignoring a leading 00 byte.
//
// The Command tag describes the message content. Use TAG_APDU (0x05) for standard
// APDU payloads, or TAG_PING (0x02) for a simple link test.
//
// The Packet sequence index describes the current sequence for fragmented payloads.
// The first fragment index is 0x00.
//
// APDU Command payloads are encoded as follows:
//
//  Description              | Length
//  -----------------------------------
//  APDU length (big endian) | 2 bytes
//  APDU CLA                 | 1 byte
//  APDU INS                 | 1 byte
//  APDU P1                  | 1 byte
//  APDU P2                  | 1 byte
//  APDU length              | 1 byte
//  Optional APDU data       | arbitrary
func (w *ledgerWallet) rawCall(opcode ledgerOpcode, p1 ledgerParam1, p2 ledgerParam2, data []byte) ([]byte, error) {
	// Construct the message payload, possibly split into multiple chunks
	apdu := make([]byte, 2, 7+len(data))

	binary.BigEndian.PutUint16(apdu, uint16(5+len(data)))
	apdu = append(apdu, []byte{0xe0, byte(opcode), byte(p1), byte(p2), byte(len(data))}...)
	apdu = append(apdu, data...)

	// Stream all the chunks to the device
	header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
	chunk := make([]byte, 64)
	space := len(chunk) - len(header)

	for i := 0; len(apdu) > 0; i++ {
		// Construct the new message to stream
		chunk = append(chunk[:0], header...)
		binary.BigEndian.PutUint16(chunk[3:], uint16(i))

		if len(apdu) > space {
			chunk = append(chunk, apdu[:space]...)
			apdu = apdu[space:]
		} else {
			chunk = append(chunk, apdu...)
			apdu = nil
		}
		// Send over to the device
		// w.log.Trace("Data chunk sent to the Ledger", "chunk", hexutil.Bytes(chunk))
		if _, err := w.device.Write(chunk); err != nil {
			return nil, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	chunk = chunk[:64] // Yeah, we surely have enough space
	for {
		// Read the next chunk from the Ledger wallet
		if _, err := io.ReadFull(w.device, chunk); err != nil {
			return nil, err
		}
		// w.log.Trace("Data chunk received from the Ledger", "chunk", hexutil.Bytes(chunk))

		// Make sure the transport header matches
		if chunk[0] != 0x01 || chunk[1] != 0x01 || chunk[2] != 0x05 {
			return nil, errLedgerReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the total message length
		var payload []byte

		if chunk[3] == 0x00 && chunk[4] == 0x00 {
			reply = make([]byte, 0, int(binary.BigEndian.Uint16(chunk[5:7])))
			payload = chunk[7:]
		} else {
			payload = chunk[5:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	return reply[:len(reply)-2], nil
}
