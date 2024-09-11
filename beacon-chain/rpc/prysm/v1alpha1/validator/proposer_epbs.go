package validator

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SubmitSignedExecutionPayloadEnvelope submits a signed execution payload envelope to the network.
func (vs *Server) SubmitSignedExecutionPayloadEnvelope(ctx context.Context, env *enginev1.SignedExecutionPayloadEnvelope) (*emptypb.Empty, error) {
	if err := vs.P2P.Broadcast(ctx, env); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to broadcast signed execution payload envelope: %v", err)
	}

	m, err := blocks.WrappedROExecutionPayloadEnvelope(env.Message)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to wrap execution payload envelope: %v", err)
	}

	if err := vs.ExecutionPayloadReceiver.ReceiveExecutionPayloadEnvelope(ctx, m, nil); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to receive execution payload envelope: %v", err)
	}

	return nil, nil
}

// SubmitSignedExecutionPayloadHeader submits a signed execution payload header to the beacon node.
func (vs *Server) SubmitSignedExecutionPayloadHeader(ctx context.Context, h *enginev1.SignedExecutionPayloadHeader) (*emptypb.Empty, error) {
	if vs.TimeFetcher.CurrentSlot() != h.Message.Slot {
		return nil, status.Errorf(codes.InvalidArgument, "current slot mismatch: expected %d, got %d", vs.TimeFetcher.CurrentSlot(), h.Message.Slot)
	}

	vs.signedExecutionPayloadHeader = h

	return nil, nil
}
