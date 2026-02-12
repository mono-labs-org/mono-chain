package ante

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	"github.com/monolythium/mono-chain/x/mono/types"
)

// ValidatorRegistrationBurnDecorator enforces that any transaction containing
// MsgCreateValidator must also include a MsgBurn of at least
// validator_registration_fee from the same key as the validator operator.
type ValidatorRegistrationBurnDecorator struct {
	monoKeeper keeper.Keeper
}

func NewValidatorRegistrationBurnDecorator(mk keeper.Keeper) ValidatorRegistrationBurnDecorator {
	return ValidatorRegistrationBurnDecorator{monoKeeper: mk}
}

func (vbd ValidatorRegistrationBurnDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Skip during genesis — initial validator set doesn't require burn
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	var createMsg *stakingtypes.MsgCreateValidator
	var burnMsg *burnmoduletypes.MsgBurn
	for _, msgs := range tx.GetMsgs() {
		switch msg := msgs.(type) {
		case *stakingtypes.MsgCreateValidator:
			if createMsg != nil {
				return ctx, types.ErrDuplicateRegistrationInfo
			}
			createMsg = msg
		case *burnmoduletypes.MsgBurn:
			if burnMsg != nil {
				return ctx, types.ErrDuplicateRegistrationInfo
			}
			burnMsg = msg
		}
	}

	if createMsg == nil {
		return next(ctx, tx, simulate)
	}

	params, err := vbd.monoKeeper.Params.Get(ctx)
	if err != nil {
		return ctx, types.ErrParamsRead
	}

	if params.ValidatorRegistrationFee.IsZero() {
		return next(ctx, tx, simulate)
	}

	if burnMsg == nil {
		return ctx, types.ErrMissingBurnInfo
	}

	valAddr, err := sdk.ValAddressFromBech32(createMsg.ValidatorAddress)
	if err != nil {
		return ctx, types.ErrInvalidValidatorAddress
	}

	burnAddr, err := sdk.AccAddressFromBech32(burnMsg.FromAddress)
	if err != nil {
		return ctx, types.ErrInvalidBurnAddress
	}

	if !bytes.Equal(valAddr.Bytes(), burnAddr.Bytes()) {
		return ctx, types.ErrBurnSenderMismatch
	}

	if burnMsg.Amount.Denom != params.ValidatorRegistrationFee.Denom {
		return ctx, types.ErrBurnDenomMismatch
	}

	if burnMsg.Amount.Amount.LT(params.ValidatorRegistrationFee.Amount) {
		return ctx, errorsmod.Wrapf(
			types.ErrInsufficientBurnAmount,
			"Validator registration requires a burn of: %s %s",
			params.ValidatorRegistrationFee.Amount,
			params.ValidatorRegistrationFee.Denom,
		)
	}

	return next(ctx, tx, simulate)
}
