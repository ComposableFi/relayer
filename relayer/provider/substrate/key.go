package substrate

import (
	"github.com/cosmos/go-bip39"
	"github.com/cosmos/relayer/relayer/provider"
	"github.com/cosmos/relayer/relayer/provider/substrate/keystore"
)

func (sp *SubstrateProvider) CreateKeystore(path string) error {
	keybase, err := keystore.New(sp.Config.ChainID, sp.Config.KeyringBackend, sp.Config.KeyDirectory, sp.Input)
	if err != nil {
		return err
	}
	sp.Keybase = keybase
	return nil
}

func (sp *SubstrateProvider) KeystoreCreated(path string) bool {
	if sp.Keybase == nil {
		return false
	}
	return true
}

func (sp *SubstrateProvider) AddKey(name string) (output *provider.KeyOutput, err error) {
	ko, err := sp.KeyAddOrRestore(name, 118)
	if err != nil {
		return nil, err
	}
	return ko, nil
}

func (sp *SubstrateProvider) RestoreKey(name, mnemonic string) (address string, err error) {
	ko, err := sp.KeyAddOrRestore(name, 118, mnemonic)
	if err != nil {
		return "", err
	}
	return ko.Address, nil
}

func (sp *SubstrateProvider) ShowAddress(name string) (address string, err error) {
	info, err := sp.Keybase.Key(name)
	if err != nil {
		return "", err
	}
	return info.GetAddress(), nil
}

func (sp *SubstrateProvider) ListAddresses() (map[string]string, error) {
	out := map[string]string{}
	info, err := sp.Keybase.List()
	if err != nil {
		return nil, err
	}
	for _, k := range info {
		addr := k.GetAddress()
		if err != nil {
			return nil, err
		}
		out[k.GetName()] = addr
	}
	return out, nil
}

func (sp *SubstrateProvider) DeleteKey(name string) error {
	if err := sp.Keybase.Delete(name); err != nil {
		return err
	}
	return nil
}

func (sp *SubstrateProvider) KeyExists(name string) bool {
	k, err := sp.Keybase.Key(name)
	if err != nil {
		return false
	}
	return k.GetName() == name
}

func (sp *SubstrateProvider) ExportPrivKeyArmor(keyName string) (armor string, err error) {
	// TODO
	return "", nil
}

func (cc *SubstrateProvider) KeyAddOrRestore(keyName string, coinType uint32, mnemonic ...string) (*provider.KeyOutput, error) {
	var mnemonicStr string
	var err error

	if len(mnemonic) > 0 {
		mnemonicStr = mnemonic[0]
	} else {
		mnemonicStr, err = CreateMnemonic()
		if err != nil {
			return nil, err
		}
	}

	info, err := cc.Keybase.NewAccount(keyName, mnemonicStr, network)
	if err != nil {
		return nil, err
	}

	return &provider.KeyOutput{Mnemonic: mnemonicStr, Address: info.GetAddress()}, nil
}

// CreateMnemonic creates a new mnemonic
func CreateMnemonic() (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}