package ante

import (
	"bytes"

	"cosmossdk.io/core/address"
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
	monoKeeper            keeper.Keeper
	accountAddressCodec   address.Codec
	validatorAddressCodec address.Codec
}

func NewValidatorRegistrationBurnDecorator(mk keeper.Keeper, accCodec, valCodec address.Codec) ValidatorRegistrationBurnDecorator {
	return ValidatorRegistrationBurnDecorator{
		monoKeeper:            mk,
		accountAddressCodec:   accCodec,
		validatorAddressCodec: valCodec,
	}
}

func (vbd ValidatorRegistrationBurnDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Skip during genesis
	// Initial validator set doesn't require burn
	if !ctx.IsCheckTx() && (ctx.BlockHeight() == 0) {
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

	valAddrBytes, err := vbd.validatorAddressCodec.StringToBytes(createMsg.ValidatorAddress)
	if err != nil {
		return ctx, errorsmod.Wrapf(types.ErrInvalidValidatorAddress, "failed to decode validator address: %s", err)
	}

	burnAddrBytes, err := vbd.accountAddressCodec.StringToBytes(burnMsg.FromAddress)
	if err != nil {
		return ctx, errorsmod.Wrapf(types.ErrInvalidBurnAddress, "failed to decode burn from address: %s", err)
	}

	if !bytes.Equal(valAddrBytes, burnAddrBytes) {
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

	if createMsg.MinSelfDelegation.LT(params.ValidatorRegistrationFee.Amount) {
		return ctx, errorsmod.Wrapf(
			types.ErrInsufficientMinSelfDelegation,
			"minimum self-delegation must be at least %s %s",
			params.ValidatorRegistrationFee.Amount,
			params.ValidatorRegistrationFee.Denom,
		)
	}

	return next(ctx, tx, simulate)
}
