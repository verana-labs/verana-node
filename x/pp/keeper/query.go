package keeper

import (
	"context"
	errors2 "errors"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-node/x/pp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListParticipants(goCtx context.Context, req *types.QueryListParticipantsRequest) (*types.QueryListParticipantsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-1-2] response_max_size: default 64, range 1..1024.
	maxSize := req.ResponseMaxSize
	if maxSize == 0 {
		maxSize = 64
	}
	if maxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1024")
	}

	// "when" is the reference time for active evaluation; defaults to now.
	when := ctx.BlockTime()
	if req.When != nil {
		when = *req.When
	}

	var participants []types.Participant
	err := k.Participant.Walk(ctx, nil, func(_ uint64, p types.Participant) (bool, error) {
		if req.SchemaId != 0 && p.SchemaId != req.SchemaId {
			return false, nil
		}
		if req.Grantee != "" && p.VsOperator != req.Grantee {
			return false, nil
		}
		if req.Did != "" && p.Did != req.Did {
			return false, nil
		}
		if req.ParticipantId != 0 && p.ValidatorParticipantId != req.ParticipantId {
			return false, nil
		}
		if req.Role != types.ParticipantRole_UNSPECIFIED && p.Role != req.Role {
			return false, nil
		}
		if req.OpState != types.OnboardingState_ONBOARDING_STATE_UNSPECIFIED && p.OpState != req.OpState {
			return false, nil
		}
		if req.ModifiedAfter != nil && p.Modified != nil && p.Modified.Before(*req.ModifiedAfter) {
			return false, nil
		}
		if req.OnlySlashed && p.Slashed == nil {
			return false, nil
		}
		if req.OnlyRepaid && p.Repaid == nil {
			return false, nil
		}
		if req.OnlyValid && !isActiveAt(p, when) {
			return false, nil
		}
		participants = append(participants, p)
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// [MOD-PP-QRY-1-3] ordered by modified asc (nil sorts first), then capped.
	sort.Slice(participants, func(i, j int) bool {
		return lessModified(participants[i].Modified, participants[j].Modified)
	})
	if len(participants) > int(maxSize) {
		participants = participants[:maxSize]
	}

	return &types.QueryListParticipantsResponse{
		Participants: participants,
	}, nil
}

// isActiveAt reports whether p is an active participant at time t: not revoked,
// slashed, or repaid, and t falls within [effective_from, effective_until).
func isActiveAt(p types.Participant, t time.Time) bool {
	if p.Revoked != nil || p.Slashed != nil || p.Repaid != nil {
		return false
	}
	if p.EffectiveFrom == nil || p.EffectiveFrom.After(t) {
		return false
	}
	if p.EffectiveUntil != nil && !t.Before(*p.EffectiveUntil) {
		return false
	}
	return true
}

func (k Keeper) GetParticipant(goCtx context.Context, req *types.QueryGetParticipantRequest) (*types.QueryGetParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-2-2] Checks
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "participant ID cannot be 0")
	}

	// [MOD-PP-QRY-2-3] Execution
	participant, err := k.Participant.Get(ctx, req.Id)
	if err != nil {
		if errors2.Is(collections.ErrNotFound, err) {
			return nil, status.Error(codes.NotFound, "participant not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
	}

	return &types.QueryGetParticipantResponse{
		Participant: participant,
	}, nil
}

func (k Keeper) GetParticipantSession(ctx context.Context, req *types.QueryGetParticipantSessionRequest) (*types.QueryGetParticipantSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	session, err := k.ParticipantSession.Get(sdkCtx, req.Id)
	if err != nil {
		if errors2.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Error(codes.Internal, "failed to get session")
	}

	return &types.QueryGetParticipantSessionResponse{
		Session: &session,
	}, nil
}

func (k Keeper) ListParticipantSessions(ctx context.Context, req *types.QueryListParticipantSessionsRequest) (*types.QueryListParticipantSessionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64 // Default value
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var sessions []types.ParticipantSession

	err := k.ParticipantSession.Walk(sdkCtx, nil, func(key string, session types.ParticipantSession) (bool, error) {
		// Apply modified_after filter if provided
		if req.ModifiedAfter != nil && !session.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		sessions = append(sessions, session)
		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sessions")
	}

	// Sort by modified time ascending (nil sorts first), then page. Truncating
	// inside the walk would page in store-key order, not by modified.
	sort.Slice(sessions, func(i, j int) bool {
		return lessModified(sessions[i].Modified, sessions[j].Modified)
	})
	if len(sessions) > int(req.ResponseMaxSize) {
		sessions = sessions[:req.ResponseMaxSize]
	}

	return &types.QueryListParticipantSessionsResponse{
		Sessions: sessions,
	}, nil
}

func (k Keeper) FindBeneficiaries(goCtx context.Context, req *types.QueryFindBeneficiariesRequest) (*types.QueryFindBeneficiariesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-4-2] Find Beneficiaries checks
	// if issuer_participant_id and verifier_participant_id are unset then MUST abort
	if req.IssuerParticipantId == 0 && req.VerifierParticipantId == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of issuer_participant_id or verifier_participant_id must be provided")
	}

	var issuerParticipant, verifierParticipant *types.Participant

	// if issuer_participant_id is specified, load issuer_participant from issuer_participant_id, Participant MUST exist and MUST be a valid participant
	if req.IssuerParticipantId != 0 {
		participant, err := k.Participant.Get(ctx, req.IssuerParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("issuer participant not found: %v", err))
		}

		// MUST be a valid participant
		if err := IsValidParticipant(participant, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("issuer participant is not valid: %v", err))
		}

		issuerParticipant = &participant
	}

	// if verifier_participant_id is specified, load verifier_participant from verifier_participant_id, Participant MUST exist and MUST be a valid participant
	if req.VerifierParticipantId != 0 {
		participant, err := k.Participant.Get(ctx, req.VerifierParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("verifier participant not found: %v", err))
		}

		// MUST be a valid participant
		if err := IsValidParticipant(participant, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("verifier participant is not valid: %v", err))
		}

		verifierParticipant = &participant
	}

	// [MOD-PP-QRY-4-3] Find Beneficiaries execution
	// create Set found_participant_set
	foundParticipantMap := make(map[uint64]types.Participant)

	// if issuer_participant is not null
	if issuerParticipant != nil {
		// set current_participant = issuer_participant
		currentParticipant := issuerParticipant

		// while current_participant.validator_participant_id is not null
		for currentParticipant.ValidatorParticipantId != 0 {
			// set current_participant to loaded participant from current_participant.validator_participant_id
			participant, err := k.Participant.Get(ctx, currentParticipant.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
			}
			currentParticipant = &participant

			// if current_participant.revoked IS NULL AND current_participant.slashed IS NULL, Add current_participant to found_participant_set
			// Note: SlashedDeposit > 0 indicates the participant has been slashed
			if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil {
				foundParticipantMap[currentParticipant.Id] = *currentParticipant
			}
		}
	}

	// Additionally, if verifier_participant is not null
	if verifierParticipant != nil {
		// if issuer_participant is not null, add issuer_participant to found_participant_set
		if issuerParticipant != nil {
			if issuerParticipant.Revoked == nil && issuerParticipant.Slashed == nil {
				foundParticipantMap[issuerParticipant.Id] = *issuerParticipant
			}
		}

		// set current_participant = verifier_participant
		currentParticipant := verifierParticipant

		// while verifier_participant.validator_participant_id is not null
		for currentParticipant.ValidatorParticipantId != 0 {
			// set current_participant to loaded participant from current_participant.validator_participant_id
			participant, err := k.Participant.Get(ctx, currentParticipant.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
			}
			currentParticipant = &participant

			// if current_participant.revoked IS NULL AND current_participant.slashed IS NULL, Add current_participant to found_participant_set
			if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil {
				foundParticipantMap[currentParticipant.Id] = *currentParticipant
			}
		}
	}

	// Convert map to array, sorted by participant id for deterministic output.
	participants := make([]types.Participant, 0, len(foundParticipantMap))
	for _, participant := range foundParticipantMap {
		participants = append(participants, participant)
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Id < participants[j].Id
	})

	return &types.QueryFindBeneficiariesResponse{
		Participants: participants,
	}, nil
}

// lessModified orders by Modified ascending with nil sorting first.
func lessModified(a, b *time.Time) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return a.Before(*b)
}
