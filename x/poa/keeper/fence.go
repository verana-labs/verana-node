package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

// MsgCreateValidator is allowed only while gentxs are delivered at genesis. Runs at
// submission; x/group executes proposals without the ante and seat changes do not abort open proposals.
func (k Keeper) CheckMessages(ctx context.Context, msgs []sdk.Msg, inGenesis bool) error {
	unseated := map[string]bool{}
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *types.MsgRemoveValidator:
			if addr, err := k.canonical(m.ValidatorAddress); err == nil {
				unseated[addr] = true
			}
		case *types.MsgSelfRemoveValidator:
			if addr, err := k.canonical(m.ValidatorAddress); err == nil {
				unseated[addr] = true
			}
		}
	}
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *stakingtypes.MsgDelegate, *stakingtypes.MsgUndelegate, *stakingtypes.MsgBeginRedelegate, *stakingtypes.MsgCancelUnbondingDelegation:
			return errorsmod.Wrap(types.ErrMsgNotAllowed, sdk.MsgTypeURL(msg))
		case *stakingtypes.MsgCreateValidator:
			if err := k.checkGenesisValidator(ctx, m, inGenesis); err != nil {
				return err
			}
		case *group.MsgUpdateGroupAdmin:
			if err := k.checkCouncilAdmin(ctx, m.GroupId, "", m.NewAdmin); err != nil {
				return err
			}
		case *group.MsgUpdateGroupPolicyAdmin:
			if err := k.checkCouncilAdmin(ctx, 0, m.GroupPolicyAddress, m.NewAdmin); err != nil {
				return err
			}
		case *group.MsgUpdateGroupPolicyDecisionPolicy:
			if err := k.checkCouncilDecisionPolicy(ctx, m); err != nil {
				return err
			}
		case *group.MsgUpdateGroupMembers:
			if err := k.checkCouncilMembers(ctx, m, unseated); err != nil {
				return err
			}
		case *group.MsgLeaveGroup:
			if err := k.checkCouncilLeave(ctx, m, unseated); err != nil {
				return err
			}
		case *authz.MsgExec:
			inner, err := m.GetMessages()
			if err != nil {
				return err
			}
			if err := k.CheckMessages(ctx, inner, inGenesis); err != nil {
				return err
			}
		case *group.MsgSubmitProposal:
			inner, err := m.GetMsgs()
			if err != nil {
				return err
			}
			if err := k.CheckMessages(ctx, inner, inGenesis); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) checkGenesisValidator(ctx context.Context, m *stakingtypes.MsgCreateValidator, inGenesis bool) error {
	if !inGenesis {
		return errorsmod.Wrap(types.ErrMsgNotAllowed, "validators are seated by council proposal")
	}
	bond, err := k.ValidatorBond(ctx)
	if err != nil {
		return err
	}
	if !m.Value.Equal(bond) {
		return errorsmod.Wrapf(types.ErrInvalidBond, "got %s want %s", m.Value, bond)
	}
	valAddr, err := k.staking.ValidatorAddressCodec().StringToBytes(m.ValidatorAddress)
	if err != nil {
		return err
	}
	operator, err := k.addressCodec.BytesToString(valAddr)
	if err != nil {
		return err
	}
	member, err := k.IsCouncilMember(ctx, operator)
	if err != nil {
		return err
	}
	if !member {
		return errorsmod.Wrap(types.ErrNotCouncilMember, m.ValidatorAddress)
	}
	return nil
}

// canonical rewrites a bech32 address the way the keeper compares it; bech32 is case-insensitive.
func (k Keeper) canonical(addr string) (string, error) {
	bz, err := k.addressCodec.StringToBytes(addr)
	if err != nil {
		return "", err
	}
	return k.addressCodec.BytesToString(bz)
}

func (k Keeper) isCouncil(ctx context.Context, groupID uint64, policy string) (bool, error) {
	if policy != "" {
		addr, err := k.canonical(policy)
		if err != nil {
			return false, err
		}
		return addr == k.authority, nil
	}
	councilID, err := k.CouncilGroupID(ctx)
	if err != nil {
		return false, err
	}
	return groupID == councilID, nil
}

