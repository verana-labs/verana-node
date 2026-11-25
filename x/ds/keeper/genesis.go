package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/ds/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.validateParams(genState.Params); err != nil {
		return err
	}
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}
	for _, vb := range genState.ValidatorBlocks {
		addr, err := k.staking.ValidatorAddressCodec().StringToBytes(vb.ValidatorAddress)
		if err != nil {
			return err
		}
		if err := k.ValidatorBlocks.Set(ctx, addr, types.ValidatorBlocks{SignedBlocks: vb.SignedBlocks, MemberBlocks: vb.MemberBlocks}); err != nil {
			return err
		}
	}
	if genState.EpochBlocks == 0 {
		return nil
	}
	return k.EpochBlocks.Set(ctx, genState.EpochBlocks)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	gs := &types.GenesisState{Params: params}
	err = k.ValidatorBlocks.Walk(ctx, nil, func(addr sdk.ValAddress, b types.ValidatorBlocks) (bool, error) {
		s, err := k.staking.ValidatorAddressCodec().BytesToString(addr)
		if err != nil {
			return true, err
		}
		gs.ValidatorBlocks = append(gs.ValidatorBlocks, types.GenesisValidatorBlocks{ValidatorAddress: s, SignedBlocks: b.SignedBlocks, MemberBlocks: b.MemberBlocks})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	gs.EpochBlocks, err = k.epochBlocks(ctx)
	if err != nil {
		return nil, err
	}
	return gs, nil
}
