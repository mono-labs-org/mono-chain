package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	monoante "github.com/monolythium/mono-chain/x/mono/ante"
	monomodulekeeper "github.com/monolythium/mono-chain/x/mono/keeper"
)

// HandlerOptions extends the SDK's default ante handler options
// with application-specific keepers.
type HandlerOptions struct {
	ante.HandlerOptions
	MonoKeeper monomodulekeeper.Keeper
}

// NewAnteHandler composes the SDK's default ante handler with
// application-specific decorators. Delegating to ante.NewAnteHandler
// ensures upstream decorator changes are inherited on SDK upgrades.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	sdkHandler, err := ante.NewAnteHandler(options.HandlerOptions)
	if err != nil {
		return nil, err
	}

	depositDecorator := monoante.NewValidatorRegistrationBurnDecorator(options.MonoKeeper)

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		newCtx, err := sdkHandler(ctx, tx, simulate)
		if err != nil {
			return newCtx, err
		}
		return depositDecorator.AnteHandle(newCtx, tx, simulate, passthrough)
	}, nil
}

func passthrough(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}
