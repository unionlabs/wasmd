package cometbls

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// CheckForMisbehaviour detects duplicate height misbehaviour and BFT time violation misbehaviour
// in a submitted Header message and verifies the correctness of a submitted Misbehaviour ClientMessage
func (ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, msg exported.ClientMessage) bool {
	switch msg := msg.(type) {
	case *Header:
		tmHeader := msg
		consState := tmHeader.ConsensusState()

		// Check if the Client store already has a consensus state for the header's height
		// If the consensus state exists, and it matches the header then we return early
		// since header has already been submitted in a previous UpdateClient.
		if existingConsState, found := GetConsensusState(clientStore, cdc, tmHeader.GetHeight()); found {
			// This header has already been submitted and the necessary state is already stored
			// in client store, thus we can return early without further validation.
			if reflect.DeepEqual(existingConsState, tmHeader.ConsensusState()) { //nolint:gosimple
				return false
			}

			// A consensus state already exists for this height, but it does not match the provided header.
			// The assumption is that Header has already been validated. Thus we can return true as misbehaviour is present
			return true
		}

		// Check that consensus state timestamps are monotonic
		prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, tmHeader.GetHeight())
		nextCons, nextOk := GetNextConsensusState(clientStore, cdc, tmHeader.GetHeight())
		// if previous consensus state exists, check consensus state time is greater than previous consensus state time
		// if previous consensus state is not before current consensus state return true
		if prevOk && prevCons.Timestamp >= consState.Timestamp {
			return true
		}
		// if next consensus state exists, check consensus state time is less than next consensus state time
		// if next consensus state is not after current consensus state return true
		if nextOk && nextCons.Timestamp <= consState.Timestamp {
			return true
		}
	case *Misbehaviour:
		// if heights are equal check that this is valid misbehaviour of a fork
		// otherwise if heights are unequal check that this is valid misbehavior of BFT time violation
		if msg.Header_1.GetHeight() == msg.Header_2.GetHeight() {
			// TODO(aeryz): we need blockid to be in header

			// blockID1, err := tmtypes.BlockIDFromProto(&msg.Header_1.SignedHeader.Commit.BlockID)
			// if err != nil {
			// 	return false
			// }

			// blockID2, err := tmtypes.BlockIDFromProto(&msg.Header_2.SignedHeader.Commit.BlockID)
			// if err != nil {
			// 	return false
			// }

			// // Ensure that Commit Hashes are different
			// if !bytes.Equal(blockID1.Hash, blockID2.Hash) {
			// 	return true
			// }
			return false

		} else if !msg.Header_1.Time.After(msg.Header_2.Time) {
			// Header_1 is at greater height than Header_2, therefore Header_1 time must be less than or equal to
			// Header_2 time in order to be valid misbehaviour (violation of monotonic time).
			return true
		}
	}

	return false
}

// verifyMisbehaviour determines whether or not two conflicting
// headers at the same height would have convinced the light client.
//
// NOTE: consensusState1 is the trusted consensus state that corresponds to the TrustedHeight
// of misbehaviour.Header_1
// Similarly, consensusState2 is the trusted consensus state that corresponds
// to misbehaviour.Header_2
// Misbehaviour sets frozen height to {0, 1} since it is only used as a boolean value (zero or non-zero).
func (cs *ClientState) verifyMisbehaviour(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, misbehaviour *Misbehaviour) error {
	// Regardless of the type of misbehaviour, ensure that both headers are valid and would have been accepted by light-client

	// TODO(aeryz): fix this

	// // Retrieve trusted consensus states for each Header in misbehaviour
	// tmConsensusState1, found := GetConsensusState(clientStore, cdc, misbehaviour.Header_1.TrustedHeight)
	// if !found {
	// 	return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header_1 at TrustedHeight: %s", misbehaviour.Header_1.TrustedHeight)
	// }

	// tmConsensusState2, found := GetConsensusState(clientStore, cdc, misbehaviour.Header_2.TrustedHeight)
	// if !found {
	// 	return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header_2 at TrustedHeight: %s", misbehaviour.Header_2.TrustedHeight)
	// }

	// // Check the validity of the two conflicting headers against their respective
	// // trusted consensus states
	// // NOTE: header height and commitment root assertions are checked in
	// // misbehaviour.ValidateBasic by the client keeper and msg.ValidateBasic
	// // by the base application.
	// if err := checkMisbehaviourHeader(
	// 	cs, tmConsensusState1, misbehaviour.Header_1, uint64(ctx.BlockTime().UnixNano()),
	// ); err != nil {
	// 	return errorsmod.Wrap(err, "verifying Header_1 in Misbehaviour failed")
	// }
	// if err := checkMisbehaviourHeader(
	// 	cs, tmConsensusState2, misbehaviour.Header_2, uint64(ctx.BlockTime().UnixNano()),
	// ); err != nil {
	// 	return errorsmod.Wrap(err, "verifying Header_2 in Misbehaviour failed")
	// }

	return nil
}

// checkMisbehaviourHeader checks that a Header in Misbehaviour is valid misbehaviour given
// a trusted ConsensusState
func checkMisbehaviourHeader(
	clientState *ClientState, consState *ConsensusState, header *Header, currentTimestamp uint64,
) error {
	// check the trusted fields for the header against ConsensusState
	// if err := checkTrustedHeader(header, consState); err != nil {
	// 	return err
	// }

	// assert that the age of the trusted consensus state is not older than the trusting period
	if currentTimestamp-consState.Timestamp >= clientState.TrustingPeriod {
		return errorsmod.Wrapf(
			ErrTrustingPeriodExpired,
			"current timestamp minus the latest consensus state timestamp is greater than or equal to the trusting period (%d >= %d)",
			currentTimestamp-consState.Timestamp, clientState.TrustingPeriod,
		)
	}

	chainID := clientState.GetChainID()
	// If chainID is in revision format, then set revision number of chainID with the revision number
	// of the misbehaviour header
	// NOTE: misbehaviour verification is not supported for chains which upgrade to a new chainID without
	// strictly following the chainID revision format
	if clienttypes.IsRevisionFormat(chainID) {
		chainID, _ = clienttypes.SetRevisionNumber(chainID, header.GetHeight().GetRevisionNumber())
	}

	// // - ValidatorSet must have TrustLevel similarity with trusted FromValidatorSet
	// // - ValidatorSets on both headers are valid given the last trusted ValidatorSet
	// if err := tmTrustedValset.VerifyCommitLightTrusting(
	// 	chainID, tmCommit, clientState.TrustLevel.ToTendermint(),
	// ); err != nil {
	// 	return errorsmod.Wrapf(clienttypes.ErrInvalidMisbehaviour, "validator set in header has too much change from trusted validator set: %v", err)
	// }
	// TODO(aeryz): currently unimplemented
	return nil
}
