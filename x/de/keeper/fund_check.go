package keeper

import (
	"context"
	"errors"
	"slices"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/de/types"
)

// ResolveDebitAccount implements [FUND-CHECK-1]; call it once per message, as it
// debits the sponsorship's remaining_spend when the sponsor pays.
func (k Keeper) ResolveDebitAccount(ctx context.Context, corporationID uint64, msgType types.MsgType, totals sdk.Coins) (sdk.AccAddress, error) {
	co, err := k.corporationKeeper().ResolveCorporationByID(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	sp, s, active, err := k.activeSponsor(ctx, co)
	if err != nil {
		return nil, err
	}
	if active {
		spAddr, err := sdk.AccAddressFromBech32(sp.PolicyAddress)
		if err != nil {
			return nil, err
		}
		if sponsorCovers(&s, msgType, totals) && k.bankKeeper.SpendableCoins(ctx, spAddr).IsAllGTE(totals) {
			if len(s.SpendLimit) > 0 {
				s.RemainingSpend = s.RemainingSpend.Sub(totals...)
				if err := k.Sponsorships.Set(ctx, collections.Join(sp.Id, co.Id), s); err != nil {
					return nil, err
				}
			}
			return spAddr, nil
		}
		events.New(types.ModuleName, types.EventTypeSponsorCouldNotPay).
			Uint(types.AttributeKeySponsorCorporationID, sp.Id).
			Uint(types.AttributeKeySponsoredCorporationID, co.Id).
			Enum(types.AttributeKeyMsgType, msgType).
			Coins(types.AttributeKeyTotals, totals).
			Emit(sdk.UnwrapSDKContext(ctx))
	}
	coAddr, err := sdk.AccAddressFromBech32(co.PolicyAddress)
	if err != nil {
		return nil, err
	}
	if !k.bankKeeper.SpendableCoins(ctx, coAddr).IsAllGTE(totals) {
		return nil, errorsmod.Wrapf(types.ErrNoFundingAccount, "corporation %d must cover %s", co.Id, totals)
	}
	return coAddr, nil
}

// ResolveCreditAccount implements [FUND-CHECK-2]; escrow refunds bypass it.
func (k Keeper) ResolveCreditAccount(ctx context.Context, corporationID uint64) (sdk.AccAddress, error) {
	co, err := k.corporationKeeper().ResolveCorporationByID(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	sp, _, active, err := k.activeSponsor(ctx, co)
	if err != nil {
		return nil, err
	}
	if active {
		return sdk.AccAddressFromBech32(sp.PolicyAddress)
	}
	return sdk.AccAddressFromBech32(co.PolicyAddress)
}

// FeeSponsor returns the fee-paying active sponsor of the granter, or nil.
func (k Keeper) FeeSponsor(ctx context.Context, granter sdk.AccAddress) (sdk.AccAddress, error) {
	co, found, err := k.corporationKeeper().GetCorporationByPolicyAddress(ctx, granter.String())
	if err != nil || !found {
		return nil, err
	}
	sp, s, active, err := k.activeSponsor(ctx, co)
	if err != nil || !active || !s.CoversFees {
		return nil, err
	}
	return sdk.AccAddressFromBech32(sp.PolicyAddress)
}

// activeSponsor resolves co's active sponsor [FUND-CHECK], resetting a due cycle.
func (k Keeper) activeSponsor(ctx context.Context, co types.CorporationView) (types.CorporationView, types.Sponsorship, bool, error) {
	if co.SponsorCorporationId == 0 {
		return types.CorporationView{}, types.Sponsorship{}, false, nil
	}
	key := collections.Join(co.SponsorCorporationId, co.Id)
	s, err := k.Sponsorships.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return types.CorporationView{}, types.Sponsorship{}, false, nil
	}
	if err != nil {
		return types.CorporationView{}, types.Sponsorship{}, false, err
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if !sponsorshipLive(&s, now) {
		return types.CorporationView{}, types.Sponsorship{}, false, nil
	}
	if s.Period != nil && !s.Expiration.After(now) {
		if len(s.SpendLimit) > 0 {
			s.RemainingSpend = s.SpendLimit
		}
		next := now.Add(*s.Period)
		s.Expiration = &next
		if err := k.Sponsorships.Set(ctx, key, s); err != nil {
			return types.CorporationView{}, types.Sponsorship{}, false, err
		}
	}
	sp, err := k.corporationKeeper().ResolveCorporationByID(ctx, co.SponsorCorporationId)
	if err != nil {
		return types.CorporationView{}, types.Sponsorship{}, false, err
	}
	return sp, s, true, nil
}

// sponsorshipLive is the expiry part of the active-sponsor test.
func sponsorshipLive(s *types.Sponsorship, now time.Time) bool {
	return s.Period != nil || s.Expiration == nil || s.Expiration.After(now)
}

// sponsorCovers is FUND-CHECK-1 step 1 without the balance check.
func sponsorCovers(s *types.Sponsorship, msgType types.MsgType, totals sdk.Coins) bool {
	if len(s.MsgTypes) > 0 && !slices.Contains(s.MsgTypes, msgType) {
		return false
	}
	return len(s.SpendLimit) == 0 || s.RemainingSpend.IsAllGTE(totals)
}