func (k Keeper) checkCouncilAdmin(ctx context.Context, groupID uint64, policy, newAdmin string) error {
	council, err := k.isCouncil(ctx, groupID, policy)
	if err != nil || !council {
		return err
	}
	admin, err := k.canonical(newAdmin)
	if err != nil {
		return err
	}
	if admin != k.authority {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council admin must stay the council policy")
	}
	return nil
}

func (k Keeper) checkCouncilDecisionPolicy(ctx context.Context, m *group.MsgUpdateGroupPolicyDecisionPolicy) error {
	council, err := k.isCouncil(ctx, 0, m.GroupPolicyAddress)
	if err != nil || !council {
		return err
	}
	policy, err := m.GetDecisionPolicy()
	if err != nil {
		return err
	}
	pct, ok := policy.(*group.PercentageDecisionPolicy)
	if !ok {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council policy must stay a percentage policy")
	}
	got, err := math.LegacyNewDecFromStr(pct.Percentage)
	if err != nil {
		return err
	}
	if got.LT(math.LegacyMustNewDecFromStr(types.CouncilPercentage)) {
		return errorsmod.Wrapf(types.ErrCouncilIntegrity, "council percentage %s below %s", pct.Percentage, types.CouncilPercentage)
	}
	if pct.Windows == nil {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council windows are required")
	}
	current, err := k.councilPolicy(ctx)
	if err != nil {
		return err
	}
	if pct.Windows.VotingPeriod < current.GetVotingPeriod() {
		return errorsmod.Wrapf(types.ErrCouncilIntegrity, "council voting period %s below %s", pct.Windows.VotingPeriod, current.GetVotingPeriod())
	}
	if pct.Windows.MinExecutionPeriod < current.GetMinExecutionPeriod() {
		return errorsmod.Wrapf(types.ErrCouncilIntegrity, "council min execution period %s below %s", pct.Windows.MinExecutionPeriod, current.GetMinExecutionPeriod())
	}
	return nil
}

func (k Keeper) checkCouncilMembers(ctx context.Context, m *group.MsgUpdateGroupMembers, unseated map[string]bool) error {
	council, err := k.isCouncil(ctx, m.GroupId, "")
	if err != nil || !council {
		return err
	}
	members, err := k.CouncilMembers(ctx)
	if err != nil {
		return err
	}
	seats := map[string]bool{}
	for _, a := range members {
		seats[a] = true
	}
	for _, u := range m.MemberUpdates {
		addr, err := k.canonical(u.Address)
		if err != nil {
			return err
		}
		switch u.Weight {
		case "0":
			if err := k.checkSeatRelease(ctx, addr, unseated); err != nil {
				return err
			}
			delete(seats, addr)
		case "1":
			seats[addr] = true
		default:
			return errorsmod.Wrapf(types.ErrCouncilIntegrity, "seat weight must be 1, got %s", u.Weight)
		}
	}
	if len(seats) == 0 {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council cannot be emptied")
	}
	return nil
}

func (k Keeper) checkCouncilLeave(ctx context.Context, m *group.MsgLeaveGroup, unseated map[string]bool) error {
	council, err := k.isCouncil(ctx, m.GroupId, "")
	if err != nil || !council {
		return err
	}
	addr, err := k.canonical(m.Address)
	if err != nil {
		return err
	}
	if err := k.checkSeatRelease(ctx, addr, unseated); err != nil {
		return err
	}
	members, err := k.CouncilMembers(ctx)
	if err != nil {
		return err
	}
	if len(members) <= 1 {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "the last seat cannot leave")
	}
	return nil
}

// A seat whose validator still holds tokens leaves only together with that validator.
func (k Keeper) checkSeatRelease(ctx context.Context, addr string, unseated map[string]bool) error {
	if unseated[addr] {
		return nil
	}
	active, err := k.hasActiveValidator(ctx, addr)
	if err != nil {
		return err
	}
	if active {
		return errorsmod.Wrapf(types.ErrCouncilIntegrity, "seat %s still runs a validator; remove it in the same tx or proposal", addr)
	}
	return nil
}
