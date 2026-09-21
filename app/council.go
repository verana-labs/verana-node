package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	poaante "github.com/verana-labs/verana-node/x/poa/ante"
	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

var (
	CouncilAuthority = poatypes.CouncilAuthorityBech32(AccountAddressPrefix)
	// FrozenAuthority has no signer; params behind it change only by software upgrade.
	FrozenAuthority = mustBech32(authtypes.NewModuleAddress("frozen_authority"))
)

func mustBech32(addr sdk.AccAddress) string {
	s, err := sdk.Bech32ifyAddressBytes(AccountAddressPrefix, addr)
	if err != nil {
		panic(err)
	}
	return s
}

func (app *App) setAnteHandler() error {
	base := app.AnteHandler()
	if base == nil {
		return fmt.Errorf("no ante handler installed")
	}
	poa := poaante.NewDecorator(func() bool { return app.inGenesis }, app.PoaKeeper)
	app.SetAnteHandler(func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return poa.AnteHandle(ctx, tx, simulate, base)
	})
	return nil
}

func (app *App) assertGenesisCouncil(ctx sdk.Context) error {
	policy, err := app.GroupKeeper.GroupPolicyInfo(ctx, &group.QueryGroupPolicyInfoRequest{Address: CouncilAuthority})
	if err != nil {
		return fmt.Errorf("genesis has no council group policy at %s: %w", CouncilAuthority, err)
	}
	if policy.Info.Admin != CouncilAuthority {
		return fmt.Errorf("council policy admin must be the policy itself, got %s", policy.Info.Admin)
	}
	info, err := app.GroupKeeper.GroupInfo(ctx, &group.QueryGroupInfoRequest{GroupId: policy.Info.GroupId})
	if err != nil {
		return err
	}
	if info.Info.Admin != CouncilAuthority {
		return fmt.Errorf("council group admin must be the policy, got %s", info.Info.Admin)
	}
	members, err := app.GroupKeeper.GroupMembers(ctx, &group.QueryGroupMembersRequest{GroupId: policy.Info.GroupId, Pagination: &query.PageRequest{Limit: 1000}})
	if err != nil {
		return err
	}
	if len(members.Members) == 0 {
		return fmt.Errorf("council has no members")
	}
	for _, m := range members.Members {
		if m.Member.Weight != "1" {
			return fmt.Errorf("council member %s has weight %s, want 1", m.Member.Address, m.Member.Weight)
		}
	}

	proposals, err := app.GroupKeeper.ProposalsByGroupPolicy(ctx, &group.QueryProposalsByGroupPolicyRequest{Address: CouncilAuthority, Pagination: &query.PageRequest{Limit: 1000}})
	if err != nil {
		return err
	}
	for _, p := range proposals.Proposals {
		if p.Status != group.PROPOSAL_STATUS_SUBMITTED && p.Status != group.PROPOSAL_STATUS_ACCEPTED {
			continue
		}
		msgs, err := p.GetMsgs()
		if err != nil {
			return err
		}
		if err := app.PoaKeeper.CheckMessages(ctx, msgs, false); err != nil {
			return fmt.Errorf("council proposal %d in genesis: %w", p.Id, err)
		}
	}

	bond, err := app.PoaKeeper.ValidatorBond(ctx)
	if err != nil {
		return err
	}
	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return err
	}
	for _, v := range validators {
		if v.Tokens.IsZero() {
			continue
		}
		if !v.Tokens.Equal(bond.Amount) {
			return fmt.Errorf("validator %s has %s tokens, want %s", v.OperatorAddress, v.Tokens, bond.Amount)
		}
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(v.OperatorAddress)
		if err != nil {
			return err
		}
		member, err := app.PoaKeeper.IsCouncilMember(ctx, sdk.AccAddress(valBz).String())
		if err != nil {
			return err
		}
		if !member {
			return fmt.Errorf("validator %s is not a council member", v.OperatorAddress)
		}
	}
	return app.assertICAHostFenced(ctx)
}

func (app *App) assertICAHostFenced(ctx sdk.Context) error {
	params := app.ICAHostKeeper.GetParams(ctx)
	if !params.HostEnabled {
		return nil
	}
	fenced := map[string]bool{
		sdk.MsgTypeURL(&authz.MsgExec{}):                             true,
		sdk.MsgTypeURL(&group.MsgSubmitProposal{}):                   true,
		sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{}):           true,
		sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}):                  true,
		sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{}):                true,
		sdk.MsgTypeURL(&stakingtypes.MsgBeginRedelegate{}):           true,
		sdk.MsgTypeURL(&stakingtypes.MsgCancelUnbondingDelegation{}): true,
	}
	for _, m := range params.AllowMessages {
		if m == "*" || fenced[m] {
			return fmt.Errorf("ica host allow_messages must not include %q", m)
		}
	}
	return nil
}
