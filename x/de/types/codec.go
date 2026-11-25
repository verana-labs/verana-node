package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/de/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgGrantDelegateAuthorization{}, "verana/x/de/MsgGrantDelegateAuthz")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeDelegateAuthorization{}, "verana/x/de/MsgRevokeDelegateAuthz")
	legacy.RegisterAminoMsg(cdc, &MsgGrantSponsorship{}, "verana/x/de/MsgGrantSponsorship")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeSponsorship{}, "verana/x/de/MsgRevokeSponsorship")
}

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgGrantDelegateAuthorization{},
		&MsgRevokeDelegateAuthorization{},
		&MsgGrantSponsorship{},
		&MsgRevokeSponsorship{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
