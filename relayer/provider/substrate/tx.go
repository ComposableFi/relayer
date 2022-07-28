package substrate

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ComposableFi/go-substrate-rpc-client/v4/signature"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	conntypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	beefyclient "github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"github.com/cosmos/relayer/v2/relayer/provider/cosmos"
	"github.com/gogo/protobuf/proto"
	"time"

	rpcClient "github.com/ComposableFi/go-substrate-rpc-client/v4"
	rpcClientTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/cosmos/relayer/v2/relayer/provider/substrate/keystore"
)

var (
	defaultChainPrefix = commitmenttypes.NewMerklePrefix([]byte("ibc"))
	defaultDelayPeriod = uint64(0)
)

func (sp *SubstrateProvider) Init() error {
	keybase, err := keystore.New(sp.Config.ChainID, sp.Config.KeyringBackend, sp.Config.KeyDirectory, sp.Input)
	if err != nil {
		return err
	}

	client, err := rpcClient.NewSubstrateAPI(sp.Config.RPCAddr)
	if err != nil {
		return err
	}

	lightClient, err := rpcClient.NewSubstrateAPI(sp.Config.LightClientRPCAddr)
	if err != nil {
		return err
	}

	sp.Keybase = keybase

	sp.RPCClient = client
	sp.LightClient = lightClient
	sp.BlockHistory = make(map[uint64]uint64)
	sp.fetchNewBlocks()
	fmt.Printf("### done fetching new blocks ========> \n")
	return nil
}

func (sp *SubstrateProvider) fetchNewBlocks() {
	fmt.Printf("### fetching new blocks ========> \n")
	ch := make(chan interface{})
	sub, err := sp.LightClient.Client.Subscribe(
		context.Background(),
		"beefy",
		"subscribeJustifications",
		"unsubscribeJustifications",
		"justifications",
		ch,
	)
	if err != nil {
		fmt.Printf("error subscribing to justifications")
		return
	}


	var beefyClientState *beefyclient.ClientState
	defer sub.Unsubscribe()
	for {
		fmt.Printf("### waiting for blocks ========> \n")
		msg := <-ch
		compactCommitment := rpcClientTypes.CompactSignedCommitment{}
		err = rpcClientTypes.DecodeFromHexString(msg.(string), &compactCommitment)
		if err != nil {
			fmt.Printf("error decoding: %v \n", err.Error())
			return
		}

		commitment := compactCommitment.Unpack()
		if beefyClientState == nil {
			beefyClientState, err = clientState(sp.LightClient, commitment)
			if err != nil {
				fmt.Printf("error creating client state: %v \n", err.Error())
				return
			}
			continue
		}

		blockHash, err :=  sp.LightClient.RPC.Chain.GetBlockHash(uint64(commitment.Commitment.BlockNumber))
		if err != nil {
			fmt.Printf("error getting block hash: %v \n", err.Error())
			return
		}

		parachainHeads, err := sp.constructParachainHeaders(blockHash, beefyClientState)
		if err != nil {
			fmt.Printf("construct parachain headers error: %v \n", err.Error())
			return
		}

		for _, h := range parachainHeads {
			header, err := beefyclient.DecodeParachainHeader(h.ParachainHeader)
			if err != nil {
				panic(fmt.Errorf("failed to decode parachain header"))
			}
			sp.BlockHistory[uint64(header.Number)] = uint64(commitment.Commitment.BlockNumber)
		}

		fmt.Printf("## updating block history %+v \n", sp.BlockHistory)
		beefyClientState, err = clientState(sp.LightClient, commitment)
		if err != nil {
			fmt.Printf("error constructing client state: %v \n", err.Error())
			return
		}
	}
}

