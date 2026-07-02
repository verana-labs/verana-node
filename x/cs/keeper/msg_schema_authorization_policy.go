package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-node/x/cs/types"
)

// getSchemaAuthPoliciesForRole returns all policies for (schema_id, role).
func (k Keeper) getSchemaAuthPoliciesForRole(ctx sdk.Context, schemaID uint64, role types.SchemaAuthorizationPolicyRole) ([]types.SchemaAuthorizationPolicy, error) {
	var policies []types.SchemaAuthorizationPolicy
	err := k.SchemaAuthorizationPolicies.Walk(ctx, nil, func(_ uint64, p types.SchemaAuthorizationPolicy) (bool, error) {
		if p.SchemaId == schemaID && p.Role == role {
			policies = append(policies, p)
		}
		return false, nil
	})
	return policies, err
}

// [MOD-CS-MSG-5] CreateSchemaAuthorizationPolicy
func (ms msgServer) CreateSchemaAuthorizationPolicy(goCtx context.Context, msg *types.MsgCreateSchemaAuthorizationPolicy) (*types.MsgCreateSchemaAuthorizationPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgCreateSchemaAuthorizationPolicy", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Load credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check ownership via ecosystem
	if err := ms.checkSchemaOwnership(ctx, cs, msg.Corporation); err != nil {
		return nil, err
	}

	existing, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing policies: %w", err)
	}

	// [MOD-CS-MSG-5-2-1] at most one draft (effective_from == null, not revoked) may exist.
	var draft *types.SchemaAuthorizationPolicy
	maxVersion := uint32(0)
	for i := range existing {
		p := &existing[i]
		if p.Version > maxVersion {
			maxVersion = p.Version
		}
		if p.EffectiveFrom == nil && !p.Revoked {
			if draft != nil {
				return nil, fmt.Errorf("more than one draft policy exists for schema_id %d and role %s", msg.SchemaId, msg.Role)
			}
			draft = p
		}
	}

	// [MOD-CS-MSG-5-3] overwrite the existing draft if there is one, else create a new version.
	if draft != nil {
		draft.Url = msg.Url
		draft.DigestSri = msg.DigestSri
		if err := ms.SchemaAuthorizationPolicies.Set(ctx, draft.Id, *draft); err != nil {
			return nil, fmt.Errorf("failed to store policy: %w", err)
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"create_schema_authorization_policy",
			sdk.NewAttribute("id", strconv.FormatUint(draft.Id, 10)),
			sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
			sdk.NewAttribute("role", msg.Role.String()),
			sdk.NewAttribute("version", strconv.FormatUint(uint64(draft.Version), 10)),
		))
		return &types.MsgCreateSchemaAuthorizationPolicyResponse{Id: draft.Id}, nil
	}

	id, err := ms.GetNextID(ctx, types.CounterKeySchemaAuthorizationPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate policy id: %w", err)
	}

	policy := types.SchemaAuthorizationPolicy{
		Id:        id,
		SchemaId:  msg.SchemaId,
		Role:      msg.Role,
		Url:       msg.Url,
		DigestSri: msg.DigestSri,
		Created:   now,
		Version:   maxVersion + 1,
	}

	if err := ms.SchemaAuthorizationPolicies.Set(ctx, id, policy); err != nil {
		return nil, fmt.Errorf("failed to store policy: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_schema_authorization_policy",
		sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("version", strconv.FormatUint(uint64(policy.Version), 10)),
	))

	return &types.MsgCreateSchemaAuthorizationPolicyResponse{Id: id}, nil
}

// [MOD-CS-MSG-6] IncreaseActiveSchemaAuthorizationPolicyVersion
func (ms msgServer) IncreaseActiveSchemaAuthorizationPolicyVersion(goCtx context.Context, msg *types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion) (*types.MsgIncreaseActiveSchemaAuthorizationPolicyVersionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgIncreaseActiveSchemaAuthorizationPolicyVersion", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Load credential schema and check ownership
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}
	if err := ms.checkSchemaOwnership(ctx, cs, msg.Corporation); err != nil {
		return nil, err
	}

	policies, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	// [MOD-CS-MSG-6-2-1] exactly one draft (effective_from == null, not revoked) must exist.
	var draft, prevActive *types.SchemaAuthorizationPolicy
	for i := range policies {
		p := &policies[i]
		if p.Revoked {
			continue
		}
		if p.EffectiveFrom == nil {
			if draft != nil {
				return nil, fmt.Errorf("more than one draft policy exists for schema_id %d and role %s", msg.SchemaId, msg.Role)
			}
			draft = p
		} else if p.EffectiveUntil == nil || p.EffectiveUntil.After(now) {
			prevActive = p
		}
	}
	if draft == nil {
		return nil, fmt.Errorf("no draft policy exists for schema_id %d and role %s", msg.SchemaId, msg.Role)
	}

	// [MOD-CS-MSG-6-3] activate the draft and deactivate the previously active version.
	draft.EffectiveFrom = &now
	if err := ms.SchemaAuthorizationPolicies.Set(ctx, draft.Id, *draft); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}
	if prevActive != nil {
		prevActive.EffectiveUntil = &now
		if err := ms.SchemaAuthorizationPolicies.Set(ctx, prevActive.Id, *prevActive); err != nil {
			return nil, fmt.Errorf("failed to deactivate previous policy: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"increase_active_schema_authorization_policy_version",
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("new_active_version", strconv.FormatUint(uint64(draft.Version), 10)),
	))

	return &types.MsgIncreaseActiveSchemaAuthorizationPolicyVersionResponse{}, nil
}

// [MOD-CS-MSG-7] RevokeSchemaAuthorizationPolicy
func (ms msgServer) RevokeSchemaAuthorizationPolicy(goCtx context.Context, msg *types.MsgRevokeSchemaAuthorizationPolicy) (*types.MsgRevokeSchemaAuthorizationPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgRevokeSchemaAuthorizationPolicy", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Load credential schema and check ownership
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}
	if err := ms.checkSchemaOwnership(ctx, cs, msg.Corporation); err != nil {
		return nil, err
	}

	// Find the policy for (schema_id, role, version)
	policies, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	var target *types.SchemaAuthorizationPolicy
	for i := range policies {
		if policies[i].Version == msg.Version {
			target = &policies[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no policy found for schema_id %d, role %s, version %d", msg.SchemaId, msg.Role, msg.Version)
	}
	if target.Revoked {
		return nil, fmt.Errorf("policy is already revoked")
	}
	// [MOD-CS-MSG-7-2-1] Policy must be active (effective_from <= now) to be revoked.
	// A null effective_from means the policy has never been activated.
	if target.EffectiveFrom == nil || target.EffectiveFrom.After(now) {
		return nil, fmt.Errorf("policy is not yet active; cannot revoke a future policy")
	}

	target.Revoked = true
	if err := ms.SchemaAuthorizationPolicies.Set(ctx, target.Id, *target); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"revoke_schema_authorization_policy",
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("version", strconv.FormatUint(uint64(msg.Version), 10)),
	))

	return &types.MsgRevokeSchemaAuthorizationPolicyResponse{}, nil
}
