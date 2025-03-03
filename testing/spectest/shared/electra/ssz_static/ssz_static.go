package ssz_static

import (
	"context"
	"errors"
	"testing"

	fssz "github.com/prysmaticlabs/fastssz"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	common "github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/common/ssz_static"
)

// RunSSZStaticTests executes "ssz_static" tests.
func RunSSZStaticTests(t *testing.T, config string) {
	common.RunSSZStaticTests(t, config, "electra", UnmarshalledSSZ, customHtr)
}

func customHtr(t *testing.T, htrs []common.HTR, object interface{}) []common.HTR {
	_, ok := object.(*ethpb.BeaconStateElectra)
	if !ok {
		return htrs
	}

	htrs = append(htrs, func(s interface{}) ([32]byte, error) {
		beaconState, err := state_native.InitializeFromProtoElectra(s.(*ethpb.BeaconStateElectra))
		require.NoError(t, err)
		return beaconState.HashTreeRoot(context.Background())
	})
	return htrs
}

// UnmarshalledSSZ unmarshalls serialized input.
func UnmarshalledSSZ(t *testing.T, serializedBytes []byte, folderName string) (interface{}, error) {
	var obj interface{}
	switch folderName {
	case "ExecutionPayload":
		obj = &enginev1.ExecutionPayloadElectra{}
	case "ExecutionPayloadHeader":
		obj = &enginev1.ExecutionPayloadHeaderElectra{}
	case "Attestation":
		obj = &ethpb.AttestationElectra{}
	case "AttestationData":
		obj = &ethpb.AttestationData{}
	case "AttesterSlashing":
		obj = &ethpb.AttesterSlashingElectra{}
	case "AggregateAndProof":
		obj = &ethpb.AggregateAttestationAndProofElectra{}
	case "BeaconBlock":
		obj = &ethpb.BeaconBlockElectra{}
	case "BeaconBlockBody":
		obj = &ethpb.BeaconBlockBodyElectra{}
	case "BeaconBlockHeader":
		obj = &ethpb.BeaconBlockHeader{}
	case "BeaconState":
		obj = &ethpb.BeaconStateElectra{}
	case "Checkpoint":
		obj = &ethpb.Checkpoint{}
	case "Deposit":
		obj = &ethpb.Deposit{}
	case "DepositMessage":
		obj = &ethpb.DepositMessage{}
	case "DepositData":
		obj = &ethpb.Deposit_Data{}
	case "Eth1Data":
		obj = &ethpb.Eth1Data{}
	case "Eth1Block":
		t.Skip("Unused type")
		return nil, nil
	case "Fork":
		obj = &ethpb.Fork{}
	case "ForkData":
		obj = &ethpb.ForkData{}
	case "HistoricalBatch":
		obj = &ethpb.HistoricalBatch{}
	case "IndexedAttestation":
		obj = &ethpb.IndexedAttestationElectra{}
	case "PendingAttestation":
		obj = &ethpb.PendingAttestation{}
	case "ProposerSlashing":
		obj = &ethpb.ProposerSlashing{}
	case "SignedAggregateAndProof":
		obj = &ethpb.SignedAggregateAttestationAndProofElectra{}
	case "SignedBeaconBlock":
		obj = &ethpb.SignedBeaconBlockElectra{}
	case "SignedBeaconBlockHeader":
		obj = &ethpb.SignedBeaconBlockHeader{}
	case "SignedVoluntaryExit":
		obj = &ethpb.SignedVoluntaryExit{}
	case "SigningData":
		obj = &ethpb.SigningData{}
	case "Validator":
		obj = &ethpb.Validator{}
	case "VoluntaryExit":
		obj = &ethpb.VoluntaryExit{}
	case "SyncCommitteeMessage":
		obj = &ethpb.SyncCommitteeMessage{}
	case "SyncCommitteeContribution":
		obj = &ethpb.SyncCommitteeContribution{}
	case "ContributionAndProof":
		obj = &ethpb.ContributionAndProof{}
	case "SignedContributionAndProof":
		obj = &ethpb.SignedContributionAndProof{}
	case "SyncAggregate":
		obj = &ethpb.SyncAggregate{}
	case "SyncAggregatorSelectionData":
		obj = &ethpb.SyncAggregatorSelectionData{}
	case "SyncCommittee":
		obj = &ethpb.SyncCommittee{}
	case "LightClientOptimisticUpdate":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientFinalityUpdate":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientBootstrap":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientSnapshot":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientUpdate":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientHeader":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "BlobIdentifier":
		obj = &ethpb.BlobIdentifier{}
	case "BlobSidecar":
		obj = &ethpb.BlobSidecar{}
	case "PowBlock":
		obj = &ethpb.PowBlock{}
	case "Withdrawal":
		obj = &enginev1.Withdrawal{}
	case "HistoricalSummary":
		obj = &ethpb.HistoricalSummary{}
	case "BLSToExecutionChange":
		obj = &ethpb.BLSToExecutionChange{}
	case "SignedBLSToExecutionChange":
		obj = &ethpb.SignedBLSToExecutionChange{}
	case "PendingBalanceDeposit":
		obj = &ethpb.PendingBalanceDeposit{}
	case "PendingPartialWithdrawal":
		obj = &ethpb.PendingPartialWithdrawal{}
	case "PendingConsolidation":
		obj = &ethpb.PendingConsolidation{}
	case "WithdrawalRequest":
		obj = &enginev1.WithdrawalRequest{}
	case "DepositRequest":
		obj = &enginev1.DepositRequest{}
	case "ConsolidationRequest":
		obj = &enginev1.ConsolidationRequest{}
	case "ExecutionRequests":
		obj = &enginev1.ExecutionRequests{}
	default:
		return nil, errors.New("type not found")
	}
	var err error
	if o, ok := obj.(fssz.Unmarshaler); ok {
		err = o.UnmarshalSSZ(serializedBytes)
	} else {
		err = errors.New("could not unmarshal object, not a fastssz compatible object")
	}
	return obj, err
}