func (sp *SubstrateProvider) CreateClient(clientState ibcexported.ClientState, dstHeader ibcexported.ClientMessage, signer string) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	// TODO: fix head decoding error in validate basic
	if err := dstHeader.ValidateBasic(); err != nil {
		return nil, err
	}

	beefyHeader, ok := dstHeader.(*beefyclient.Header)
	if !ok {
		return nil, fmt.Errorf("got data of type %T but wanted beefyclient.Header \n", dstHeader)
	}

	acc = signer
	if acc == "" {
		acc, err = sp.Address()
		if  err != nil {
			return nil, err
		}
	}


	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}

	anyConsensusState, err := clienttypes.PackConsensusState(beefyHeader.ConsensusState())
	if err != nil {
		return nil, err
	}

	msg := &clienttypes.MsgCreateClient{
		ClientState:    anyClientState,
		ConsensusState: anyConsensusState,
		Signer:         acc,
	}
	//return NewSubstrateRelayerMessage(msg), nil
	// TODO: return appropriate message type for target chain or create message translator
	return cosmos.NewCosmosMessage(msg), nil
}

func (sp *SubstrateProvider) SubmitMisbehavior( /*TODO TBD*/ ) (provider.RelayerMessage, error) {
	return nil, nil
}

func (sp *SubstrateProvider) UpdateClient(srcClientId string, dstHeader ibcexported.ClientMessage) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if err := dstHeader.ValidateBasic(); err != nil {
		return nil, err
	}
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	anyHeader, err := clienttypes.PackClientMessage(dstHeader)
	if err != nil {
		return nil, err
	}

	msg := &clienttypes.MsgUpdateClient{
		ClientId: srcClientId,
		ClientMessage:   anyHeader,
		Signer:   acc,
	}

	return NewSubstrateRelayerMessage(msg), nil
}

