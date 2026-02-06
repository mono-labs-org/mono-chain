package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/mono module sentinel errors
var (
	ErrInvalidSigner          = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidFeeBurnPercent  = errors.Register(ModuleName, 1101, "invalid fee burn percent")
	ErrInvalidRegistrationFee = errors.Register(ModuleName, 1102, "invalid validator registration fee")
	ErrFeeSplitFailed         = errors.Register(ModuleName, 1103, "fee split processing failed")
	ErrProposerNotFound       = errors.Register(ModuleName, 1104, "block proposer not found")
)
