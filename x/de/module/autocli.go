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
				{RpcMethod: "GetSponsorship", Skip: true},
				{RpcMethod: "ListSponsorships", Skip: true},
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
					Long:      "[MOD-DE-MSG-3] Grant delegate authorization. The signer (--from) is an authorized delegate, or the corporation's policy_address on the group-proposal path. The grantee receives authorization to execute specified message types on behalf of the corporation and cannot be the corporation's own policy_address or the signer. A fee allowance is granted when --feegrant-spend-limit is set.",
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
							Usage: "end of the authorization, or of its first cycle with a spend limit period (RFC3339)",
						},
						"authz_spend_limit": {
							Name:  "authz-spend-limit",
							Usage: "maximum spendable amount (e.g. 100stake)",
						},
						"authz_spend_limit_period": {
							Name:  "authz-spend-limit-period",
							Usage: "reset period for the authz spend limit; requires --expiration",
						},
						"feegrant_spend_limit": {
							Name:  "feegrant-spend-limit",
							Usage: "maximum fee amount the corporation pays for the grantee (e.g. 100stake); no fee allowance when absent",
						},
						"feegrant_spend_limit_period": {
							Name:  "feegrant-spend-limit-period",
							Usage: "reset period for the fee spend limit; requires --expiration, --authz-spend-limit and --authz-spend-limit-period",
						},
					},
				},
				{
					RpcMethod: "RevokeDelegateAuthorization",
					Use:       "revoke-delegate-authz [grantee]",
					Short:     "Revoke delegate authorization for a grantee",
					Long:      "[MOD-DE-MSG-4] Revoke delegate authorization. The signer (--from) is an authorized delegate, or the corporation's policy_address on the group-proposal path. Removes the authorization entry and any associated fee grant for the given corporation/grantee pair.",
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
				{
					RpcMethod: "GrantSponsorship",
					Use:       "grant-sponsorship [sponsored-corporation-id]",
					Short:     "Grant a sponsorship to another corporation",
					Long:      "[MOD-DE-MSG-10] Grant or raise a sponsorship. Not delegable: the signer (--from) is the sponsor's policy_address, so the message runs through a group proposal (--generate-only). Re-grants may only raise the terms.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sponsored_corporation_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"msg_types":       {Name: "msg-types", Usage: "covered paying MsgType values, one per flag; all of them when absent"},
						"spend_limit":     {Name: "spend-limit", Usage: "maximum operation funds paid per cycle (e.g. 1000uvna)"},
						"period":          {Name: "period", Usage: "cycle length for spend-limit"},
						"expiration":      {Name: "expiration", Usage: "end of the sponsorship, or of its first cycle with --period (RFC3339)"},
						"covers_fees":     {Name: "covers-fees", Usage: "the sponsor also pays the network fees"},
						"fee_spend_limit": {Name: "fee-spend-limit", Usage: "maximum network fees per period, or in total; unlimited when absent"},
					},
				},
				{
					RpcMethod: "RevokeSponsorship",
					Use:       "revoke-sponsorship [sponsored-corporation-id]",
					Short:     "Revoke a sponsorship",
					Long:      "[MOD-DE-MSG-11] Revoke a sponsorship and clear the sponsored corporation's election of this sponsor. The signer (--from) is an authorized delegate, or the sponsor's policy_address on the group-proposal path.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sponsored_corporation_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {Name: "corporation", Usage: "policy_address of the sponsor corporation"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
