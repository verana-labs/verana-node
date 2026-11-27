package gf

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana-node/api/verana/gf/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GetGovernanceFrameworkVersion",
					Use:       "get-governance-framework-version [id]",
					Short:     "Get a GovernanceFrameworkVersion by ID with its nested documents",
					Long:      "Get a GovernanceFrameworkVersion by ID. Returns the version + all nested GovernanceFrameworkDocument entries. Use --preferred-language to filter to one document per version.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"preferred_language": {
							Name:         "preferred-language",
							DefaultValue: "",
							Usage:        "If set, return one document: in this BCP 47 language, else in the owner's language, else the earliest.",
						},
					},
				},
				{
					RpcMethod: "GetEcosystemAcceptance",
					Use:       "get-ecosystem-acceptance [ecosystem-id] [corporation-id]",
					Short:     "Get the last version of an ecosystem's governance framework a corporation accepted",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "ecosystem_id"},
						{ProtoField: "corporation_id"},
					},
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Get the current MOD-GF module parameters",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "AddGovernanceFrameworkDocument",
					Skip:      true, // custom command in cli_tx.go
				},
				{
					RpcMethod: "IncreaseActiveGovernanceFrameworkVersion",
					Skip:      true, // custom command in cli_tx.go
				},
				{
					RpcMethod: "AcceptGovernanceFrameworkVersion",
					Use:       "accept-gf-version [corporation] [version]",
					Short:     "[MOD-GF-MSG-3] Accept the version in force of the network governance framework (--network) or of an ecosystem's (--ecosystem-id)",
					Long:      "The signer (--from) is an authorized delegate, or the corporation's policy_address on the group-proposal path.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "version"},
					},
				},
				{
					RpcMethod: "AcceptCorporationGovernanceFramework",
					Use:       "accept-corporation-gf [corporation-id] [version]",
					Short:     "[MOD-GF-MSG-4] Accept, as its delegate, the version in force of a corporation's governance framework",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation_id"},
						{ProtoField: "version"},
					},
				},
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // governance-only
				},
			},
		},
	}
}
