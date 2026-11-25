package ante

import (
	"context"
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
}

type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// FeeSponsorKeeper returns the sponsor paying the fees granted by granter, or nil.
type FeeSponsorKeeper interface {
	FeeSponsor(ctx context.Context, granter sdk.AccAddress) (sdk.AccAddress, error)
}

// DeductFeeDecorator is the SDK fee decorator plus the sponsored-fees rule.
type DeductFeeDecorator struct {
	ak       AccountKeeper
	bk       BankKeeper
	fk       ante.FeegrantKeeper
	sponsors FeeSponsorKeeper
	checkFee ante.TxFeeChecker
}

func NewDeductFeeDecorator(ak AccountKeeper, bk BankKeeper, fk ante.FeegrantKeeper, sponsors FeeSponsorKeeper, tfc ante.TxFeeChecker) DeductFeeDecorator {
	if tfc == nil {
		tfc = checkTxFeeWithValidatorMinGasPrices
	}
	return DeductFeeDecorator{ak: ak, bk: bk, fk: fk, sponsors: sponsors, checkFee: tfc}
}

func (d DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	fee, priority := feeTx.GetFee(), int64(0)
	if !simulate {
		var err error
		if fee, priority, err = d.checkFee(ctx, tx); err != nil {
			return ctx, err
		}
	}
	payer, err := d.deductFee(ctx, feeTx, fee)
	if err != nil {
		return ctx, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
		sdk.NewAttribute(sdk.AttributeKeyFeePayer, payer.String()),
	))
	return next(ctx.WithPriority(priority), tx, simulate)
}

// deductFee debits fee and returns the account that paid it.
func (d DeductFeeDecorator) deductFee(ctx sdk.Context, feeTx sdk.FeeTx, fee sdk.Coins) (sdk.AccAddress, error) {
	if d.ak.GetModuleAddress(authtypes.FeeCollectorName) == nil {
		return nil, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}
	payer := sdk.AccAddress(feeTx.FeePayer())
	granter := sdk.AccAddress(feeTx.FeeGranter())
	if granter.Empty() {
		return payer, d.debit(ctx, payer, fee)
	}
	if !granter.Equals(payer) {
		if err := d.fk.UseGrantedFees(ctx, granter, payer, fee, feeTx.GetMsgs()); err != nil {
			return nil, errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", granter, payer)
		}
	}
	sponsor, err := d.sponsors.FeeSponsor(ctx, granter)
	if err != nil {
		return nil, err
	}
	if sponsor != nil && d.sponsorPays(ctx, sponsor, granter, fee, feeTx.GetMsgs()) {
		return sponsor, nil
	}
	return granter, d.debit(ctx, granter, fee)
}

// sponsorPays checks the balance first; a failed step leaves no state change.
func (d DeductFeeDecorator) sponsorPays(ctx sdk.Context, sponsor, corporation sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) bool {
	if !d.bk.SpendableCoins(ctx, sponsor).IsAllGTE(fee) {
		return false
	}
	cacheCtx, write := ctx.CacheContext()
	if d.fk.UseGrantedFees(cacheCtx, sponsor, corporation, fee, msgs) != nil || d.debit(cacheCtx, sponsor, fee) != nil {
		return false
	}
	write()
	return true
}

func (d DeductFeeDecorator) debit(ctx sdk.Context, addr sdk.AccAddress, fee sdk.Coins) error {
	if d.ak.GetAccount(ctx, addr) == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", addr)
	}
	if fee.IsZero() {
		return nil
	}
	if !fee.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fee)
	}
	if err := d.bk.SendCoinsFromAccountToModule(ctx, addr, authtypes.FeeCollectorName, fee); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
	}
	return nil
}

// checkTxFeeWithValidatorMinGasPrices is the SDK default fee checker.
func checkTxFeeWithValidatorMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	fee, gas := feeTx.GetFee(), feeTx.GetGas()
	if ctx.IsCheckTx() && !ctx.MinGasPrices().IsZero() {
		glDec := sdkmath.LegacyNewDec(int64(gas))
		required := make(sdk.Coins, len(ctx.MinGasPrices()))
		for i, gp := range ctx.MinGasPrices() {
			required[i] = sdk.NewCoin(gp.Denom, gp.Amount.Mul(glDec).Ceil().RoundInt())
		}
		if !fee.IsAnyGTE(required) {
			return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", fee, required)
		}
	}
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		if gp := c.Amount.QuoRaw(int64(gas)); gp.IsInt64() {
			p = gp.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}
	return fee, priority, nil
}
