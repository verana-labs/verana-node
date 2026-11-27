package keeper

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantSponsorship implements [MOD-DE-MSG-10].
func (ms msgServer) GrantSponsorship(goCtx context.Context, msg *types.MsgGrantSponsorship) (*types.MsgGrantSponsorshipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-10-2] Basic checks.
	co, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Signer)
	if err != nil {
		return nil, err
	}
	if msg.SponsoredCorporationId == co.Id {
		return nil, errorsmod.Wrap(types.ErrInvalidSponsorship, "a corporation cannot sponsor itself")
	}
	sponsored, err := ms.corporationKeeper().ResolveCorporationByID(ctx, msg.SponsoredCorporationId)
	if errors.Is(err, collections.ErrNotFound) {
		return nil, errorsmod.Wrapf(types.ErrInvalidSponsorship, "sponsored corporation %d not found", msg.SponsoredCorporationId)
	}
	if err != nil {
		return nil, err
	}
	if err := validateSponsorshipTerms(msg, now); err != nil {
		return nil, err
	}

	key := collections.Join(co.Id, sponsored.Id)
	current, err := ms.Sponsorships.Get(ctx, key)
	found := err == nil
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	if err := ms.checkSponsorshipSlot(ctx, co, sponsored, found && current.CoversFees); err != nil {
		return nil, err
	}
	if found {
		if err := checkRaiseOnly(&current, msg); err != nil {
			return nil, err
		}
	}

	// [MOD-DE-MSG-10-4] Execution.
	s := types.Sponsorship{
		SponsorCorporationId:   co.Id,
		SponsoredCorporationId: sponsored.Id,
		MsgTypes:               msg.MsgTypes,
		SpendLimit:             msg.SpendLimit,
		RemainingSpend:         msg.SpendLimit,
		Expiration:             msg.Expiration,
		Period:                 msg.Period,
		CoversFees:             msg.CoversFees,
		Created:                now,
	}
	if found {
		s.Created = current.Created
		s.RemainingSpend = raisedRemaining(&current, msg.SpendLimit)
	}
	if msg.CoversFees {
		s.FeeSpendLimit = msg.FeeSpendLimit
	}
	if err := ms.Sponsorships.Set(ctx, key, s); err != nil {
		return nil, err
	}
	// covers_fees false: the slot check leaves no allowance to revoke.
	if msg.CoversFees {
		if err := ms.grantSponsorshipAllowance(ctx, co, sponsored, &s); err != nil {
			return nil, err
		}
	}

	ev := events.New(types.ModuleName, types.EventTypeGrantSponsorship).
		Uint(types.AttributeKeySponsorCorporationID, co.Id).
		Uint(types.AttributeKeySponsoredCorporationID, sponsored.Id)
	if !found {
		ev.Time(events.AttrNow, now)
	}
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(ctx)
	return &types.MsgGrantSponsorshipResponse{}, nil
}

// RevokeSponsorship implements [MOD-DE-MSG-11].
func (ms msgServer) RevokeSponsorship(goCtx context.Context, msg *types.MsgRevokeSponsorship) (*types.MsgRevokeSponsorshipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-DE-MSG-11-2] Basic checks.
	co, group, err := ms.corporationKeeper().ResolveSigner(ctx, msg.Corporation, msg.Signer)
	if err != nil {
		return nil, err
	}
	if !group {
		if err := ms.CheckDelegateAuthorization(ctx, co.Id, msg.Signer, types.MsgType_MSG_TYPE_REVOKE_SPONSORSHIP, ctx.BlockTime()); err != nil {
			return nil, err
		}
	}
	key := collections.Join(co.Id, msg.SponsoredCorporationId)
	has, err := ms.Sponsorships.Has(ctx, key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errorsmod.Wrapf(types.ErrSponsorshipNotFound, "sponsor %d, sponsored %d", co.Id, msg.SponsoredCorporationId)
	}

	// [MOD-DE-MSG-11-4] Execution.
	if err := ms.Sponsorships.Remove(ctx, key); err != nil {
		return nil, err
	}
	sponsored, err := ms.corporationKeeper().ResolveCorporationByID(ctx, msg.SponsoredCorporationId)
	if err != nil {
		return nil, err
	}
	granter, grantee, err := policyAddrs(co, sponsored)
	if err != nil {
		return nil, err
	}
	fk := ms.feegrantKeeper()
	if _, err := fk.GetAllowance(ctx, granter, grantee); err == nil {
		if err := fk.RevokeAllowance(ctx, granter, grantee); err != nil {
			return nil, err
		}
	}
	cleared, err := ms.corporationKeeper().ClearSponsor(ctx, sponsored.Id, co.Id)
	if err != nil {
		return nil, err
	}

	ev := events.New(types.ModuleName, types.EventTypeRevokeSponsorship).
		Uint(types.AttributeKeySponsorCorporationID, co.Id).
		Uint(types.AttributeKeySponsoredCorporationID, sponsored.Id).
		Bool(types.AttributeKeySponsorCleared, cleared)
	if group {
		if err := ev.Msg(ms.cdc, msg); err != nil {
			return nil, err
		}
	}
	ev.Emit(ctx)
	return &types.MsgRevokeSponsorshipResponse{}, nil
}

