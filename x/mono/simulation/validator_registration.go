package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	"github.com/monolythium/mono-chain/x/mono/types"
)

// SimulateValidatorRegistration generates a multi-msg tx containing
// MsgBurn (registration fee) + MsgCreateValidator, mirroring the
// real validator registration flow enforced by ValidatorRegistrationBurnDecorator.
func SimulateValidatorRegistration(
	ak types.AuthKeeper,
	bk types.BankKeeper,
	mk keeper.Keeper,
	txGen client.TxConfig,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		msgType := sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{})

		params, err := mk.Params.Get(ctx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "failed to read params"), nil, nil
		}

		regFee := params.ValidatorRegistrationFee
		if regFee.IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "registration fee disabled"), nil, nil
		}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)
		if account == nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "account not found"), nil, nil
		}

		spendable := bk.SpendableCoins(ctx, account.GetAddress())
		bondDenom := sdk.DefaultBondDenom
		available := spendable.AmountOf(bondDenom)

		// Self-delegation matches registration fee amount
		selfDelegationAmt := regFee.Amount
		totalNeeded := regFee.Amount.Add(selfDelegationAmt)

		if available.LT(totalNeeded) {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "insufficient balance"), nil, nil
		}

		selfDelegation := sdk.NewCoin(bondDenom, selfDelegationAmt)

		// Calculate fees from remaining balance
		totalSpent := sdk.NewCoins(sdk.NewCoin(bondDenom, totalNeeded))
		remaining, hasNeg := spendable.SafeSub(totalSpent...)
		if hasNeg {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "insufficient for fees"), nil, nil
		}
		fees, err := simtypes.RandomFees(r, ctx, remaining)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "random fees failed"), nil, nil
		}

		msgBurn := &burnmoduletypes.MsgBurn{
			FromAddress: simAccount.Address.String(),
			Amount:      regFee,
		}

		valAddr := sdk.ValAddress(simAccount.Address)

		msgCreateVal, err := stakingtypes.NewMsgCreateValidator(
			valAddr.String(),
			simAccount.ConsKey.PubKey(),
			selfDelegation,
			stakingtypes.Description{Moniker: simtypes.RandStringOfLength(r, 10)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 2),
				MaxRate:       math.LegacyNewDecWithPrec(20, 2),
				MaxChangeRate: math.LegacyNewDecWithPrec(1, 2),
			},
			selfDelegationAmt,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "create validator msg failed"), nil, nil
		}

		tx, err := sims.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msgBurn, msgCreateVal},
			fees,
			sims.DefaultGenTxGas,
			chainID,
			[]uint64{account.GetAccountNumber()},
			[]uint64{account.GetSequence()},
			simAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, "tx sign failed"), nil, nil
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msgType, err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(msgCreateVal, true, ""), nil, nil
	}
}
