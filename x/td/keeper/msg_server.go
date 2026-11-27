package keeper

import (
	"context"
	"fmt"
	mathstd "math"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/td/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// authorize runs AUTHZ-CHECK-5 (with GF-CHECK-1), then AUTHZ-CHECK-1 for a
// delegate signer, and returns the resolved corporation id and whether the
// signer is on the group-proposal path.
func (ms msgServer) authorize(ctx sdk.Context, corporation, signer string, msgType detypes.MsgType) (uint64, bool, error) {
	if ms.Keeper.delegationKeeper == nil {
		return 0, false, fmt.Errorf("delegation keeper is required for delegate authorization")
	}
	co, group, err := ms.Keeper.coKeeper.ResolveSigner(ctx, corporation, signer)
	if err != nil || group {
		return co.Id, group, err
	}
	if err := ms.Keeper.delegationKeeper.CheckDelegateAuthorization(ctx, co.Id, signer, msgType, ctx.BlockTime()); err != nil {
		return 0, false, fmt.Errorf("authorization check failed: %w", err)
	}
	return co.Id, false, nil
}

// ReclaimTrustDepositYield is refused: [MOD-TD-MSG-2] is void in v5, so the
// message has no MsgType and no authorization covers it.
func (ms msgServer) ReclaimTrustDepositYield(_ context.Context, _ *types.MsgReclaimTrustDepositYield) (*types.MsgReclaimTrustDepositYieldResponse, error) {
	return nil, errorsmod.Wrap(types.ErrMsgVoid, "MsgReclaimTrustDepositYield (MOD-TD-MSG-2)")
}

// ShareToAmount converts share value to amount using decimal math
func (k Keeper) ShareToAmount(share math.LegacyDec, shareValue math.LegacyDec) uint64 {
	amountDec := share.Mul(shareValue)
	return amountDec.TruncateInt().Uint64()
}

// AmountToShare converts amount to share value using decimal math
func (k Keeper) AmountToShare(amount uint64, shareValue math.LegacyDec) math.LegacyDec {
	if shareValue.IsZero() {
		return math.LegacyZeroDec() // Prevent division by zero
	}
	return math.LegacyNewDecFromInt(math.NewIntFromUint64(amount)).Quo(shareValue)
}

// SlashTrustDeposit handles governance slashing of trust deposits
func (ms msgServer) SlashTrustDeposit(goCtx context.Context, msg *types.MsgSlashTrustDeposit) (*types.MsgSlashTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [CRITICAL] Authority check - only governance can call this
	if ms.Keeper.authority != msg.Signer {
		return nil, fmt.Errorf("invalid signer; expected %s, got %s", ms.Keeper.authority, msg.Signer)
	}

	// [MOD-TD-MSG-5-2-1] Basic checks
	if msg.Deposit.IsZero() || msg.Deposit.IsNegative() {
		return nil, fmt.Errorf("deposit must be greater than 0")
	}

	// [BUG-H2] Guard against uint64 overflow before calling Uint64()
	if !msg.Deposit.IsUint64() {
		return nil, fmt.Errorf("deposit amount exceeds uint64")
	}

	// Check if TrustDeposit entry exists for the corporation
	td, err := ms.Keeper.TrustDeposit.Get(ctx, msg.CorporationId)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for corporation_id: %d", msg.CorporationId)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Deposit).LT(msg.Deposit) {
		return nil, fmt.Errorf("insufficient trust deposit: deposit=%d, required=%s", td.Deposit, msg.Deposit.String())
	}

	// [MOD-TD-MSG-5-3] Execute the slash
	now := ctx.BlockTime()

	shareValue := ms.Keeper.GetTrustDepositShareValue(ctx)
	if !shareValue.IsPositive() {
		return nil, fmt.Errorf("trust_deposit_share_value must be positive")
	}

	// Calculate share reduction
	shareReduction := math.LegacyNewDecFromInt(msg.Deposit).Quo(shareValue)

	// [MOD-TD-MSG-5-3] Update TrustDeposit entry
	td.Deposit = td.Deposit - msg.Deposit.Uint64()
	td.Share = td.Share.Sub(shareReduction)
	if td.Share.IsNegative() {
		td.Share = math.LegacyZeroDec()
	}
	td.SlashedDeposit = td.SlashedDeposit + msg.Deposit.Uint64()
	td.LastSlashed = &now
	td.SlashCount++

	// [MOD-TD-MSG-5] Slash invariant: refunded MUST NOT exceed the post-slash
	// deposit. If a slash reduces deposit below the outstanding refunded amount,
	// clip refunded to the new deposit (the excess is forfeit).
	if td.Refunded > td.Deposit {
		td.Refunded = td.Deposit
	}

	// Save state before the bank op.
	if err := ms.Keeper.TrustDeposit.Set(ctx, msg.CorporationId, td); err != nil {
		return nil, fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// [MOD-TD-MSG-5-3] burn the slashed amount from the TrustDeposit account.
	if !msg.Deposit.IsInt64() {
		return nil, fmt.Errorf("slash amount exceeds int64")
	}
	if err := ms.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName,
		sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, msg.Deposit.Int64()))); err != nil {
		return nil, fmt.Errorf("failed to burn slashed coins: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlashTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(msg.CorporationId, 10)),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Deposit.String()),
			sdk.NewAttribute(types.AttributeKeySlashCount, strconv.FormatUint(td.SlashCount, 10)),
			sdk.NewAttribute("reason", msg.Reason),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)

	return &types.MsgSlashTrustDepositResponse{}, nil
}