// validateSponsorshipTerms holds the stateless checks of MOD-DE-MSG-10-2.
func validateSponsorshipTerms(msg *types.MsgGrantSponsorship, now time.Time) error {
	if len(msg.MsgTypes) > 0 {
		if err := types.ValidateMsgTypes(msg.MsgTypes, types.SponsorshipMsgTypes); err != nil {
			return err
		}
	}
	if len(msg.SpendLimit) > 0 && !msg.SpendLimit.IsValid() {
		return errorsmod.Wrap(types.ErrInvalidSpendLimit, "spend_limit")
	}
	if msg.Period != nil {
		if *msg.Period <= 0 {
			return errorsmod.Wrap(types.ErrInvalidSponsorship, "period must be a positive duration")
		}
		if len(msg.SpendLimit) == 0 || msg.Expiration == nil {
			return errorsmod.Wrap(types.ErrInvalidSponsorship, "period requires spend_limit and expiration")
		}
	}
	if msg.Expiration != nil && !msg.Expiration.After(now) {
		return types.ErrExpirationInPast
	}
	if len(msg.FeeSpendLimit) > 0 && !msg.FeeSpendLimit.IsValid() {
		return errorsmod.Wrap(types.ErrInvalidSpendLimit, "fee_spend_limit")
	}
	return nil
}

// checkSponsorshipSlot refuses a slot held by another allowance; ownsSlot means
// a fee-covering sponsorship already holds it.
func (k Keeper) checkSponsorshipSlot(ctx context.Context, co, sponsored types.CorporationView, ownsSlot bool) error {
	slot := collections.Join(co.Id, sponsored.PolicyAddress)
	if has, err := k.FeeGrants.Has(ctx, slot); err != nil || has {
		return slotErr(err, "is the grantee of a fee grant of the sponsor")
	}
	if has, err := k.VSOAByCorpOp.Has(ctx, slot); err != nil || has {
		return slotErr(err, "is a VS account of the sponsor")
	}
	if ownsSlot {
		return nil
	}
	granter, grantee, err := policyAddrs(co, sponsored)
	if err != nil {
		return err
	}
	if _, err := k.feegrantKeeper().GetAllowance(ctx, granter, grantee); err == nil {
		return slotErr(nil, "holds an allowance of the sponsor")
	}
	return nil
}

func slotErr(err error, reason string) error {
	if err != nil {
		return err
	}
	return errorsmod.Wrap(types.ErrSponsorshipSlotTaken, reason)
}

