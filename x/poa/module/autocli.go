package poa

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/verana-labs/verana-node/x/poa/types"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Council",
					Use:       "council",
					Short:     "Show the council authority, its members, and members without a bonded validator",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "AddValidator",
					Skip:      true, // authority gated
				},
				{
					RpcMethod: "RemoveValidator",
					Skip:      true, // authority gated
				},
				{
					RpcMethod: "SelfRemoveValidator",
					Use:       "self-remove-validator",
					Short:     "Remove your own validator and burn its bond",
				},
			},
		},
	}
}
