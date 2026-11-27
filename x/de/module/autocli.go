package de

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/verana-labs/verana-node/x/de/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Query_serviceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					// Skip autocli for this RPC -- custom command provided in cli_query.go
					// to work around gogo/pulsar proto codec mismatch in autocli JSON rendering.
					RpcMethod: "ListDelegateAuthorizations",
					Skip:      true,
				},
				{
					// Skip autocli for this RPC -- custom command provided in cli_query.go
					RpcMethod: "ListVSOperatorAuthorizations",
					Skip:      true,
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GrantDelegateAuthorization",
					Use:       "grant-delegate-authz [grantee]",
					Short:     "Grant delegate authorization to a grantee on behalf of a corporation",
					Long:      "[MOD-DE-MSG-3] Grant delegate authorization. The signer (--from) is the delegate: the corporation's policy_address on the group-proposal path, or an authorized delegate. The grantee receives authorization to execute specified message types on behalf of the corporation and cannot be the corporation's own policy_address. Optionally includes a fee grant.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "grantee"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "policy_address of the corporation on whose behalf the grant is made",
						},
						"msg_types": {
							Name:  "msg-types",
							Usage: "MsgType values, one per flag: --msg-types create-ecosystem --msg-types store-digest",
						},
						"expiration": {
							Name:  "expiration",
							Usage: "authorization expiration timestamp (RFC3339)",
						},
						"authz_spend_limit": {
							Name:  "authz-spend-limit",
							Usage: "maximum spendable amount (e.g. 100stake)",
						},
						"authz_spend_limit_period": {
							Name:  "authz-spend-limit-period",
							Usage: "reset period for authz spend limit",
						},
						"with_feegrant": {
							Name:         "with-feegrant",
							Usage:        "whether to also grant fee allowance",
							DefaultValue: "false",
						},
						"feegrant_spend_limit": {
							Name:  "feegrant-spend-limit",
							Usage: "maximum fee amount (e.g. 100stake). Ignored if --with-feegrant is false",
						},
						"feegrant_spend_limit_period": {
							Name:  "feegrant-spend-limit-period",
							Usage: "reset period for fee spend limit. Ignored if --with-feegrant is false",
						},
					},
				},
				{
					RpcMethod: "RevokeDelegateAuthorization",
					Use:       "revoke-delegate-authz [grantee]",
					Short:     "Revoke delegate authorization for a grantee",
					Long:      "[MOD-DE-MSG-4] Revoke delegate authorization. The signer (--from) is the delegate: the corporation's policy_address on the group-proposal path, or an authorized delegate. Removes the authorization entry and any associated fee grant for the given corporation/grantee pair.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "grantee"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "policy_address of the corporation on whose behalf the revoke is made",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
