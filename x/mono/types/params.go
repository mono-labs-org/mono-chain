package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	DefaultFeeBurnPercent = math.LegacyZeroDec()

	DefaultValidatorRegistrationFee = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
)

func NewParams(
	feeBurnPercent math.LegacyDec,
	validatorRegistrationFee sdk.Coin,
) Params {
	return Params{
		FeeBurnPercent:           feeBurnPercent,
		ValidatorRegistrationFee: validatorRegistrationFee,
	}
}

func DefaultParams() Params {
	return NewParams(
		DefaultFeeBurnPercent,
		DefaultValidatorRegistrationFee,
	)
}

func (p Params) Validate() error {
	if err := validateFeeBurnPercent(p.FeeBurnPercent); err != nil {
		return err
	}

	if err := validateValidatorRegistrationFee(p.ValidatorRegistrationFee); err != nil {
		return err
	}

	return nil
}

func validateFeeBurnPercent(v math.LegacyDec) error {
	if v.IsNil() {
		return errorsmod.Wrap(ErrInvalidFeeBurnPercent, "must not be nil")
	}
	if v.IsNegative() {
		return errorsmod.Wrapf(ErrInvalidFeeBurnPercent, "must not be negative: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return errorsmod.Wrapf(ErrInvalidFeeBurnPercent, "must not exceed 1.0: %s", v)
	}
	return nil
}

func validateValidatorRegistrationFee(v sdk.Coin) error {
	if err := v.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidRegistrationFee, "%s", err)
	}
	if !v.IsZero() && v.Denom != sdk.DefaultBondDenom {
		return errorsmod.Wrapf(ErrInvalidRegistrationFee, "denom must be %s, got %s", sdk.DefaultBondDenom, v.Denom)
	}
	return nil
}
