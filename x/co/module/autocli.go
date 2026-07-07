package co

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana-node/api/verana/co/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "GetCorporation",
					Use:            "get-corporation [corporation-id]",
					Short:          "Get a Corporation by ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "corporation_id"}},
				},
				{
					RpcMethod: "ListCorporations",
					Use:       "list-corporations",
					Short:     "List Corporations (ordered by modified DESC)",
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Get the current MOD-CO module parameters",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{RpcMethod: "UpdateParams", Skip: true}, // governance-only
			},
		},
	}
}
