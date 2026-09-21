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

// MsgCreateValidator is allowed only while gentxs are delivered at genesis.
func (k Keeper) CheckMessages(ctx context.Context, msgs []sdk.Msg, inGenesis bool) error {
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
			if err := k.checkCouncilDecisionPolicy(m); err != nil {
				return err
			}
		case *group.MsgUpdateGroupMembers:
			if err := k.checkCouncilMembers(ctx, m); err != nil {
				return err
			}
		case *group.MsgLeaveGroup:
			if err := k.checkCouncilLeave(ctx, m); err != nil {
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

func (k Keeper) isCouncil(ctx context.Context, groupID uint64, policy string) (bool, error) {
	if policy != "" {
		return policy == k.authority, nil
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
	if newAdmin != k.authority {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council admin must stay the council policy")
	}
	return nil
}

func (k Keeper) checkCouncilDecisionPolicy(m *group.MsgUpdateGroupPolicyDecisionPolicy) error {
	if m.GroupPolicyAddress != k.authority {
		return nil
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
	return nil
}

func (k Keeper) checkCouncilMembers(ctx context.Context, m *group.MsgUpdateGroupMembers) error {
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
		switch u.Weight {
		case "0":
			delete(seats, u.Address)
		case "1":
			seats[u.Address] = true
		default:
			return errorsmod.Wrapf(types.ErrCouncilIntegrity, "seat weight must be 1, got %s", u.Weight)
		}
	}
	if len(seats) == 0 {
		return errorsmod.Wrap(types.ErrCouncilIntegrity, "council cannot be emptied")
	}
	return nil
}

func (k Keeper) checkCouncilLeave(ctx context.Context, m *group.MsgLeaveGroup) error {
	council, err := k.isCouncil(ctx, m.GroupId, "")
	if err != nil || !council {
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