func (sp *SubstrateProvider) ConnectionOpenInit(srcClientId, dstClientId string, dstHeader ibcexported.ClientMessage) ([]provider.RelayerMessage, error) {
	var (
		acc     string
		err     error
		version *conntypes.Version
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	counterparty := conntypes.Counterparty{
		ClientId:     dstClientId,
		ConnectionId: "",
		Prefix:       defaultChainPrefix,
	}
	msg := &conntypes.MsgConnectionOpenInit{
		ClientId:     srcClientId,
		Counterparty: counterparty,
		Version:      version,
		DelayPeriod:  defaultDelayPeriod,
		Signer:       acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ConnectionOpenTry(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, srcClientId, dstClientId, srcConnId, dstConnId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}

	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	clientState, clientStateProof, consensusStateProof, connStateProof, proofHeight, err := dstQueryProvider.GenerateConnHandshakeProof(ctx, cph, dstClientId, dstConnId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	csAny, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}

	counterparty := conntypes.Counterparty{
		ClientId:     dstClientId,
		ConnectionId: dstConnId,
		Prefix:       defaultChainPrefix,
	}

	// TODO: Get DelayPeriod from counterparty connection rather than using default value
	msg := &conntypes.MsgConnectionOpenTry{
		ClientId:             srcClientId,
		PreviousConnectionId: srcConnId,
		ClientState:          csAny,
		Counterparty:         counterparty,
		DelayPeriod:          defaultDelayPeriod,
		CounterpartyVersions: conntypes.ExportedVersionsToProto(conntypes.GetCompatibleVersions()),
		ProofHeight: clienttypes.Height{
			RevisionNumber: proofHeight.GetRevisionNumber(),
			RevisionHeight: proofHeight.GetRevisionHeight(),
		},
		ProofInit:       connStateProof,
		ProofClient:     clientStateProof,
		ProofConsensus:  consensusStateProof,
		ConsensusHeight: clientState.GetLatestHeight().(clienttypes.Height),
		Signer:          acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ConnectionOpenAck(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, srcClientId, srcConnId, dstClientId, dstConnId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)

	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}
	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	clientState, clientStateProof, consensusStateProof, connStateProof,
		proofHeight, err := dstQueryProvider.GenerateConnHandshakeProof(ctx, cph, dstClientId, dstConnId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	csAny, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}

	msg := &conntypes.MsgConnectionOpenAck{
		ConnectionId:             srcConnId,
		CounterpartyConnectionId: dstConnId,
		Version:                  conntypes.DefaultIBCVersion,
		ClientState:              csAny,
		ProofHeight: clienttypes.Height{
			RevisionNumber: proofHeight.GetRevisionNumber(),
			RevisionHeight: proofHeight.GetRevisionHeight(),
		},
		ProofTry:        connStateProof,
		ProofClient:     clientStateProof,
		ProofConsensus:  consensusStateProof,
		ConsensusHeight: clientState.GetLatestHeight().(clienttypes.Height),
		Signer:          acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ConnectionOpenConfirm(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, dstConnId, srcClientId, srcConnId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}

	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}
	counterpartyConnState, err := dstQueryProvider.QueryConnection(ctx, cph, dstConnId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &conntypes.MsgConnectionOpenConfirm{
		ConnectionId: srcConnId,
		ProofAck:     counterpartyConnState.Proof,
		ProofHeight:  counterpartyConnState.ProofHeight,
		Signer:       acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ChannelOpenInit(srcClientId, srcConnId, srcPortId, srcVersion, dstPortId string, order chantypes.Order, dstHeader ibcexported.ClientMessage) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelOpenInit{
		PortId: srcPortId,
		Channel: chantypes.Channel{
			State:    chantypes.INIT,
			Ordering: order,
			Counterparty: chantypes.Counterparty{
				PortId:    dstPortId,
				ChannelId: "",
			},
			ConnectionHops: []string{srcConnId},
			Version:        srcVersion,
		},
		Signer: acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ChannelOpenTry(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, srcPortId, dstPortId, srcChanId, dstChanId, srcVersion, srcConnectionId, srcClientId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}
	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	counterpartyChannelRes, err := dstQueryProvider.QueryChannel(ctx, cph, dstChanId, dstPortId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelOpenTry{
		PortId:            srcPortId,
		PreviousChannelId: srcChanId,
		Channel: chantypes.Channel{
			State:    chantypes.TRYOPEN,
			Ordering: counterpartyChannelRes.Channel.Ordering,
			Counterparty: chantypes.Counterparty{
				PortId:    dstPortId,
				ChannelId: dstChanId,
			},
			ConnectionHops: []string{srcConnectionId},
			Version:        srcVersion,
		},
		CounterpartyVersion: counterpartyChannelRes.Channel.Version,
		ProofInit:           counterpartyChannelRes.Proof,
		ProofHeight:         counterpartyChannelRes.ProofHeight,
		Signer:              acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ChannelOpenAck(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, srcClientId, srcPortId, srcChanId, dstChanId, dstPortId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}

	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	counterpartyChannelRes, err := dstQueryProvider.QueryChannel(ctx, cph, dstChanId, dstPortId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelOpenAck{
		PortId:                srcPortId,
		ChannelId:             srcChanId,
		CounterpartyChannelId: dstChanId,
		CounterpartyVersion:   counterpartyChannelRes.Channel.Version,
		ProofTry:              counterpartyChannelRes.Proof,
		ProofHeight:           counterpartyChannelRes.ProofHeight,
		Signer:                acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ChannelOpenConfirm(ctx context.Context, dstQueryProvider provider.QueryProvider, dstHeader ibcexported.ClientMessage, srcClientId, srcPortId, srcChanId, dstPortId, dstChanId string) ([]provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	updateMsg, err := sp.UpdateClient(srcClientId, dstHeader)
	if err != nil {
		return nil, err
	}
	cph, err := dstQueryProvider.QueryLatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	counterpartyChanState, err := dstQueryProvider.QueryChannel(ctx, cph, dstChanId, dstPortId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelOpenConfirm{
		PortId:      srcPortId,
		ChannelId:   srcChanId,
		ProofAck:    counterpartyChanState.Proof,
		ProofHeight: counterpartyChanState.ProofHeight,
		Signer:      acc,
	}

	return []provider.RelayerMessage{updateMsg, NewSubstrateRelayerMessage(msg)}, nil
}

func (sp *SubstrateProvider) ChannelCloseInit(srcPortId, srcChanId string) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelCloseInit{
		PortId:    srcPortId,
		ChannelId: srcChanId,
		Signer:    acc,
	}

	return NewSubstrateRelayerMessage(msg), nil
}

func (sp *SubstrateProvider) ChannelCloseConfirm(ctx context.Context, dstQueryProvider provider.QueryProvider, dsth int64, dstChanId, dstPortId, srcPortId, srcChanId string) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	dstChanResp, err := dstQueryProvider.QueryChannel(ctx, dsth, dstChanId, dstPortId)
	if err != nil {
		return nil, err
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	msg := &chantypes.MsgChannelCloseConfirm{
		PortId:      srcPortId,
		ChannelId:   srcChanId,
		ProofInit:   dstChanResp.Proof,
		ProofHeight: dstChanResp.ProofHeight,
		Signer:      acc,
	}

	return NewSubstrateRelayerMessage(msg), nil
}

func (sp *SubstrateProvider) MsgRelayAcknowledgement(ctx context.Context, dst provider.ChainProvider, dstChanId, dstPortId, srcChanId, srcPortId string, dsth int64, packet provider.RelayPacket) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	msgPacketAck, ok := packet.(*relayMsgPacketAck)
	if !ok {
		return nil, fmt.Errorf("got data of type %T but wanted relayMsgPacketAck \n", packet)
	}

	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	ackRes, err := dst.QueryPacketAcknowledgement(ctx, dsth, dstChanId, dstPortId, packet.Seq())
	switch {
	case err != nil:
		return nil, err
	case ackRes.Proof == nil || ackRes.Acknowledgement == nil:
		return nil, fmt.Errorf("ack packet acknowledgement query seq(%d) is nil", packet.Seq())
	case ackRes == nil:
		return nil, fmt.Errorf("ack packet [%s]seq{%d} has no associated proofs", dst.ChainId(), packet.Seq())
	default:
		msg := &chantypes.MsgAcknowledgement{
			Packet: chantypes.Packet{
				Sequence:           packet.Seq(),
				SourcePort:         srcPortId,
				SourceChannel:      srcChanId,
				DestinationPort:    dstPortId,
				DestinationChannel: dstChanId,
				Data:               packet.Data(),
				TimeoutHeight:      packet.Timeout(),
				TimeoutTimestamp:   packet.TimeoutStamp(),
			},
			Acknowledgement: msgPacketAck.ack,
			ProofAcked:      ackRes.Proof,
			ProofHeight:     ackRes.ProofHeight,
			Signer:          acc,
		}

		return NewSubstrateRelayerMessage(msg), nil
	}
}

func (sp *SubstrateProvider) MsgTransfer(amount sdk.Coin, dstChainId, dstAddr, srcPortId, srcChanId string, timeoutHeight, timeoutTimestamp uint64) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	version := clienttypes.ParseChainID(dstChainId)

	msg := &transfertypes.MsgTransfer{
		SourcePort:    srcPortId,
		SourceChannel: srcChanId,
		Token:         amount,
		Sender:        acc,
		Receiver:      dstAddr,
		TimeoutHeight: clienttypes.Height{
			RevisionNumber: version,
			RevisionHeight: timeoutHeight,
		},
		TimeoutTimestamp: timeoutTimestamp,
	}

	return NewSubstrateRelayerMessage(msg), nil
}

func (sp *SubstrateProvider) MsgRelayTimeout(ctx context.Context, dst provider.ChainProvider, dsth int64, packet provider.RelayPacket, dstChanId, dstPortId, srcChanId, srcPortId string) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	recvRes, err := dst.QueryPacketReceipt(ctx, dsth, dstChanId, dstPortId, packet.Seq())
	switch {
	case err != nil:
		return nil, err
	case recvRes.Proof == nil:
		return nil, fmt.Errorf("timeout packet receipt proof seq(%d) is nil", packet.Seq())
	case recvRes == nil:
		return nil, fmt.Errorf("timeout packet [%s]seq{%d} has no associated proofs", sp.Config.ChainID, packet.Seq())
	default:
		msg := &chantypes.MsgTimeout{
			Packet: chantypes.Packet{
				Sequence:           packet.Seq(),
				SourcePort:         srcPortId,
				SourceChannel:      srcChanId,
				DestinationPort:    dstPortId,
				DestinationChannel: dstChanId,
				Data:               packet.Data(),
				TimeoutHeight:      packet.Timeout(),
				TimeoutTimestamp:   packet.TimeoutStamp(),
			},
			ProofUnreceived:  recvRes.Proof,
			ProofHeight:      recvRes.ProofHeight,
			NextSequenceRecv: packet.Seq(),
			Signer:           acc,
		}

		return NewSubstrateRelayerMessage(msg), nil
	}
}

func (sp *SubstrateProvider) MsgRelayRecvPacket(ctx context.Context, dst provider.ChainProvider, dsth int64, packet provider.RelayPacket, dstChanId, dstPortId, srcChanId, srcPortId string) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}

	comRes, err := dst.QueryPacketCommitment(ctx, dsth, dstChanId, dstPortId, packet.Seq())
	switch {
	case err != nil:
		return nil, err
	case comRes.Proof == nil || comRes.Commitment == nil:
		return nil, fmt.Errorf("recv packet commitment query seq(%d) is nil", packet.Seq())
	case comRes == nil:
		return nil, fmt.Errorf("receive packet [%s]seq{%d} has no associated proofs", sp.Config.ChainID, packet.Seq())
	default:
		msg := &chantypes.MsgRecvPacket{
			Packet: chantypes.Packet{
				Sequence:           packet.Seq(),
				SourcePort:         dstPortId,
				SourceChannel:      dstChanId,
				DestinationPort:    srcPortId,
				DestinationChannel: srcChanId,
				Data:               packet.Data(),
				TimeoutHeight:      packet.Timeout(),
				TimeoutTimestamp:   packet.TimeoutStamp(),
			},
			ProofCommitment: comRes.Proof,
			ProofHeight:     comRes.ProofHeight,
			Signer:          acc,
		}

		return NewSubstrateRelayerMessage(msg), nil
	}
}

func (sp *SubstrateProvider) MsgUpgradeClient(srcClientId string, consRes *clienttypes.QueryConsensusStateResponse, clientRes *clienttypes.QueryClientStateResponse) (provider.RelayerMessage, error) {
	var (
		acc string
		err error
	)
	if acc, err = sp.Address(); err != nil {
		return nil, err
	}
	return NewSubstrateRelayerMessage(&clienttypes.MsgUpgradeClient{ClientId: srcClientId, ClientState: clientRes.ClientState,
		ConsensusState: consRes.ConsensusState, ProofUpgradeClient: consRes.GetProof(),
		ProofUpgradeConsensusState: consRes.ConsensusState.Value, Signer: acc}), nil
}

func (sp *SubstrateProvider) RelayPacketFromSequence(ctx context.Context, src, dst provider.ChainProvider, srch, dsth, seq uint64, dstChanId, dstPortId, dstClientId string, srcChanId, srcPortId, srcClientId string) (provider.RelayerMessage, provider.RelayerMessage, error) {

	allPackets, err := sp.RPCClient.RPC.IBC.QueryPackets(srcClientId, dstPortId, []uint64{seq})
	switch {
	case err != nil:
		return nil, nil, err
	case len(allPackets) == 0:
		return nil, nil, fmt.Errorf("no transactions returned with query")
	case len(allPackets) > 1:
		return nil, nil, fmt.Errorf("more than one transaction returned with query")
	}

	rcvPackets, timeoutPackets, err := relayPacketsFromPacket(ctx, src, dst, int64(dsth), allPackets, dstChanId, dstPortId, srcChanId, srcPortId, srcClientId)
	switch {
	case err != nil:
		return nil, nil, err
	case len(rcvPackets) == 0 && len(timeoutPackets) == 0:
		return nil, nil, fmt.Errorf("no relay msgs created from query response")
	case len(rcvPackets)+len(timeoutPackets) > 1:
		return nil, nil, fmt.Errorf("more than one relay msg found in tx query")
	}

	if err != nil {
		return nil, nil, err
	}
	if len(rcvPackets) == 1 {
		pkt := rcvPackets[0]
		if seq != pkt.Seq() {
			return nil, nil, fmt.Errorf("wrong sequence: expected(%d) got(%d)", seq, pkt.Seq())
		}

		packet, err := dst.MsgRelayRecvPacket(ctx, src, int64(srch), pkt, srcChanId, srcPortId, dstChanId, dstPortId)
		if err != nil {
			return nil, nil, err
		}

		return packet, nil, nil
	}

	if len(timeoutPackets) == 1 {
		pkt := timeoutPackets[0]
		if seq != pkt.Seq() {
			return nil, nil, fmt.Errorf("wrong sequence: expected(%d) got(%d)", seq, pkt.Seq())
		}

		timeout, err := src.MsgRelayTimeout(ctx, dst, int64(dsth), pkt, dstChanId, dstPortId, srcChanId, srcPortId)
		if err != nil {
			return nil, nil, err
		}
		return nil, timeout, nil
	}

	return nil, nil, fmt.Errorf("should have errored before here")
}

func (sp *SubstrateProvider) AcknowledgementFromSequence(ctx context.Context, dst provider.ChainProvider, dsth, seq uint64, dstChanId, dstPortId, srcChanId, srcPortId string) (provider.RelayerMessage, error) {
	return nil, nil
}

func (sp *SubstrateProvider) SendMessage(ctx context.Context, msg provider.RelayerMessage) (*provider.RelayerTxResponse, bool, error) {
	return sp.SendMessages(ctx, []provider.RelayerMessage{msg})
}

func hexToByte(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return b
}

func (sp *SubstrateProvider) SendMessages(ctx context.Context, msgs []provider.RelayerMessage) (*provider.RelayerTxResponse, bool, error) {
	meta, err := sp.RPCClient.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, false, err
	}

	type Any struct {
		TypeUrl string `protobuf:"bytes,1,opt,name=type_url,json=typeUrl,proto3" json:"type_url,omitempty"`
		// Must be a valid serialized protocol buffer of the above specified type.
		Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	}

	msg := msgs[0]
	msgBytes, err := msg.MsgBytes()
	if err != nil {
		return nil, false, err
	}

	anyMsg := Any{
		TypeUrl: msg.Type(),
		Value: msgBytes,
	}

	var call string
	if msg.Type() == ("/" + proto.MessageName(&clienttypes.MsgCreateClient{})) {
		call = "Ibc.create_client"
	} else {
		call = "Ibc.deliver"
	}
	// call = "Ibc.deliver"
	c, err := rpcClientTypes.NewCall(meta, call, anyMsg)
	if err != nil {
		return nil, false, err
	}

	sc, err := rpcClientTypes.NewCall(meta, "Sudo.sudo", c)
	if err != nil {
		return nil, false, err
	}

	// Create the extrinsic
	ext := rpcClientTypes.NewExtrinsic(sc)

	genesisHash, err := sp.RPCClient.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, false, err
	}

	rv, err := sp.RPCClient.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, false, err
	}

	//info, err := sp.Keybase.Key(sp.Key())
	//if err != nil {
	//	return nil, false, err
	//}

	// fmt.Printf("\n KEY %v ADDRESS %v URI %v  PUBKEY %+v\n", sp.Key(), info.GetAddress(), info.GetKeyringPair().URI, info.GetPublicKey())
	// TODO: remove when cleaning up
	devAccountPubKey := hexToByte("d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	devAccountAddress := "5yNZjX24n2eg7W6EVamaTXNQbWCwchhThEaSWB7V3GRjtHeL"
	key, err := rpcClientTypes.CreateStorageKey(meta, "System", "Account", devAccountPubKey, nil)
	if err != nil {
		return nil, false, err
	}

	// TODO: remove when cleaning up
	keyring := signature.TestKeyringPairAlice
	keyring.PublicKey = devAccountPubKey
	keyring.Address = devAccountAddress

	var accountInfo rpcClientTypes.AccountInfo
	ok, err := sp.RPCClient.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return nil, false, err
	}

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

	//err = ext.Sign(info.GetKeyringPair(), o)
	err = ext.Sign(keyring, o)
	if err != nil {
		return nil, false, err
	}

	// Send the extrinsic
	sub, err := sp.RPCClient.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		fmt.Printf("Extrinsic Error: %v \n", err.Error())
		return nil, false, err
	}

	var status rpcClientTypes.ExtrinsicStatus
	defer sub.Unsubscribe()
	for {
		status = <-sub.Chan()

		// wait until finalisation
		if status.IsFinalized {
			fmt.Printf("BLOCK %v is finalized \n", status.AsFinalized.Hex())
			break
		}

		fmt.Printf("waiting for the extrinsic to be included/finalized \n")
	}

	fmt.Printf("block hash is %v \n", status.AsFinalized.Hex())

	clients, err := sp.RPCClient.RPC.IBC.QueryNewlyCreatedClients(status.AsFinalized)
	if err != nil {
		return nil, false, err
	}

	rlyRes := &provider.RelayerTxResponse{
		TxHash: status.AsUsurped.Hex(),
		Events: []provider.RelayerEvent{
			{
				EventType: clienttypes.EventTypeCreateClient,
				AttributeKey: clienttypes.AttributeKeyClientID,
				AttributeValue: clients[0].ClientId,
			},
		},
	}

	return rlyRes, true, nil
}

func (sp *SubstrateProvider) GetLightSignedHeaderAtHeight(ctx context.Context, h int64) (ibcexported.ClientMessage, error) {
	return sp.GetIBCUpdateHeader(ctx, h, nil, "not_used")
}

// GetIBCUpdateHeader updates the off chain beefy light client and
// returns an IBC Update Header which can be used to update an on chain
// light client on the destination chain. The source is used to construct
// the header data.
func (sp *SubstrateProvider) GetIBCUpdateHeader(ctx context.Context, srch int64, dst provider.ChainProvider, dstClientId string) (ibcexported.ClientMessage, error) {
	// Construct header data from light client representing source.
	h, err := sp.QueryHeaderAtHeight(ctx, srch)
	if err != nil {
		return nil, err
	}

	//// Inject trusted fields based on previous header data from source
	//// TODO: implement InjectTrustedFields, make findings on getting validator set from beefy header
	//// return sp.InjectTrustedFields(h, dst, dstClientId)
	//panic("implement me -> GetIBCUpdateHeader -> https://github.com/ComposableFi/relayer/issues/5")
	return h, nil
}

func (sp *SubstrateProvider) ChainId() string {
	return sp.Config.ChainID
}

func (sp *SubstrateProvider) Type() string {
	return "substrate"
}

func (sp *SubstrateProvider) ProviderConfig() provider.ProviderConfig {
	return sp.Config
}

func (sp *SubstrateProvider) Key() string {
	return sp.Config.Key
}

func (sp *SubstrateProvider) Address() (string, error) {
	info, err := sp.Keybase.Key(sp.Config.Key)
	if err != nil {
		return "", err
	}

	return info.GetAddress(), nil
}

func (sp *SubstrateProvider) Timeout() string {
	return sp.Config.Timeout
}

// TODO: define a more accurate trusting period
func (sp *SubstrateProvider) TrustingPeriod(ctx context.Context) (time.Duration, error) {
	return maxDuration(), nil
}

func (sp *SubstrateProvider) WaitForNBlocks(ctx context.Context, n int64) error {
	return nil
}
