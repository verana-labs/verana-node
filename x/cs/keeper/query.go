package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-node/x/cs/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListCredentialSchemas(goCtx context.Context, req *types.QueryListCredentialSchemasRequest) (*types.QueryListCredentialSchemasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, fmt.Errorf("response_max_size must be between 1 and 1024")
	}

	var schemas []types.CredentialSchema
	err := k.CredentialSchema.Walk(ctx, nil, func(key uint64, schema types.CredentialSchema) (bool, error) {
		// Filter by ecosystem if specified
		if req.EcosystemId != 0 && schema.EcosystemId != req.EcosystemId {
			return false, nil
		}

		// Filter by modification time if specified
		if req.ModifiedAfter != nil && !schema.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		// Filter archived entries if only_active is set
		if req.OnlyActive && schema.Archived != nil {
			return false, nil
		}

		// Filter by issuer_onboarding_mode if specified
		if req.IssuerOnboardingMode != types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_UNSPECIFIED &&
			schema.IssuerOnboardingMode != req.IssuerOnboardingMode {
			return false, nil
		}

		// Filter by verifier_onboarding_mode if specified
		if req.VerifierOnboardingMode != types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_UNSPECIFIED &&
			schema.VerifierOnboardingMode != req.VerifierOnboardingMode {
			return false, nil
		}

		// Filter by holder_onboarding_mode if specified
		if req.HolderOnboardingMode != types.HolderOnboardingMode_HOLDER_ONBOARDING_MODE_UNSPECIFIED &&
			schema.HolderOnboardingMode != req.HolderOnboardingMode {
			return false, nil
		}

		// Ensure canonical $id is present in the JSON schema
		schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
		if err != nil {
			return true, status.Error(codes.Internal, fmt.Sprintf("failed to ensure canonical ID: %v", err))
		}
		schema.JsonSchema = schemaWithCanonicalID

		schemas = append(schemas, schema)
		return false, nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modified DESC, id ASC on ties (spec: ordered by modified DESC; stable for same-block ties).
	sort.SliceStable(schemas, func(i, j int) bool {
		if schemas[i].Modified.Equal(schemas[j].Modified) {
			return schemas[i].Id < schemas[j].Id
		}
		return schemas[i].Modified.After(schemas[j].Modified)
	})

	// Apply response_max_size limit after sorting
	if len(schemas) > int(req.ResponseMaxSize) {
		schemas = schemas[:req.ResponseMaxSize]
	}

	return &types.QueryListCredentialSchemasResponse{
		Schemas: schemas,
	}, nil
}

func (k Keeper) GetCredentialSchema(goCtx context.Context, req *types.QueryGetCredentialSchemaRequest) (*types.QueryGetCredentialSchemaResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	schema, err := k.CredentialSchema.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "credential schema not found")
	}

	// Ensure canonical $id is present in the JSON schema
	schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to ensure canonical ID: %v", err))
	}
	schema.JsonSchema = schemaWithCanonicalID

	return &types.QueryGetCredentialSchemaResponse{
		Schema: schema,
	}, nil
}

func (k Keeper) RenderJsonSchema(goCtx context.Context, req *types.QueryRenderJsonSchemaRequest) (*types.QueryRenderJsonSchemaResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	schema, err := k.CredentialSchema.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "credential schema not found")
	}

	// Ensure canonical $id is present in the JSON schema
	schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to ensure canonical ID: %v", err))
	}

	// Apply full JCS canonicalization (RFC 8785): sorted keys, no insignificant whitespace
	canonicalized, err := types.CanonicalizeJCS(schemaWithCanonicalID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to JCS-canonicalize schema: %v", err))
	}

	return &types.QueryRenderJsonSchemaResponse{
		Schema: canonicalized,
	}, nil
}

func (k Keeper) GetSchemaAuthorizationPolicy(goCtx context.Context, req *types.QueryGetSchemaAuthorizationPolicyRequest) (*types.QueryGetSchemaAuthorizationPolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be provided")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	policy, err := k.SchemaAuthorizationPolicies.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "schema authorization policy not found")
	}

	return &types.QueryGetSchemaAuthorizationPolicyResponse{AuthorizationPolicy: policy}, nil
}

func (k Keeper) ListSchemaAuthorizationPolicies(goCtx context.Context, req *types.QueryListSchemaAuthorizationPoliciesRequest) (*types.QueryListSchemaAuthorizationPoliciesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.SchemaId == 0 {
		return nil, status.Error(codes.InvalidArgument, "schema_id must be provided")
	}
	if req.Role != types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER &&
		req.Role != types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_VERIFIER {
		return nil, status.Error(codes.InvalidArgument, "role must be ISSUER or VERIFIER")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if _, err := k.CredentialSchema.Get(ctx, req.SchemaId); err != nil {
		return nil, status.Error(codes.NotFound, "credential schema not found")
	}

	policies, err := k.getSchemaAuthPoliciesForRole(ctx, req.SchemaId, req.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	sort.Slice(policies, func(i, j int) bool { return policies[i].Version < policies[j].Version })

	return &types.QueryListSchemaAuthorizationPoliciesResponse{Policies: policies}, nil
}
