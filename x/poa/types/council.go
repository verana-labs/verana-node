package types

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/verana-labs/cosmos-group"
)

// groupPolicyTablePrefix mirrors x/group's keeper.GroupPolicyTablePrefix (pinned by test).
const groupPolicyTablePrefix byte = 0x20

// GroupPolicyCredential is the module credential x/group derives for policy sequence seq.
func GroupPolicyCredential(seq uint64) *authtypes.ModuleCredential {
	dk := make([]byte, 8)
	binary.BigEndian.PutUint64(dk, seq)
	cred, err := authtypes.NewModuleCredential(group.ModuleName, []byte{groupPolicyTablePrefix}, dk)
	if err != nil {
		panic(err)
	}
	return cred
}

func GroupPolicyAddress(seq uint64) sdk.AccAddress {
	return sdk.AccAddress(GroupPolicyCredential(seq).Address())
}

func CouncilAuthorityBech32(prefix string) string {
	s, err := sdk.Bech32ifyAddressBytes(prefix, GroupPolicyAddress(1))
	if err != nil {
		panic(err)
	}
	return s
}

func ValidatorBond(denom string) sdk.Coin {
	return sdk.NewCoin(denom, sdk.DefaultPowerReduction)
}
