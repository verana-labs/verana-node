package credentialschema_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/testutil/nullify"
	credentialschema "github.com/verana-labs/verana-node/x/cs/module"
	"github.com/verana-labs/verana-node/x/cs/types"
)

func TestGenesisImportExport(t *testing.T) {
	// Create a valid JSON schema for testing
	validJsonSchema := `{
  "$id": "vpr:verana:mainnet:cs:1",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ExampleCredential",
  "description": "ExampleCredential using JsonSchema",
  "type": "object",
  "properties": {
    "credentialSubject": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "format": "uri"
        },
        "firstName": {
          "type": "string",
          "minLength": 0,
          "maxLength": 256
        },
        "lastName": {
          "type": "string",
          "minLength": 1,
          "maxLength": 256
        },
        "expirationDate": {
          "type": "string",
          "format": "date"
        },
        "countryOfResidence": {
          "type": "string",
          "minLength": 2,
          "maxLength": 2
        }
      },
      "required": [
        "id",
        "lastName",
        "birthDate",
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`

	// Create test schemas with valid values
	now := time.Now().UTC()
	schemaA := types.CredentialSchema{
		Id:                                      1,
		EcosystemId:                             100,
		Created:                                 now.Add(-24 * time.Hour),
		Modified:                                now.Add(-12 * time.Hour),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerOnboardingMode:                    types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		VerifierOnboardingMode:                  types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		HolderOnboardingMode:                    types.HolderOnboardingMode_HOLDER_ONBOARDING_MODE_UNSPECIFIED,
		PricingAssetType:                        types.PricingAssetType_TU,
		PricingAsset:                            "tu",
		DigestAlgorithm:                         "sha256",
	}

	schemaB := types.CredentialSchema{
		Id:                                      2,
		EcosystemId:                             101,
		Created:                                 now.Add(-10 * time.Hour),
		Modified:                                now.Add(-5 * time.Hour),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   180,
		VerifierGrantorValidationValidityPeriod: 180,
		IssuerValidationValidityPeriod:          90,
		VerifierValidationValidityPeriod:        90,
		HolderValidationValidityPeriod:          90,
		IssuerOnboardingMode:                    types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		VerifierOnboardingMode:                  types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
		HolderOnboardingMode:                    types.HolderOnboardingMode_HOLDER_ONBOARDING_MODE_UNSPECIFIED,
		PricingAssetType:                        types.PricingAssetType_COIN,
		PricingAsset:                            "uvna",
		DigestAlgorithm:                         "sha384",
	}

	policyA := types.SchemaAuthorizationPolicy{Id: 1, SchemaId: 1, Created: now.Add(-8 * time.Hour), Version: 1}
	policyB := types.SchemaAuthorizationPolicy{Id: 2, SchemaId: 2, Created: now.Add(-6 * time.Hour), Version: 1}

	// Create a test genesis state with multiple schemas
	genesisState := types.GenesisState{
		Params:                      types.DefaultParams(),
		CredentialSchemas:           []types.CredentialSchema{schemaB, schemaA},          // Deliberately out of order
		SchemaAuthorizationPolicies: []types.SchemaAuthorizationPolicy{policyB, policyA}, // Deliberately out of order
	}

	// Setup the module
	k, _, ctx := keepertest.CredentialschemaKeeper(t)

	// Test import
	credentialschema.InitGenesis(ctx, k, genesisState)

	// Verify schemas were imported correctly
	schemaFromState1, err := k.CredentialSchema.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, schemaA.Id, schemaFromState1.Id)
	require.Equal(t, schemaA.EcosystemId, schemaFromState1.EcosystemId)
	require.Equal(t, schemaA.IssuerOnboardingMode, schemaFromState1.IssuerOnboardingMode)

	schemaFromState2, err := k.CredentialSchema.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, schemaB.Id, schemaFromState2.Id)
	require.Equal(t, schemaB.EcosystemId, schemaFromState2.EcosystemId)
	require.Equal(t, schemaB.IssuerOnboardingMode, schemaFromState2.IssuerOnboardingMode)

	// Verify counter was set correctly (to highest ID)
	counterId, err := k.Counter.Get(ctx, "cs")
	require.NoError(t, err)
	require.Equal(t, uint64(2), counterId)

	// Test export
	exportedGenesis := credentialschema.ExportGenesis(ctx, k)
	require.NotNil(t, exportedGenesis)

	// Verify schemas are exported in deterministic order (by ID)
	require.Len(t, exportedGenesis.CredentialSchemas, 2)
	require.Equal(t, uint64(1), exportedGenesis.CredentialSchemas[0].Id)
	require.Equal(t, uint64(2), exportedGenesis.CredentialSchemas[1].Id)

	// Schema authorization policies and their counter round-trip (regression: were dropped).
	require.Len(t, exportedGenesis.SchemaAuthorizationPolicies, 2)
	require.Equal(t, uint64(1), exportedGenesis.SchemaAuthorizationPolicies[0].Id)
	require.Equal(t, uint64(2), exportedGenesis.SchemaAuthorizationPolicies[1].Id)
	policyCounter, err := k.Counter.Get(ctx, types.CounterKeySchemaAuthorizationPolicy)
	require.NoError(t, err)
	require.Equal(t, uint64(2), policyCounter)

	// Verify all parameters match
	require.Equal(t, genesisState.Params, exportedGenesis.Params)

	// Use nullify to ignore irrelevant fields and verify the exported data matches the imported data
	nullify.Fill(&genesisState)
	nullify.Fill(exportedGenesis)

	require.Equal(t, genesisState.Params, exportedGenesis.Params)
	require.ElementsMatch(t, genesisState.CredentialSchemas, exportedGenesis.CredentialSchemas)
	require.ElementsMatch(t, genesisState.SchemaAuthorizationPolicies, exportedGenesis.SchemaAuthorizationPolicies)
}

