package app

import (
	"encoding/json"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// GenesisState of the blockchain is represented here as a map of raw json
// messages key'd by a identifier string.
// The identifier is used to determine which module genesis information belongs
// to so it may be appropriately routed during init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager which populates json from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage

func (app *App) NewEVMGenesisState(genesisState map[string]json.RawMessage) {
	if raw, ok := genesisState[evmtypes.ModuleName]; ok {
		var evmGenesis evmtypes.GenesisState
		app.appCodec.MustUnmarshalJSON(raw, &evmGenesis)
		evmGenesis.Preinstalls = evmtypes.DefaultPreinstalls
		genesisState[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(&evmGenesis)
	}
}
