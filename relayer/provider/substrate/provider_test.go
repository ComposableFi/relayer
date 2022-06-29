package substrate_test

import (
	"context"
	"fmt"
	rpcClient "github.com/ComposableFi/go-substrate-rpc-client/v4"
	rpcClientTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/ComposableFi/go-substrate-rpc-client/v4/signature"
	"github.com/cosmos/relayer/v2/relayer/provider/substrate"
	"github.com/cosmos/relayer/v2/relayer/provider/substrate/keystore"
	"github.com/stretchr/testify/require"
)

const homePath = "/tmp"
const rpcAddress = "ws://127.0.0.1:9988"

func TestGetTrustingPeriod(t *testing.T) {
	testProvider, err := getTestProvider()
	require.NoError(t, err)
	tp, err := testProvider.TrustingPeriod(context.Background())
	require.NoError(t, err)
	require.NotNil(t, tp)
}

func getSubstrateConfig(keyHome string, debug bool) *substrate.SubstrateProviderConfig {
	return &substrate.SubstrateProviderConfig{
		Key:            "default",
		ChainID:        "substrate-test",
		RPCAddr:        rpcAddress,
		KeyringBackend: keystore.BackendTest,
		KeyDirectory:   keyHome,
		Timeout:        "20s",
	}
}

func getTestProvider() (*substrate.SubstrateProvider, error) {
	testProvider, err := substrate.NewSubstrateProvider(getSubstrateConfig(homePath, true), "")
	if err != nil {
		return nil, err
	}

	err = testProvider.Init()
	if err != nil {
		return nil, err
	}
	return testProvider, err

}

func TestBalanceTransfer(t *testing.T) {
	api, err := rpcClient.NewSubstrateAPI(rpcAddress)
	assert.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	assert.NoError(t, err)

	bob, err := rpcClientTypes.NewMultiAddressFromHexAccountID("0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48")
	assert.NoError(t, err)

	amount := rpcClientTypes.NewUCompactFromUInt(12345)
	c, err := rpcClientTypes.NewCall(meta, "Balances.transfer", bob, amount)
	assert.NoError(t, err)

	// Create the extrinsic
	ext := rpcClientTypes.NewExtrinsic(c)
	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	assert.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	assert.NoError(t, err)

		// Get the nonce for Alice
	key, err := rpcClientTypes.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey)
	assert.NoError(t, err)

	var accountInfo rpcClientTypes.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	assert.NoError(t, err)
	assert.True(t, ok)
	nonce := uint32(accountInfo.Nonce)
	o := rpcClientTypes.SignatureOptions{
		BlockHash:   genesisHash,
		Era:         rpcClientTypes.ExtrinsicEra{IsMortalEra: false},
		GenesisHash: genesisHash,
		Nonce:       rpcClientTypes.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion: rv.SpecVersion,
		Tip:         rpcClientTypes.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	assert.NoError(t, err)

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		assert.NoError(t, err)
	}

	t.Log(hash.Hex())
	require.NoError(t, err)
}

func TestBalances(t *testing.T) {
	// Instantiate the API
	api, err := rpcClient.NewSubstrateAPI(rpcAddress)
	assert.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	assert.NoError(t, err)

	// Create a call, transferring 12345 units to Bob
	bob, err := rpcClientTypes.NewMultiAddressFromHexAccountID("0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48")
	assert.NoError(t, err)

	amount := rpcClientTypes.NewUCompactFromUInt(12345)
	c, err := rpcClientTypes.NewCall(meta, "Balances.transfer", bob, amount)
	assert.NoError(t, err)

	for {
		// Create the extrinsic
		ext := rpcClientTypes.NewExtrinsic(c)
		genesisHash, err := api.RPC.Chain.GetBlockHash(0)
		assert.NoError(t, err)

		rv, err := api.RPC.State.GetRuntimeVersionLatest()
		assert.NoError(t, err)

		// Get the nonce for Alice
		key, err := rpcClientTypes.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey)
		assert.NoError(t, err)

		var accountInfo rpcClientTypes.AccountInfo
		ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
		assert.NoError(t, err)
		assert.True(t, ok)
		nonce := uint32(accountInfo.Nonce)
		o := rpcClientTypes.SignatureOptions{
			BlockHash:          genesisHash,
			Era:                rpcClientTypes.ExtrinsicEra{IsMortalEra: false},
			GenesisHash:        genesisHash,
			Nonce:              rpcClientTypes.NewUCompactFromUInt(uint64(nonce)),
			SpecVersion:        rv.SpecVersion,
			Tip:                rpcClientTypes.NewUCompactFromUInt(0),
			TransactionVersion: rv.TransactionVersion,
		}

		fmt.Printf("Sending %v from %#x to %#x with nonce %v\n", amount, signature.TestKeyringPairAlice.PublicKey,
			bob.AsID, nonce)

		// Sign the transaction using Alice's default account
		err = ext.Sign(signature.TestKeyringPairAlice, o)
		assert.NoError(t, err)

		res, err := api.RPC.Author.SubmitExtrinsic(ext)
		if err != nil {
			t.Logf("extrinsic submit failed: %v", err)
			continue
		}

		hex, err := rpcClientTypes.Hex(res)
		assert.NoError(t, err)
		assert.NotEmpty(t, hex)
		break
	}
}
