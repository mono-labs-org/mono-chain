package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/monolythium/mono-chain/x/mono/types"
)

// ProcessFeeSplit burns fee_burn_percent of accumulated transaction fees
// and sends the remainder to the block proposer.
// Called in BeginBlocker, BEFORE the distribution module's BeginBlocker.
func (k Keeper) ProcessFeeSplit(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get mono params")
	}

	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	fees := k.bankKeeper.GetAllBalances(ctx, feeCollectorAddr)
	if fees.IsZero() {
		return nil
	}

	feeBurnPercent := params.FeeBurnPercent

	// Compute amounts
	var burnCoins sdk.Coins
	var proposerCoins sdk.Coins

	for _, coin := range fees {
		if coin.IsZero() {
			continue
		}

		burnAmt := coin.Amount.ToLegacyDec().Mul(feeBurnPercent).TruncateInt()
		proposerAmt := coin.Amount.Sub(burnAmt)

		if burnAmt.IsPositive() {
			burnCoins = burnCoins.Add(sdk.NewCoin(coin.Denom, burnAmt))
		}
		if proposerAmt.IsPositive() {
			proposerCoins = proposerCoins.Add(sdk.NewCoin(coin.Denom, proposerAmt))
		}
	}

	// Burn portion: fee_collector -> mono module -> burn
	if burnCoins.IsAllPositive() {
		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.ModuleName, burnCoins)
		if err != nil {
			return errorsmod.Wrap(types.ErrFeeSplitFailed, "transfer to burn intermediary failed")
		}

		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins)
		if err != nil {
			return errorsmod.Wrap(types.ErrFeeSplitFailed, "burn failed after transfer")
		}
	}

	// Proposer: resolve and send
	if proposerCoins.IsAllPositive() {
		proposerAddr := sdkCtx.BlockHeader().ProposerAddress
		if len(proposerAddr) == 0 {
			// No proposer (genesis block). Burn any remaining fees.
			err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.ModuleName, proposerCoins)
			if err != nil {
				return errorsmod.Wrap(types.ErrFeeSplitFailed, "transfer remainder to burn failed")
			}
			err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, proposerCoins)
			if err != nil {
				return errorsmod.Wrap(types.ErrFeeSplitFailed, "burn remainder failed")
			}
		} else {
			accAddr, resolveErr := k.resolveProposerAccount(ctx, sdk.ConsAddress(proposerAddr))
			if resolveErr != nil {
				// Proposer can't be resolved. Burn instead of losing funds
				sdkCtx.Logger().Error("fee split: proposer not found, burning remainder",
					"consAddr", sdk.ConsAddress(proposerAddr).String(),
					"error", resolveErr,
				)
				err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.ModuleName, proposerCoins)
				if err != nil {
					return errorsmod.Wrap(types.ErrFeeSplitFailed, "fallback burn transfer failed")
				}
				err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, proposerCoins)
				if err != nil {
					return errorsmod.Wrap(types.ErrFeeSplitFailed, "fallback burn failed")
				}
			} else {
				err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, accAddr, proposerCoins)
				if err != nil {
					return errorsmod.Wrap(types.ErrFeeSplitFailed, "send to proposer failed")
				}
			}
		}
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeFeeSplit,
			sdk.NewAttribute(types.AttributeKeyBurnAmount, burnCoins.String()),
			sdk.NewAttribute(types.AttributeKeyProposerReward, proposerCoins.String()),
		),
	)

	return nil
}

// resolveProposerAccount converts a consensus address to an account address
// by looking up the validator in the staking module.
func (k Keeper) resolveProposerAccount(ctx context.Context, consAddr sdk.ConsAddress) (sdk.AccAddress, error) {
	validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrProposerNotFound, "validator lookup failed: %v", err)
	}
	if validator == nil {
		return nil, errorsmod.Wrap(types.ErrProposerNotFound, "nil validator for consensus address")
	}

	operAddrStr := validator.GetOperator()
	operAddr, err := sdk.ValAddressFromBech32(operAddrStr)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrProposerNotFound, "invalid operator address %s: %v", operAddrStr, err)
	}

	return sdk.AccAddress(operAddr), nil
}
