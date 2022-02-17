package substrate

// import (
// 	"errors"

// 	"github.com/vedhavyas/go-subkey"
// )

// type Keyring map[string]subkey.KeyPair

const network = 42

// func (k *Keyring) isNil() bool {
// 	if k == nil {
// 		return true
// 	}
// 	return false
// }

// func (k *Keyring) keyExists(name string) bool {
// 	if k.isNil() {
// 		return false
// 	}
// 	_, ok := (*k)[name]
// 	if ok {
// 		return true
// 	}
// 	return false
// }

// func (k *Keyring) getKey(name string) (subkey.KeyPair, error) {
// 	if k.isNil() {
// 		return nil, errors.New("keystore not initialized")
// 	}
// 	keyPair, ok := (*k)[name]
// 	if !ok {
// 		return nil, errors.New("key not found")
// 	}
// 	return keyPair, nil
// }

// func (k *Keyring) deleteKey(name string) error {
// 	if k.isNil() {
// 		return errors.New("keystore not initialized")
// 	}
// 	_, ok := (*k)[name]
// 	if !ok {
// 		return errors.New("key not found")
// 	}
// 	delete(*k, name)
// 	return nil
// }
