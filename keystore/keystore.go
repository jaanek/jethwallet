package keystore

import (
	"bufio"
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jaanek/jethwallet/ui"
)

var (
	ErrLocked  = accounts.NewAuthNeededError("password or unlock")
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given password")

	// ErrAccountAlreadyExists is returned if an account attempted to import is
	// already present in the keystore.
	ErrAccountAlreadyExists = errors.New("account already exists")
)

// KeyStoreScheme is the protocol scheme prefixing account and wallet URLs.
const KeyStoreScheme = "keystore"

type KeyStore struct {
	ui      ui.Screen
	keydir  string
	storage keyStore // Storage backend, might be cleartext or encrypted
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore(ui ui.Screen, keydir string) *KeyStore {
	scryptN, scryptP := keystore.StandardScryptN, keystore.StandardScryptP
	keydir, _ = filepath.Abs(keydir)
	ks := &KeyStore{
		ui:      ui,
		keydir:  keydir,
		storage: &keyStorePassphrase{keydir, scryptN, scryptP, false}}
	return ks
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() ([]*accounts.Account, error) {
	// List all the files from the keystore folder
	files, err := ioutil.ReadDir(ks.keydir)
	if err != nil {
		return nil, err
	}
	accounts := []*accounts.Account{}
	for _, fi := range files {
		path := ks.storage.JoinPath(fi.Name())
		// Skip any non-key files from the folder
		if nonKeyFile(fi) {
			fmt.Printf("Ignoring file on account scan: %s\n", path)
			continue
		}
		acc, err := readAccount(path)
		if err != nil {
			fmt.Printf("Error while reading keystore account from path: %s, %v\n", path, err)
			continue
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// SignTx signs the given transaction with the requested address.
func (ks *KeyStore) SignTx(addr common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	acc, err := ks.FindOne(addr)
	if err != nil {
		return nil, err
	}
	ks.ui.Print("*** Enter passphrase (not echoed)...")
	passphrase, err := ks.ui.ReadPassword(&acc)
	if err != nil {
		return nil, err
	}
	key, err := ks.getDecryptedKey(acc, string(passphrase))
	if err != nil {
		return nil, err
	}
	fmt.Printf("Account key found from path: %s\n", acc.URL.Path)

	// Depending on the presence of the chain ID, sign with 2718 or homestead
	signer := types.LatestSignerForChainID(chainID)
	return types.SignTx(tx, signer, key.PrivateKey)
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (ks *KeyStore) NewAccount(passphrase string) (accounts.Account, error) {
	_, account, err := storeNewKey(ks.storage, crand.Reader, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}
	return account, nil
}

// ImportECDSA stores the given key into the key directory, encrypting it with the passphrase.
func (ks *KeyStore) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (accounts.Account, error) {
	key := newKeyFromECDSA(priv)
	accs, err := ks.Find(key.Address)
	if err != nil {
		return accounts.Account{}, err
	}
	if len(accs) > 0 {
		return accounts.Account{
			Address: key.Address,
		}, ErrAccountAlreadyExists
	}
	return ks.importKey(key, passphrase)
}

func (ks *KeyStore) importKey(key *Key, passphrase string) (accounts.Account, error) {
	acc := accounts.Account{
		Address: key.Address,
		URL: accounts.URL{
			Scheme: KeyStoreScheme,
			Path:   ks.storage.JoinPath(keyFileName(key.Address)),
		},
	}
	if err := ks.storage.StoreKey(acc.URL.Path, key, passphrase); err != nil {
		return accounts.Account{}, err
	}
	return acc, nil
}

// find all accounts with an address in keystore
func (ks *KeyStore) Find(a common.Address) ([]accounts.Account, error) {
	accs, err := ks.Accounts()
	if err != nil {
		return nil, err
	}
	found := []accounts.Account{}
	for _, acc := range accs {
		if acc.Address == a {
			found = append(found, *acc)
		}
	}
	return found, nil
}

func (ks *KeyStore) FindOne(a common.Address) (accounts.Account, error) {
	accs, err := ks.Find(a)
	if err != nil {
		return accounts.Account{}, err
	}
	if len(accs) == 0 {
		return accounts.Account{}, errors.New(fmt.Sprintf("No accounts found for address: %v", a))
	}
	if len(accs) > 1 {
		return accounts.Account{}, errors.New(fmt.Sprintf("Found %d accounts for address: %v", len(accs), a))
	}
	return accs[0], nil
}

func (ks *KeyStore) getDecryptedKey(acc accounts.Account, auth string) (*Key, error) {
	key, err := ks.storage.GetKey(acc.Address, acc.URL.Path, auth)
	return key, err
}

// nonKeyFile ignores editor backups, hidden files and folders/symlinks.
func nonKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}

func readAccount(path string) (*accounts.Account, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to open keystore file: %s\n, err: %v", path, err))
	}
	defer fd.Close()

	var (
		buf = new(bufio.Reader)
		key struct {
			Address string `json:"address"`
		}
	)
	buf.Reset(fd)

	// Parse the address.
	err = json.NewDecoder(buf).Decode(&key)
	addr := common.HexToAddress(key.Address)
	switch {
	case err != nil:
		return nil, errors.New(fmt.Sprintf("Failed to decode keystore key: %s, err: %v\n", path, err))
	case addr == common.Address{}:
		return nil, errors.New(fmt.Sprintf("Failed to decode keystore key: %s, err: %s\n", path, "missing or zero address"))
	}
	return &accounts.Account{
		Address: addr,
		URL:     accounts.URL{Scheme: "keystore", Path: path},
	}, nil
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}
