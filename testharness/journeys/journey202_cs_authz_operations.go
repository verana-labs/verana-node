package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana-node/x/cs/types"

	"github.com/verana-labs/verana-node/testharness/lib"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// RunCredentialSchemaAuthzOperationsJourney implements Journey 202: Test all CS operations with delegate authorization.
// For each of the 3 CS message types: (a) try without auth → fail, (b) grant auth, (c) try with auth → succeed.
// Depends on Journey 201 (setup) having been run first.
func RunCredentialSchemaAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 202: CS Operations with Delegate Authorization (fail-then-pass)")

	// Load results from Journey 201
	setup := lib.LoadJourneyResult("journey201")
	policyAddr := setup.GroupPolicyAddr
	delegateAccount := lib.GetAccount(client, csDelegateName)
	delegateAddr := setup.DelegateAddr
	adminAccount := lib.GetAccount(client, csGroupAdminName)
	member1Account := lib.GetAccount(client, csGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Delegate:     %s\n", delegateAddr)

	// =========================================================================
	// PREREQUISITE: Create Trust Registry with controller = group policy
	// To do this, we first grant TR create auth to the delegate, then create the TR.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE: Create Trust Registry (controller = group policy) ===")

	// Grant TR create authorization to the delegate
	fmt.Println("\n--- Prerequisite 1: Grant delegate auth for CreateEcosystem ---")
	err := lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_CREATE_ECOSYSTEM},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("✅ Prerequisite 1: Granted CreateEcosystem authorization")
	waitForTx("grant TR create auth")

	// Create TR with controller = policyAddr
	fmt.Println("\n--- Prerequisite 2: Create Trust Registry with controller = group policy ---")
	did := lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		did,
		"https://cs-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("✅ Prerequisite 2: Trust Registry created with ID: %d, DID: %s\n", trID, did)
	waitForTx("TR creation")

	// Verify TR creation
	verified := lib.VerifyEcosystem(client, ctx, trID, did)
	if !verified {
		return fmt.Errorf("prerequisite 2 verification failed: trust registry not found or DID mismatch")
	}

	// =========================================================================
	// TEST 1: CreateCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateCredentialSchema ===")

	schemaData := lib.GenerateSimpleSchema(trIDStr)

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Delegate tries CreateCredentialSchema without auth (expect failure) ---")
	_, err = lib.CreateCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CreateCS rejection")

	// 1b: Grant authorization for CreateCredentialSchema
	fmt.Println("\n--- Step 1b: Grant delegate auth for CreateCredentialSchema ---")
	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_CREATE_CREDENTIAL_SCHEMA},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("✅ Step 1b: Granted CreateCredentialSchema authorization")
	waitForTx("grant CreateCS auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Delegate creates credential schema with auth (expect success) ---")
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("✅ Step 1c: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation")

	// Verify CS creation
	verified = lib.VerifyCredentialSchema(client, ctx, csID, trID)
	if !verified {
		return fmt.Errorf("step 1c verification failed: credential schema not found or TR mismatch")
	}

	// =========================================================================
	// TEST 2: UpdateCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 2: UpdateCredentialSchema ===")

	// 2a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 2a: Delegate tries UpdateCredentialSchema without auth (expect failure) ---")
	err = lib.UpdateCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, 365, 365, 180, 180, 90,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	waitForTx("UpdateCS rejection")

	// 2b: Grant authorization for UpdateCredentialSchema
	fmt.Println("\n--- Step 2b: Grant delegate auth for UpdateCredentialSchema ---")
	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA},
	)
	if err != nil {
		return fmt.Errorf("step 2b failed: %w", err)
	}
	fmt.Println("✅ Step 2b: Granted UpdateCredentialSchema authorization")
	waitForTx("grant UpdateCS auth")

	// 2c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 2c: Delegate updates credential schema with auth (expect success) ---")
	err = lib.UpdateCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, 365, 365, 180, 180, 90,
	)
	if err != nil {
		return fmt.Errorf("step 2c failed: %w", err)
	}
	fmt.Println("✅ Step 2c: Successfully updated credential schema")
	waitForTx("UpdateCS success")

	// Verify update
	updatedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 2c verification query failed: %w", err)
	}
	if updatedSchema.Schema.IssuerGrantorValidationValidityPeriod != 365 {
		return fmt.Errorf("step 2c verification failed: IssuerGrantorValidationValidityPeriod should be 365, got %d",
			updatedSchema.Schema.IssuerGrantorValidationValidityPeriod)
	}
	fmt.Printf("✅ Step 2c: Verified IssuerGrantorValidationValidityPeriod = %d days\n",
		updatedSchema.Schema.IssuerGrantorValidationValidityPeriod)

	// =========================================================================
	// TEST 3: ArchiveCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 3: ArchiveCredentialSchema ===")

	// 3a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 3a: Delegate tries ArchiveCredentialSchema without auth (expect failure) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, true,
	)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	waitForTx("ArchiveCS rejection")

	// 3b: Grant authorization for ArchiveCredentialSchema
	fmt.Println("\n--- Step 3b: Grant delegate auth for ArchiveCredentialSchema ---")
	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA},
	)
	if err != nil {
		return fmt.Errorf("step 3b failed: %w", err)
	}
	fmt.Println("✅ Step 3b: Granted ArchiveCredentialSchema authorization")
	waitForTx("grant ArchiveCS auth")

	// 3c: Try WITH authorization — archive (expect success)
	fmt.Println("\n--- Step 3c: Delegate archives credential schema with auth (expect success) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, true,
	)
	if err != nil {
		return fmt.Errorf("step 3c failed: %w", err)
	}
	fmt.Println("✅ Step 3c: Credential schema archived")
	waitForTx("ArchiveCS success")

	// Verify archived state
	archivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 3c verification query failed: %w", err)
	}
	if archivedSchema.Schema.Archived == nil {
		return fmt.Errorf("step 3c verification failed: credential schema should be archived")
	}
	fmt.Println("✅ Step 3c: Verified credential schema is archived")

	// 3d: [MOD-CS-MSG-3-3] spec v4 draft 13: archive is bidirectional; archive=false unarchives.
	fmt.Println("\n--- Step 3d: Delegate unarchives credential schema (expect success) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, false,
	)
	if err != nil {
		return fmt.Errorf("step 3d failed: expected unarchive to succeed per spec v4 draft 13: %w", err)
	}
	unarchivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 3d verification query failed: %w", err)
	}
	if unarchivedSchema.Schema.Archived != nil {
		return fmt.Errorf("step 3d verification failed: credential schema should be unarchived")
	}
	fmt.Println("✅ Step 3d: Verified credential schema is unarchived per [MOD-CS-MSG-3-3]")

	// 3e: [MOD-CS-MSG-3-2-1] unarchiving a non-archived CS must abort.
	fmt.Println("\n--- Step 3e: Unarchive a non-archived CS (expect failure) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, false,
	)
	if err == nil {
		return fmt.Errorf("step 3e failed: expected rejection for unarchiving a non-archived CS")
	}
	fmt.Printf("✅ Step 3e: Correctly rejected unarchive on non-archived CS: %v\n", err)

	// =========================================================================
	// TEST 4: Unauthorized delegate (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Unauthorized delegate (negative test) ===")
	fmt.Println("\n--- Step 4: Unauthorized delegate tries CreateCredentialSchema (expect failure) ---")

	// Use cooluser as an unauthorized delegate (has funds but no CS authorization)
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	_, err = lib.CreateCredentialSchemaWithAuthority(
		client, ctx, cooluser, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
	)
	if err := expectAuthorizationError("Step 4", err); err != nil {
		return err
	}

	// Save results for potential downstream journeys
	result := lib.JourneyResult{
		EcosystemID:     trIDStr,
		SchemaID:        csIDStr,
		DID:             did,
		GroupID:         setup.GroupID,
		GroupPolicyAddr: policyAddr,
		DelegateAddr:    delegateAddr,
	}
	lib.SaveJourneyResult("journey202", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 202 completed successfully! ✨")
	fmt.Println("All 3 CS operations tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
