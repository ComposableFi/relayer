package substrate

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v2/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v2/modules/core/exported"
	"github.com/cosmos/relayer/relayer/provider"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func (sp *SubstrateProvider) QueryTx(hashHex string) (*ctypes.ResultTx, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryTxs(page, limit int, events []string) ([]*ctypes.ResultTx, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryLatestHeight() (int64, error) {
	return 0, nil
}

func (sp *SubstrateProvider) QueryHeaderAtHeight(height int64) (ibcexported.Header, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryBalance(keyName string) (sdk.Coins, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryBalanceWithAddress(addr string) (sdk.Coins, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryUnbondingPeriod() (time.Duration, error) {
	return 0, nil
}

func (sp *SubstrateProvider) QueryClientState(height int64, clientid string) (ibcexported.ClientState, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryClientStateResponse(height int64, srcClientId string) (*clienttypes.QueryClientStateResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryClientConsensusState(chainHeight int64, clientid string, clientHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryUpgradedClient(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryUpgradedConsState(height int64) (*clienttypes.QueryConsensusStateResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryConsensusState(height int64) (ibcexported.ConsensusState, int64, error) {
	return nil, 0, nil
}

func (sp *SubstrateProvider) QueryClients() (clienttypes.IdentifiedClientStates, error) {
	return nil, nil
}

func (sp *SubstrateProvider) AutoUpdateClient(dst provider.ChainProvider, thresholdTime time.Duration, srcClientId, dstClientId string) (time.Duration, error) {
	return 0, nil
}

func (sp *SubstrateProvider) FindMatchingClient(counterparty provider.ChainProvider, clientState ibcexported.ClientState) (string, bool) {
	return "", false
}

func (sp *SubstrateProvider) QueryConnection(height int64, connectionid string) (*conntypes.QueryConnectionResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryConnections() (conns []*conntypes.IdentifiedConnection, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryConnectionsUsingClient(height int64, clientid string) (*conntypes.QueryConnectionsResponse, error) {
	return nil, nil
}

func (sp *SubstrateProvider) GenerateConnHandshakeProof(height int64, clientId, connId string) (clientState ibcexported.ClientState, clientStateProof []byte, consensusProof []byte, connectionProof []byte, connectionProofHeight ibcexported.Height, err error) {
	return nil, nil, nil, nil, nil, nil
}

func (sp *SubstrateProvider) NewClientState(dstUpdateHeader ibcexported.Header, dstTrustingPeriod, dstUbdPeriod time.Duration, allowUpdateAfterExpiry, allowUpdateAfterMisbehaviour bool) (ibcexported.ClientState, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryChannel(height int64, channelid, portid string) (chanRes *chantypes.QueryChannelResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryChannelClient(height int64, channelid, portid string) (*clienttypes.IdentifiedClientState, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryConnectionChannels(height int64, connectionid string) ([]*chantypes.IdentifiedChannel, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryChannels() ([]*chantypes.IdentifiedChannel, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryPacketCommitments(height uint64, channelid, portid string) (commitments *chantypes.QueryPacketCommitmentsResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryPacketAcknowledgements(height uint64, channelid, portid string) (acknowledgements []*chantypes.PacketState, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryUnreceivedPackets(height uint64, channelid, portid string, seqs []uint64) ([]uint64, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryUnreceivedAcknowledgements(height uint64, channelid, portid string, seqs []uint64) ([]uint64, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryNextSeqRecv(height int64, channelid, portid string) (recvRes *chantypes.QueryNextSequenceReceiveResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryPacketCommitment(height int64, channelid, portid string, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryPacketAcknowledgement(height int64, channelid, portid string, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryPacketReceipt(height int64, channelid, portid string, seq uint64) (recRes *chantypes.QueryPacketReceiptResponse, err error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryDenomTrace(denom string) (*transfertypes.DenomTrace, error) {
	return nil, nil
}

func (sp *SubstrateProvider) QueryDenomTraces(offset, limit uint64, height int64) ([]transfertypes.DenomTrace, error) {
	return nil, nil
}
