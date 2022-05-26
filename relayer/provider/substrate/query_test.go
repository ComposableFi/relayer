package substrate_test

import (
	"context"
	"testing"

	"github.com/cosmos/relayer/v2/relayer/provider/substrate"
)

var provider *substrate.SubstrateProvider

func init() {
	provider = initProvider()
}

func initProvider() *substrate.SubstrateProvider {
	p, err := substrate.NewSubstrateProvider(&substrate.SubstrateProviderConfig{RPCAddr: "ws://127.0.0.1:9944"}, "")
	if err != nil {
		panic(err)
	}
	return p
}

func TestQueryLatestHeight(t *testing.T) {
	height, err := provider.QueryLatestHeight(context.Background())
	if err != nil {
		panic(err)
	}

	if height <= 0 {
		t.Errorf("latest height should be greater than genesis height")
	}
}

func TestQueryHeaderAtHeight(t *testing.T) {
	finalizedBlockHash, err := provider.RPCClient.RPC.Chain.GetFinalizedHead()
	if err != nil {
		t.Error(err)
		return
	}

	_, err = provider.RPCClient.RPC.Chain.GetHeader(finalizedBlockHash)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = provider.QueryHeaderAtHeight(nil, int64(461))
	if err != nil {
		t.Error(err)
		return
	}
}
