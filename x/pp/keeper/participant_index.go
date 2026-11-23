package keeper

import (
	"context"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana-node/x/pp/types"
)

// IndexParticipant writes the create-time rows of every secondary index.
func (k Keeper) IndexParticipant(ctx context.Context, p types.Participant) error {
	if err := k.IndexParticipantDID(ctx, p); err != nil {
		return err
	}
	return k.indexParticipantLookups(ctx, p)
}

// indexParticipantLookups writes the root and unrepaid-slash rows.
func (k Keeper) indexParticipantLookups(ctx context.Context, p types.Participant) error {
	if p.Role == types.ParticipantRole_ECOSYSTEM {
		if err := k.RootParticipantBySchemaCorp.Set(ctx, collections.Join3(p.SchemaId, p.CorporationId, p.Id)); err != nil {
			return err
		}
	}
	return k.indexUnrepaidSlash(ctx, p)
}

// indexUnrepaidSlash keeps a row while the entry has an unrepaid slash.
// slashed is never cleared, so an unslashed entry never has a row to remove.
func (k Keeper) indexUnrepaidSlash(ctx context.Context, p types.Participant) error {
	if p.Slashed == nil {
		return nil
	}
	key := collections.Join3(p.CorporationId, p.SchemaId, p.Id)
	if p.RepaidDeposit < p.SlashedDeposit {
		return k.UnrepaidSlashByCorp.Set(ctx, key)
	}
	return k.UnrepaidSlashByCorp.Remove(ctx, key)
}

// walkParticipantsByDID calls fn for every Participant with this did, in id order.
func (k Keeper) walkParticipantsByDID(ctx context.Context, did string, fn func(p types.Participant) (bool, error)) error {
	rng := collections.NewPrefixedPairRange[string, uint64](did)
	return k.ParticipantByDIDCorp.Walk(ctx, rng, func(key collections.Pair[string, uint64], _ uint64) (bool, error) {
		p, err := k.Participant.Get(ctx, key.K2())
		if err != nil {
			return true, err
		}
		return fn(p)
	})
}