// checkRaiseOnly enforces that a re-grant is at least as generous as cur.
func checkRaiseOnly(cur *types.Sponsorship, msg *types.MsgGrantSponsorship) error {
	notRaised := func(reason string) error { return errorsmod.Wrap(types.ErrSponsorshipNotRaised, reason) }
	if !sameDuration(cur.Period, msg.Period) {
		return notRaised("period must be unchanged")
	}
	if len(msg.MsgTypes) > 0 && !coversMsgTypes(msg.MsgTypes, cur.MsgTypes) {
		return notRaised("msg_types must be absent or a superset")
	}
	if !raisedLimit(cur.SpendLimit, msg.SpendLimit) {
		return notRaised("spend_limit must be absent or at least the current one")
	}
	if msg.Expiration != nil && (cur.Expiration == nil || !msg.Expiration.After(*cur.Expiration)) {
		return notRaised("expiration must be absent or later than the current one")
	}
	if cur.CoversFees && !msg.CoversFees {
		return notRaised("covers_fees cannot be withdrawn")
	}
	if cur.CoversFees && !raisedLimit(cur.FeeSpendLimit, msg.FeeSpendLimit) {
		return notRaised("fee_spend_limit must be absent or at least the current one")
	}
	return nil
}

func sameDuration(a, b *time.Duration) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

// coversMsgTypes reports whether next lists every value of cur; empty is all.
func coversMsgTypes(next, cur []types.MsgType) bool {
	listed := make(map[types.MsgType]bool, len(next))
	for _, mt := range next {
		listed[mt] = true
	}
	if len(cur) == 0 {
		for mt := range types.SponsorshipMsgTypes {
			if !listed[mt] {
				return false
			}
		}
		return true
	}
	for _, mt := range cur {
		if !listed[mt] {
			return false
		}
	}
	return true
}

// raisedLimit reports whether next is at least cur; absent is unlimited.
func raisedLimit(cur, next sdk.Coins) bool {
	if len(next) == 0 {
		return true
	}
	return len(cur) > 0 && next.IsAllGTE(cur)
}

// raisedRemaining is the remaining_spend after a raise, per MOD-DE-MSG-10-4.
func raisedRemaining(cur *types.Sponsorship, limit sdk.Coins) sdk.Coins {
	if len(cur.SpendLimit) == 0 || len(limit) == 0 {
		return limit
	}
	remaining := make([]sdk.Coin, 0, len(limit))
	for _, c := range limit {
		amt := c.Amount
		if old := cur.SpendLimit.AmountOf(c.Denom); old.IsPositive() {
			amt = cur.RemainingSpend.AmountOf(c.Denom).Add(c.Amount).Sub(old)
		}
		remaining = append(remaining, sdk.NewCoin(c.Denom, amt))
	}
	return sdk.NewCoins(remaining...)
}

// grantSponsorshipAllowance creates or replaces the allowance of covers_fees.
func (k Keeper) grantSponsorshipAllowance(ctx context.Context, co, sponsored types.CorporationView, s *types.Sponsorship) error {
	urls, err := k.networkFeeRef.K.MaxGasMsgTypeURLs(ctx)
	if err != nil {
		return err
	}
	var inner feegrant.FeeAllowanceI
	if len(s.FeeSpendLimit) > 0 && s.Period != nil {
		inner = &feegrant.PeriodicAllowance{
			Period:           *s.Period,
			PeriodSpendLimit: s.FeeSpendLimit,
			PeriodCanSpend:   s.FeeSpendLimit,
			PeriodReset:      *s.Expiration,
		}
	} else {
		basic := &feegrant.BasicAllowance{SpendLimit: s.FeeSpendLimit}
		if s.Period == nil {
			basic.Expiration = s.Expiration
		}
		inner = basic
	}
	allowed, err := feegrant.NewAllowedMsgAllowance(inner, urls)
	if err != nil {
		return err
	}
	if err := allowed.ValidateBasic(); err != nil {
		return err
	}
	granter, grantee, err := policyAddrs(co, sponsored)
	if err != nil {
		return err
	}
	fk := k.feegrantKeeper()
	if _, err := fk.GetAllowance(ctx, granter, grantee); err == nil {
		if err := fk.RevokeAllowance(ctx, granter, grantee); err != nil {
			return err
		}
	}
	return fk.GrantAllowance(ctx, granter, grantee, allowed)
}

func policyAddrs(granter, grantee types.CorporationView) (sdk.AccAddress, sdk.AccAddress, error) {
	g, err := sdk.AccAddressFromBech32(granter.PolicyAddress)
	if err != nil {
		return nil, nil, err
	}
	e, err := sdk.AccAddressFromBech32(grantee.PolicyAddress)
	if err != nil {
		return nil, nil, err
	}
	return g, e, nil
}