func (ms msgServer) RepaySlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepaySlashedTrustDeposit) (*types.MsgRepaySlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account := msg.Corporation

	// [MOD-TD-MSG-6-2-1] [AUTHZ-CHECK]
	corpID, group, err := ms.authorize(ctx, msg.Corporation, msg.Signer, detypes.MsgType_MSG_TYPE_REPAY_SLASH)
	if err != nil {
		return nil, err
	}

	// [MOD-TD-MSG-6-2-1] Load TrustDeposit entry for corporation (must exist, keyed by corporation_id)
	td, err := ms.Keeper.TrustDeposit.Get(ctx, corpID)
	if err != nil {
		return nil, fmt.Errorf("trust deposit entry not found for corporation %s: %w", account, err)
	}

	// [MOD-TD-MSG-6-2-1] deposit MUST be re-checked > 0 at execution.
	if msg.Deposit == 0 {
		return nil, fmt.Errorf("deposit must be greater than zero")
	}
	// [MOD-TD-MSG-6-2-1] deposit MUST equal the outstanding slashed amount (slashed - repaid).
	outstanding := td.SlashedDeposit - td.RepaidDeposit
	if msg.Deposit != outstanding {
		return nil, fmt.Errorf("deposit must exactly equal outstanding slashed amount: expected %d, got %d", outstanding, msg.Deposit)
	}

	// Validate corporation address for bank transfer
	corporationAddr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-TD-MSG-6-3] Execution
	now := ctx.BlockTime()

	// Update trust deposit fields
	td.Deposit += msg.Deposit

	// td.share = td.share + deposit / GlobalVariables.trust_deposit_share_value
	shareIncrease := ms.Keeper.AmountToShare(msg.Deposit, ms.Keeper.GetTrustDepositShareValue(ctx))
	td.Share = td.Share.Add(shareIncrease)

	// [MOD-TD-MSG-6-3] slashed_deposit stays cumulative (only MOD-TD-MSG-5 increments it);
	// repaid_deposit accumulates. Outstanding = slashed - repaid.
	td.RepaidDeposit += msg.Deposit
	// td.last_repaid = now
	td.LastRepaid = &now

	// Save updated trust deposit BEFORE bank transfer to ensure atomicity
	if err := ms.Keeper.TrustDeposit.Set(ctx, corpID, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// [MOD-TD-MSG-6-2-2] / [MOD-TD-MSG-6-3] Transfer deposit from corporation to TrustDeposit account.
	// The corporation sends new coins to replenish the locked slashed amount.
	if msg.Deposit > uint64(mathstd.MaxInt64) {
		return nil, fmt.Errorf("repay amount exceeds maximum coin amount: %d", msg.Deposit)
	}
	// [MOD-TD-MSG-6-3] add amount to the TrustDeposit account (slashed coins were
	// already burned at slash time per MOD-TD-MSG-5-3).
	transferCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(msg.Deposit)))
	// AUTHZ-CHECK-1 step 3: the repayment is the consumed amount of a delegate.
	if !group {
		if err := ms.Keeper.delegationKeeper.ConsumeDelegateSpend(
			ctx, corpID, msg.Signer,
			detypes.MsgType_MSG_TYPE_REPAY_SLASH, now, transferCoins,
		); err != nil {
			return nil, fmt.Errorf("failed to consume delegate spend: %w", err)
		}
	}
	if err := ms.Keeper.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		corporationAddr,
		types.ModuleName,
		transferCoins,
	); err != nil {
		return nil, fmt.Errorf("failed to transfer tokens: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRepaySlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corpID, 10)),
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAmount, strconv.FormatUint(msg.Deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepaySlashedTrustDepositResponse{}, nil
}
