package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/gf/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}
	// [NGG] Version 1 of the network framework is created and active at
	// genesis time unless the genesis sets those times.
	var genesisNGF uint64
	if ids := gs.NetworkGovernance.NetworkGovernanceFrameworkVersions; len(ids) > 0 {
		genesisNGF = ids[0]
	}
	now := ctx.BlockTime()
	var maxGFV, maxGFD uint64
	for _, gfv := range gs.Versions {
		if gfv.Id == genesisNGF {
			if gfv.Created.IsZero() {
				gfv.Created = now
			}
			if gfv.ActiveSince == nil {
				gfv.ActiveSince = &now
			}
		}
		if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return err
		}
		switch {
		case gfv.EcosystemId > 0:
			if err := k.GFVersionByEcosystem.Set(ctx, collections.Join(gfv.EcosystemId, gfv.Version), gfv.Id); err != nil {
				return err
			}
		case gfv.CorporationId > 0:
			if err := k.GFVersionByCorporation.Set(ctx, collections.Join(gfv.CorporationId, gfv.Version), gfv.Id); err != nil {
				return err
			}
		}
		if gfv.Id > maxGFV {
			maxGFV = gfv.Id
		}
	}
	for _, gfd := range gs.Documents {
		if gfd.GfvId == genesisNGF && gfd.Created.IsZero() {
			gfd.Created = now
		}
		if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
			return err
		}
		if err := k.GFDocumentByGFVLang.Set(ctx, collections.Join(gfd.GfvId, gfd.Language), gfd.Id); err != nil {
			return err
		}
		if gfd.Id > maxGFD {
			maxGFD = gfd.Id
		}
	}
	if maxGFV < gs.GfvCounter {
		maxGFV = gs.GfvCounter
	}
	if maxGFD < gs.GfdCounter {
		maxGFD = gs.GfdCounter
	}
	if err := k.Counter.Set(ctx, "gfv", maxGFV); err != nil {
		return err
	}
	if err := k.Counter.Set(ctx, "gfd", maxGFD); err != nil {
		return err
	}
	if err := k.NetworkGovernance.Set(ctx, gs.NetworkGovernance); err != nil {
		return err
	}
	for _, acc := range gs.EcosystemAcceptances {
		if err := k.setEcosystemAcceptance(ctx, acc); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
	}
	if err := k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		gs.Versions = append(gs.Versions, gfv)
		return false, nil
	}); err != nil {
		panic(err)
	}
	if err := k.GFDocument.Walk(ctx, nil, func(_ uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		gs.Documents = append(gs.Documents, gfd)
		return false, nil
	}); err != nil {
		panic(err)
	}
	if c, err := k.Counter.Get(ctx, "gfv"); err == nil {
		gs.GfvCounter = c
	} else {
		for _, gfv := range gs.Versions {
			if gfv.Id > gs.GfvCounter {
				gs.GfvCounter = gfv.Id
			}
		}
	}
	if c, err := k.Counter.Get(ctx, "gfd"); err == nil {
		gs.GfdCounter = c
	} else {
		for _, gfd := range gs.Documents {
			if gfd.Id > gs.GfdCounter {
				gs.GfdCounter = gfd.Id
			}
		}
	}
	ng, err := k.getNetworkGovernance(ctx)
	if err != nil {
		panic(err)
	}
	gs.NetworkGovernance = ng
	if err := k.EcosystemAcceptances.Walk(ctx, nil, func(key collections.Pair[uint64, uint64], acc types.EcosystemAcceptance) (bool, error) {
		acc.EcosystemId, acc.CorporationId = key.K1(), key.K2()
		gs.EcosystemAcceptances = append(gs.EcosystemAcceptances, acc)
		return false, nil
	}); err != nil {
		panic(err)
	}
	return gs
}
