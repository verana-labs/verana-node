package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	deante "github.com/verana-labs/verana-node/x/de/ante"
)

// newAnteHandler is the SDK default chain with the sponsored-fees decorator in
// place of the SDK fee decorator.
func (app *App) newAnteHandler() sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(app.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
		deante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.DeKeeper, nil),
		ante.NewSetPubKeyDecorator(app.AccountKeeper),
		ante.NewValidateSigCountDecorator(app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
		ante.NewIncrementSequenceDecorator(app.AccountKeeper),
	)
}
