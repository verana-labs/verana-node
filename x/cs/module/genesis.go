package credentialschema

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/cs/keeper"
	"github.com/verana-labs/verana-node/x/cs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Initialize counter - we'll update with the highest ID after importing
	maxID := uint64(0)

	// Initialize Credential Schemas - sorted by ID for deterministic import
	schemas := genState.CredentialSchemas
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Id < schemas[j].Id
	})

	for _, cs := range schemas {
		// Set credential schema
		if err := k.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
			panic(fmt.Sprintf("failed to set Credential Schema: %s", err))
		}
		// Track highest ID to set counter correctly
		if cs.Id > maxID {
			maxID = cs.Id
		}
	}

	// Restore the exported counter, never below the highest imported id.
	schemaCounter := genState.SchemaCounter
	if maxID > schemaCounter {
		schemaCounter = maxID
	}
	if err := k.Counter.Set(ctx, "cs", schemaCounter); err != nil {
		panic(fmt.Sprintf("failed to set counter: %s", err))
	}

	policies := genState.SchemaAuthorizationPolicies
	sort.Slice(policies, func(i, j int) bool { return policies[i].Id < policies[j].Id })
	var maxPolicyID uint64
	for _, p := range policies {
		if err := k.SchemaAuthorizationPolicies.Set(ctx, p.Id, p); err != nil {
			panic(fmt.Sprintf("failed to set schema authorization policy: %s", err))
		}
		if p.Id > maxPolicyID {
			maxPolicyID = p.Id
		}
	}
	policyCounter := genState.SchemaAuthorizationPolicyCounter
	if maxPolicyID > policyCounter {
		policyCounter = maxPolicyID
	}
	if err := k.Counter.Set(ctx, types.CounterKeySchemaAuthorizationPolicy, policyCounter); err != nil {
		panic(fmt.Sprintf("failed to set schema authorization policy counter: %s", err))
	}

	k.Logger().Info("Initialized Credential Schema module",
		"schemas_count", len(schemas),
		"highest_id", maxID,
		"policies_count", len(policies))
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export all credential schemas in a deterministic order
	var credentialSchemas []types.CredentialSchema
	err := k.CredentialSchema.Walk(ctx, nil, func(key uint64, cs types.CredentialSchema) (bool, error) {
		credentialSchemas = append(credentialSchemas, cs)
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to export Credential Schemas: %s", err))
	}

	// Sort by ID for deterministic export
	sort.Slice(credentialSchemas, func(i, j int) bool {
		return credentialSchemas[i].Id < credentialSchemas[j].Id
	})

	genesis.CredentialSchemas = credentialSchemas

	// Export the counter value - THIS IS THE MISSING PART
	counter, err := k.Counter.Get(ctx, "cs")
	if err != nil {
		// If there's no counter but we have schemas, use the highest ID
		if len(credentialSchemas) > 0 {
			highestID := uint64(0)
			for _, cs := range credentialSchemas {
				if cs.Id > highestID {
					highestID = cs.Id
				}
			}
			counter = highestID
		}
	}
	genesis.SchemaCounter = counter

	// Export schema authorization policies and their counter.
	var policies []types.SchemaAuthorizationPolicy
	if err := k.SchemaAuthorizationPolicies.Walk(ctx, nil, func(_ uint64, p types.SchemaAuthorizationPolicy) (bool, error) {
		policies = append(policies, p)
		return false, nil
	}); err != nil {
		panic(fmt.Sprintf("failed to export schema authorization policies: %s", err))
	}
	sort.Slice(policies, func(i, j int) bool { return policies[i].Id < policies[j].Id })
	genesis.SchemaAuthorizationPolicies = policies

	policyCounter, err := k.Counter.Get(ctx, types.CounterKeySchemaAuthorizationPolicy)
	if err != nil {
		policyCounter = 0
		for _, p := range policies {
			if p.Id > policyCounter {
				policyCounter = p.Id
			}
		}
	}
	genesis.SchemaAuthorizationPolicyCounter = policyCounter

	return genesis
}