func TestGenesisValidation(t *testing.T) {
	// Create a valid JSON schema for testing
	validJsonSchema := `{
  "$id": "vpr:verana:VPR_CHAIN_ID:cs:VPR_CREDENTIAL_SCHEMA_ID",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ExampleCredential",
  "description": "ExampleCredential using JsonSchema",
  "type": "object",
  "properties": {
    "credentialSubject": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "format": "uri"
        },
        "firstName": {
          "type": "string",
          "minLength": 0,
          "maxLength": 256
        },
        "lastName": {
          "type": "string",
          "minLength": 1,
          "maxLength": 256
        },
        "expirationDate": {
          "type": "string",
          "format": "date"
        },
        "countryOfResidence": {
          "type": "string",
          "minLength": 2,
          "maxLength": 2
        }
      },
      "required": [
        "id",
        "lastName",
        "birthDate",
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`

	now := time.Now().UTC()
	validSchema := types.CredentialSchema{
		Id:                                      1,
		EcosystemId:                             100,
		Created:                                 now.Add(-24 * time.Hour),
		Modified:                                now.Add(-12 * time.Hour),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerOnboardingMode:                    types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		VerifierOnboardingMode:                  types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		HolderOnboardingMode:                    types.HolderOnboardingMode_HOLDER_ONBOARDING_MODE_UNSPECIFIED,
		PricingAssetType:                        types.PricingAssetType_TU,
		PricingAsset:                            "tu",
		DigestAlgorithm:                         "sha256",
	}

	tests := []struct {
		name         string
		genesisState types.GenesisState
		valid        bool
		errorMsg     string
	}{
		{
			name:         "default is valid",
			genesisState: *types.DefaultGenesis(),
			valid:        true,
		},
		{
			name: "valid genesis state with schemas",
			genesisState: types.GenesisState{
				Params:            types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{validSchema},
				SchemaCounter:     1,
			},
			valid: true,
		},
		{
			name: "schema_counter behind highest schema id",
			genesisState: types.GenesisState{
				Params:            types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{validSchema},
				SchemaCounter:     0,
			},
			valid:    false,
			errorMsg: "schema_counter",
		},
		{
			name: "duplicate schema authorization policy ID",
			genesisState: types.GenesisState{
				Params:        types.DefaultParams(),
				SchemaCounter: 0,
				SchemaAuthorizationPolicies: []types.SchemaAuthorizationPolicy{
					{Id: 1, SchemaId: 1},
					{Id: 1, SchemaId: 2},
				},
				SchemaAuthorizationPolicyCounter: 1,
			},
			valid:    false,
			errorMsg: "duplicate schema authorization policy ID",
		},
		{
			name: "policy counter behind highest policy id",
			genesisState: types.GenesisState{
				Params:                           types.DefaultParams(),
				SchemaAuthorizationPolicies:      []types.SchemaAuthorizationPolicy{{Id: 5, SchemaId: 1}},
				SchemaAuthorizationPolicyCounter: 1,
			},
			valid:    false,
			errorMsg: "schema_authorization_policy_counter",
		},
		{
			name: "duplicate schema ID",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					validSchema,
					{
						Id:                     1, // Same ID
						EcosystemId:            101,
						Created:                now.Add(-10 * time.Hour),
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "duplicate credential schema ID",
		},
		{
			name: "invalid schema ID (zero)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     0, // Invalid ID
						EcosystemId:            101,
						Created:                now.Add(-10 * time.Hour),
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "invalid ID 0",
		},
		{
			name: "invalid trust registry ID (zero)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     3,
						EcosystemId:            0, // Invalid TR ID
						Created:                now.Add(-10 * time.Hour),
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "invalid ecosystem_id 0",
		},
		{
			name: "invalid empty JSON schema",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     3,
						EcosystemId:            101,
						Created:                now.Add(-10 * time.Hour),
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             "", // Empty schema
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "empty JSON schema",
		},
		{
			name: "invalid perm management mode",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     3,
						EcosystemId:            101,
						Created:                now.Add(-10 * time.Hour),
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_UNSPECIFIED, // Invalid mode
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "invalid issuer onboarding mode",
		},
		{
			name: "zero creation time",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     3,
						EcosystemId:            101,
						Created:                time.Time{}, // Zero time
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "invalid creation time",
		},
		{
			name: "creation time after modified time (inconsistent timestamps)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                     3,
						EcosystemId:            101,
						Created:                now, // More recent than modified
						Modified:               now.Add(-5 * time.Hour),
						JsonSchema:             validJsonSchema,
						IssuerOnboardingMode:   types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode: types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:       types.PricingAssetType_TU,
						PricingAsset:           "tu",
						DigestAlgorithm:        "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "creation time after modified time",
		},
		{
			name: "exceeding max validity period days",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                                      3,
						EcosystemId:                             101,
						Created:                                 now.Add(-10 * time.Hour),
						Modified:                                now.Add(-5 * time.Hour),
						JsonSchema:                              validJsonSchema,
						IssuerGrantorValidationValidityPeriod:   9999, // Exceeds max
						VerifierGrantorValidationValidityPeriod: 365,
						IssuerValidationValidityPeriod:          180,
						VerifierValidationValidityPeriod:        180,
						HolderValidationValidityPeriod:          180,
						IssuerOnboardingMode:                    types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode:                  types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
						PricingAssetType:                        types.PricingAssetType_TU,
						PricingAsset:                            "tu",
						DigestAlgorithm:                         "sha256",
					},
				},
			},
			valid:    false,
			errorMsg: "issuer grantor validation validity period exceeding maximum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesisState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg)
				}
			}
		})
	}
}
