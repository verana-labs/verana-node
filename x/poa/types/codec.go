package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgAddValidator{}, "verana/x/poa/MsgAddValidator")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveValidator{}, "verana/x/poa/MsgRemoveValidator")
	legacy.RegisterAminoMsg(cdc, &MsgSelfRemoveValidator{}, "verana/x/poa/MsgSelfRemoveValidator")
}

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddValidator{},
		&MsgRemoveValidator{},
		&MsgSelfRemoveValidator{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
