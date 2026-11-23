package v0_10_7_test

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/stretchr/testify/require"

	v0_10_7 "github.com/verana-labs/verana-node/app/upgrades/v0_10_7"
)

type paramStore struct {
	cp cmtproto.ConsensusParams
}

func (p *paramStore) GetConsensusParams(sdk.Context) cmtproto.ConsensusParams { return p.cp }
func (p *paramStore) StoreConsensusParams(_ sdk.Context, cp cmtproto.ConsensusParams) error {
	p.cp = cp
	return nil
}

func TestUpgradeHandler_SetsBlockMaxGas(t *testing.T) {
	store := &paramStore{cp: cmtproto.ConsensusParams{Block: &cmtproto.BlockParams{MaxBytes: 22020096, MaxGas: -1}}}
	cfg := module.NewConfigurator(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), baseapp.NewMsgServiceRouter(), baseapp.NewGRPCQueryRouter())
	handler := v0_10_7.CreateUpgradeHandler(module.NewManager(), cfg, store, nil)

	_, err := handler(sdk.Context{}, upgradetypes.Plan{Name: v0_10_7.UpgradeName}, module.VersionMap{})
	require.NoError(t, err)
	require.Equal(t, v0_10_7.BlockMaxGas, store.cp.Block.MaxGas)
	require.Equal(t, int64(22020096), store.cp.Block.MaxBytes)
}
