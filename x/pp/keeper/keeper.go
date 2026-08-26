package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/pp/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
		// state
		Participant          collections.Map[uint64, types.Participant]
		ParticipantByDIDCorp collections.Map[collections.Pair[string, uint64], uint64]
		ParticipantCounter   collections.Item[uint64]
		ParticipantSession   collections.Map[string, types.ParticipantSession]

		// external keeper
		credentialSchemaKeeper types.CredentialSchemaKeeper
		ecosystemKeeper        types.EcosystemKeeper
		exchangeRateKeeper     types.ExchangeRateKeeper
		coKeeper               types.CorporationKeeper
		trustDeposit           types.TrustDepositKeeper
		bankKeeper             types.BankKeeper
		delegationKeeper       types.DelegationKeeper
		digestKeeper           types.DigestKeeper

		// txDecoderRef holds the tx decoder; CSPS reads the tx fee for AUTHZ-CHECK-4.
		txDecoderRef *txDecoderHolder
	}

	txDecoderHolder struct{ decode sdk.TxDecoder }
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	credentialSchemaKeeper types.CredentialSchemaKeeper,
	ecosystemKeeper types.EcosystemKeeper,
	exchangeRateKeeper types.ExchangeRateKeeper,
	coKeeper types.CorporationKeeper,
	trustDeposit types.TrustDepositKeeper,
	bankKeeper types.BankKeeper,
	delegationKeeper types.DelegationKeeper,
	digestKeeper types.DigestKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:                    cdc,
		storeService:           storeService,
		authority:              authority,
		logger:                 logger,
		Participant:            collections.NewMap(sb, types.ParticipantKey, "participant", collections.Uint64Key, codec.CollValue[types.Participant](cdc)),
		ParticipantByDIDCorp:   collections.NewMap(sb, types.ParticipantByDIDCorpKey, "participant_by_did_corp", collections.PairKeyCodec(collections.StringKey, collections.Uint64Key), collections.Uint64Value),
		ParticipantCounter:     collections.NewItem(sb, types.ParticipantCounterKey, "participant_counter", collections.Uint64Value),
		ParticipantSession:     collections.NewMap(sb, types.ParticipantSessionKey, "participant_session", collections.StringKey, codec.CollValue[types.ParticipantSession](cdc)),
		credentialSchemaKeeper: credentialSchemaKeeper,
		ecosystemKeeper:        ecosystemKeeper,
		exchangeRateKeeper:     exchangeRateKeeper,
		coKeeper:               coKeeper,
		trustDeposit:           trustDeposit,
		bankKeeper:             bankKeeper,
		delegationKeeper:       delegationKeeper,
		digestKeeper:           digestKeeper,
		txDecoderRef:           &txDecoderHolder{},
	}
}

// SetTxDecoder wires the tx decoder after construction (used by CSPS AUTHZ-CHECK-4).
func (k Keeper) SetTxDecoder(d sdk.TxDecoder) {
	k.txDecoderRef.decode = d
}

func (k Keeper) txDecoder() sdk.TxDecoder {
	return k.txDecoderRef.decode
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetParticipantByID(ctx sdk.Context, id uint64) (types.Participant, error) {
	return k.Participant.Get(ctx, id)
}

// CreateParticipant creates a new participant and returns its ID
func (k Keeper) CreateParticipant(ctx sdk.Context, participant types.Participant) (uint64, error) {
	id, err := k.getNextParticipantID(ctx)
	if err != nil {
		return 0, err
	}
	participant.Id = id
	if err := k.Participant.Set(ctx, id, participant); err != nil {
		return 0, err
	}
	// Maintain the (did, id) index (single chokepoint for all create paths).
	if err := k.IndexParticipantDID(ctx, participant); err != nil {
		return 0, err
	}

	return id, nil
}

// getNextParticipantID returns the next id to allocate. The counter stores
// next-id (not last-id) so it matches the next_participant_id genesis field.
func (k Keeper) getNextParticipantID(ctx sdk.Context) (uint64, error) {
	id, err := k.ParticipantCounter.Get(ctx)
	if err != nil || id == 0 {
		id = 1
	}

	if err := k.ParticipantCounter.Set(ctx, id+1); err != nil {
		return 0, fmt.Errorf("failed to set participant counter: %w", err)
	}

	return id, nil
}

func (k Keeper) UpdateParticipant(ctx sdk.Context, participant types.Participant) error {
	return k.Participant.Set(ctx, participant.Id, participant)
}

// IsValidParticipant checks if a participant is valid (ACTIVE) at a given time:
// effective_from is set and <= checkTime, checkTime < effective_until when set,
// and revoked, slashed and repaid are all null.
func IsValidParticipant(participant types.Participant, checkTime time.Time) error {
	// Check if participant is repaid (REPAID state)
	if participant.Repaid != nil {
		return fmt.Errorf("participant is repaid since %v", participant.Repaid)
	}

	// Check if participant is slashed (SLASHED state)
	if participant.Slashed != nil {
		return fmt.Errorf("participant is slashed since %v", participant.Slashed)
	}

	// Check if participant is revoked (REVOKED state). Glossary: an active
	// participant has revoked == null, so a revocation set this block also aborts.
	if participant.Revoked != nil {
		return fmt.Errorf("participant is revoked since %v", participant.Revoked)
	}

	// Check if participant is expired (EXPIRED state)
	if participant.EffectiveUntil != nil && !checkTime.Before(*participant.EffectiveUntil) {
		return fmt.Errorf("participant expired: ended at %v", participant.EffectiveUntil)
	}

	// Check if participant is in FUTURE state (effective_from is after now)
	if participant.EffectiveFrom != nil && checkTime.Before(*participant.EffectiveFrom) {
		return fmt.Errorf("participant not yet effective: begins at %v", participant.EffectiveFrom)
	}

	// INACTIVE: effective_from is null (never validated)
	if participant.EffectiveFrom == nil {
		return fmt.Errorf("participant is INACTIVE: effective_from is null")
	}

	return nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.String()
}
