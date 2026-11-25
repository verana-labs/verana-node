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
							Usage:        "If set, return only the document matching this BCP 47 language (falls back to all docs if no match).",
						},
					},
				},
				{
					RpcMethod: "ListGovernanceFrameworkVersions",
					Use:       "list-governance-framework-versions",
					Short:     "List GovernanceFrameworkVersions for an Ecosystem or Corporation",
					Long:      "List GovernanceFrameworkVersion entries owned by the target subject. Exactly one of --ecosystem-id and --corporation-id must be set. Results ordered by ascending version.",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"ecosystem_id": {
							Name:         "ecosystem-id",
							DefaultValue: "0",
							Usage:        "Ecosystem ID (XOR with --corporation-id).",
						},
						"corporation_id": {
							Name:         "corporation-id",
							DefaultValue: "0",
							Usage:        "Corporation ID (uint64, XOR with --ecosystem-id).",
						},
						"active_only": {
							Name:         "active-only",
							DefaultValue: "false",
							Usage:        "If true, return only the entry corresponding to the subject's active_version.",
						},
						"preferred_language": {
							Name:         "preferred-language",
							DefaultValue: "",
							Usage:        "If set, return only one document per version, preferring this language.",
						},
						"response_max_size": {
							Name:         "response-max-size",
							DefaultValue: "64",
							Usage:        "Maximum results to return (1-1024, default 64).",
						},
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
					RpcMethod: "UpdateParams",
					Skip:      true, // governance-only
				},
			},
		},
	}
}
