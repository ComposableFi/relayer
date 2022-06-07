package provider

import (
	ibcexported "github.com/cosmos/ibc-go/v3/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
	beefyclient "github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
)

// ClientsMatch checks for the type of the clients that is to be compared, and runs the respective matcher function for each
// client type. Each provider that is added to the relayer would have its matcher function implemented here and registered
// in the ClientsMatch function.
func ClientsMatch(
	provider ChainProvider,
	existingClient, newClient ibcexported.ClientState,
	) (string, bool) {
	if existingClient.ClientType() != newClient.ClientType() {
		return "", false
	}

	switch c := existingClient.(type) {
	case *tmclient.ClientState:
		n := newClient.(*tmclient.ClientState)
		return TendermintMatcher(provider, c, n)
	case *beefyclient.ClientState:
		n := newClient.(*beefyclient.ClientState)
		return BeefyMatcher(provider, c, n)
	}
	return "", false
}

// TendermintMatcher accepts a tendermint chain provider and two light clients. It checks if they match, returns
// true if they do and false if they don't
func TendermintMatcher(
	provider ChainProvider,
	existingClient,
	newClient *tmclient.ClientState,
	) (string, bool) {
	// TODO: add code that checks that two tendermint clients match
	return "", false
}

// BeefyMatcher accepts a beefy chain provider and two light clients. It checks if they match, returns
// true if they do and false if they don't
func BeefyMatcher(
	provider ChainProvider,
	existingClient,
	newClient *beefyclient.ClientState,
	) (string, bool) {
	// TODO: add code that checks that two beefy clients match
	return "", false
}
