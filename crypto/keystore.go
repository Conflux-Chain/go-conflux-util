package crypto

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"

	commonutil "github.com/Conflux-Chain/go-conflux-util/common"
	gethks "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
)

const (
	// Scrypt encryption algorithm parameters
	keyStoreEncryptionScryptN = gethks.StandardScryptN
	keyStoreEncryptionScryptP = gethks.StandardScryptP
)

// KeyStoreManager manages a key storage directory on disk. It will store
// key partitions distibuted into different sub directories by key hashing.
type KeyStoreManager struct {
	keydir     string // keystore directory
	partitions uint32 // number of partitions for distribution
}

// NewKeyStoreManager creates a keystore manager for the given directory.
func NewKeyStoreManager(keydir string, partitions uint32) (*KeyStoreManager, error) {
	keydir, err := filepath.Abs(keydir)
	if err != nil {
		return nil, errors.WithMessagef(err, "bad keystore dir %v", keydir)
	}

	return &KeyStoreManager{keydir: keydir, partitions: partitions}, nil
}

// HasKey checks if key file of specified identifier exists in keystore.
func (ks *KeyStoreManager) HasKey(kid PrivKeyIdentifier) (bool, error) {
	kp := ks.keyPath(kid)

	fileStat, err := os.Stat(kp)
	if err != nil {
		return false, errors.WithMessagef(err, "failed to check file stats for %v", kp)
	}

	return fileStat.Size() != 0, nil
}

// GetKey loads and decrypts the key from disk.
func (ks *KeyStoreManager) GetKey(kid PrivKeyIdentifier, auth string) (*gethks.Key, error) {
	kpath := ks.keyPath(kid)

	// Load the key from the keystore and decrypt its contents
	keyjson, err := ioutil.ReadFile(kpath)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to read key path %v", kpath)
	}

	key, err := gethks.DecryptKey(keyjson, auth)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to decrypt key file")
	}

	return key, nil
}

// StoreKey writes and encrypts the key.
func (ks *KeyStoreManager) StoreKey(kid PrivKeyIdentifier, key *gethks.Key, auth string) error {
	// Encrypt key content and store the key to the keystore.
	keyjson, err := gethks.EncryptKey(key, auth, keyStoreEncryptionScryptN, keyStoreEncryptionScryptP)
	if err != nil {
		return errors.WithMessage(err, "failed to encrypt key")
	}

	kpath := ks.keyPath(kid)

	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(kpath), dirPerm); err != nil {
		return errors.WithMessagef(err, "failed to create directory for %v", kpath)
	}

	err = commonutil.WriteFileAtomically(kpath, keyjson)
	return errors.WithMessagef(err, "failed to write to keyfile %v atomically", kpath)
}

// keyPath generates key file path with specified indentifier.
func (ks *KeyStoreManager) keyPath(kid PrivKeyIdentifier) string {
	partition := ks.hashToParition(kid)
	return filepath.Join(ks.keydir, fmt.Sprint(partition), string(kid))
}

// hashToParition allocates partitons for specified indentifier by hashing
// it and moding result hash to the total partitions.
func (ks *KeyStoreManager) hashToParition(kid PrivKeyIdentifier) uint32 {
	h := fnv.New32a()
	h.Write([]byte(kid))

	return h.Sum32() % ks.partitions
}
