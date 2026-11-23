package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana-node/x/cs/types"
	permtypes "github.com/verana-labs/verana-node/x/pp/types"

	"github.com/verana-labs/verana-node/testharness/lib"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// RunPermissionAuthzOperationsJourney implements Journey 302: Test StartPermissionVP and RenewPermissionVP
// with delegate authorization. For each operation: (a) try without auth -> fail, (b) grant auth, (c) try with auth -> succeed.
// Depends on Journey 301 (setup) having been run first.
func RunPermissionAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 302: Perm Operations with Delegate Authorization (fail-then-pass)")

	// Load results from Journey 301
	setup := lib.LoadJourneyResult("journey301")
	policyAddr := setup.GroupPolicyAddr
	delegateAccount := lib.GetAccount(client, permDelegateName)
	delegateAddr := setup.DelegateAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Delegate:     %s\n", delegateAddr)

	// =========================================================================
	// PREREQUISITES: Create TR, CS, and Root Permission
	// The delegate grants itself a self-delegation, then creates TR, CS, and
	// Root Permission directly (controller = delegateAddr). This avoids complex
	// group proposal chains for prerequisite setup. The actual authz tests
	// below focus on StartPermissionVP and RenewPermissionVP.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create TR, CS, Root Permission ===")

	// --- Prerequisite 1: Grant delegate authorization from the Corporation ---
	// Spec v4 (AUTHZ-CHECK-5): the delegate acts on behalf of the registered
	// Corporation (policyAddr); authz is granted by the corp's group, not via
	// self-delegation (a plain account is not a registered Corporation). Grants
	// are in-place replacements, so each later test grant re-includes this set.
	prereqMsgTypes := []detypes.MsgType{
		detypes.MsgType_MSG_TYPE_CREATE_ECOSYSTEM,
		detypes.MsgType_MSG_TYPE_CREATE_CREDENTIAL_SCHEMA,
		detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING,
		detypes.MsgType_MSG_TYPE_CREATE_ROOT_ACCREDITATION,
		detypes.MsgType_MSG_TYPE_SET_ACCREDITATION_EFFECTIVE_UNTIL,
	}
	fmt.Println("\n--- Prerequisite 1: Grant delegate authz via the Corporation group ---")
	err := lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr, prereqMsgTypes,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Granted delegate authz from Corporation for prereq operations")
	waitForTx("delegate authz grant")

	// --- Prerequisite 2: Create Trust Registry (controller = delegateAddr) ---
	fmt.Println("\n--- Prerequisite 2: Create Trust Registry ---")
	did := lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		did,
		"https://perm-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2: Trust Registry created with ID: %d, DID: %s\n", trID, did)
	waitForTx("TR creation")

	// --- Prerequisite 3: Create Credential Schema with GRANTOR_VALIDATION modes ---
	fmt.Println("\n--- Prerequisite 3: Create Credential Schema ---")
	schemaData := lib.GenerateSimpleSchema(trIDStr)
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 3: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation")

	// --- Prerequisite 4: Create Root Permission (ECOSYSTEM type) ---
	// Delegate is the TR controller, so creator=delegateAddr matches.
	fmt.Println("\n--- Prerequisite 4: Create Root Permission (ECOSYSTEM type) ---")
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(3 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	rootPermID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: %w", err)
	}
	fmt.Printf("OK Prerequisite 4: Root Permission created with ID: %d\n", rootPermID)
	waitForTx("Root perm creation")

	// Root must be effective before it's used as a validator below.
	if err := lib.WaitForPermissionEffective(client, ctx, effectiveFrom, 60); err != nil {
		return fmt.Errorf("prerequisite 4 wait-for-effective failed: %w", err)
	}

	// Verify root permission
	rootPerm, err := lib.GetParticipant(client, ctx, rootPermID)
	if err != nil {
		return fmt.Errorf("prerequisite 4 verification failed: %w", err)
	}
	fmt.Printf("  Root Permission type: %s, schema_id: %d, op_state: %s\n",
		rootPerm.Role.String(), rootPerm.SchemaId, rootPerm.OpState.String())

	// =========================================================================
	// TEST 1: StartPermissionVP
	// =========================================================================
	fmt.Println("\n=== TEST 1: StartPermissionVP ===")

	startPermDID := lib.GenerateUniqueDID(client, ctx)

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Delegate tries StartPermissionVP without auth (expect failure) ---")
	_, err = lib.StartPermissionVPWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		startPermDID,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("StartPermVP rejection")

	// 1b: Grant authorization for StartPermissionVP
	fmt.Println("\n--- Step 1b: Grant delegate auth for StartPermissionVP ---")
	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		append(append([]detypes.MsgType{}, prereqMsgTypes...), detypes.MsgType_MSG_TYPE_START_ONBOARDING),
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted StartPermissionVP authorization")
	waitForTx("grant StartPermVP auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Delegate starts permission VP with auth (expect success) ---")
	permIDStr, err := lib.StartPermissionVPWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		startPermDID,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	permID, _ := strconv.ParseUint(permIDStr, 10, 64)
	fmt.Printf("OK Step 1c: StartPermissionVP succeeded, permission ID: %d\n", permID)
	waitForTx("StartPermVP success")

	// Verify the permission was created in PENDING state
	perm, err := lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 1c verification query failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_PENDING {
		return fmt.Errorf("step 1c verification failed: expected PENDING state, got %s", perm.OpState.String())
	}
	if perm.CorporationId == 0 {
		return fmt.Errorf("step 1c verification failed: expected authority=%s, got %d", policyAddr, perm.CorporationId)
	}
	fmt.Printf("OK Step 1c: Verified permission is PENDING, authority=%d\n", perm.CorporationId)

	// =========================================================================
	// TEST 2: RenewPermissionVP
	// To test renewal, we first need to set the permission to VALIDATED state.
	// The validator (root permission authority = delegateAddr) validates it.
	// =========================================================================
	fmt.Println("\n=== TEST 2: RenewPermissionVP ===")

	// 2-prereq: Validate the permission (set to VALIDATED state)
	fmt.Println("\n--- Step 2-prereq: Validate the permission (set op_state=VALIDATED) ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, delegateAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation:    policyAddr,
		Id:             permID,
		ValidationFees: 0,
		IssuanceFees:   0,
	})
	if err != nil {
		return fmt.Errorf("step 2-prereq failed: %w", err)
	}
	waitForTx("SetPermissionVPToValidated")

	// Verify the permission is now VALIDATED
	perm, err = lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2-prereq verification failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_VALIDATED {
		return fmt.Errorf("step 2-prereq verification failed: expected VALIDATED state, got %s", perm.OpState.String())
	}
	fmt.Printf("OK Step 2-prereq: Permission is now VALIDATED\n")

	// 2a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 2a: Delegate tries RenewPermissionVP without auth (expect failure) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		permID,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	waitForTx("RenewPermVP rejection")

	// 2b: Grant authorization for RenewPermissionVP
	fmt.Println("\n--- Step 2b: Grant delegate auth for RenewPermissionVP ---")
	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, delegateAddr, delegateAddr,
		append(append([]detypes.MsgType{}, prereqMsgTypes...), detypes.MsgType_MSG_TYPE_START_ONBOARDING, detypes.MsgType_MSG_TYPE_RENEW_ONBOARDING),
	)
	if err != nil {
		return fmt.Errorf("step 2b failed: %w", err)
	}
	fmt.Println("OK Step 2b: Granted RenewPermissionVP authorization")
	waitForTx("grant RenewPermVP auth")

	// 2c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 2c: Delegate renews permission VP with auth (expect success) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, delegateAccount, policyAddr,
		permID,
	)
	if err != nil {
		return fmt.Errorf("step 2c failed: %w", err)
	}
	fmt.Println("OK Step 2c: RenewPermissionVP succeeded")
	waitForTx("RenewPermVP success")

	// Verify the permission is back to PENDING state
	perm, err = lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2c verification query failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_PENDING {
		return fmt.Errorf("step 2c verification failed: expected PENDING state after renewal, got %s", perm.OpState.String())
	}
	fmt.Printf("OK Step 2c: Verified permission is PENDING after renewal\n")

	// =========================================================================
	// TEST 3: Unauthorized delegate (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized delegate (negative test) ===")

	// First, validate the permission again so we can test renewal rejection
	_, err = lib.SetPermissionVPToValidated(client, ctx, delegateAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation: policyAddr,
		Id:          permID,
	})
	if err != nil {
		return fmt.Errorf("step 3-prereq failed: %w", err)
	}
	waitForTx("Re-validate for test 3")

	fmt.Println("\n--- Step 3a: Unauthorized delegate tries StartPermissionVP (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	unauthorizedDID := lib.GenerateUniqueDID(client, ctx)
	_, err = lib.StartPermissionVPWithAuthority(
		client, ctx, cooluser, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		unauthorizedDID,
	)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}

	fmt.Println("\n--- Step 3b: Unauthorized delegate tries RenewPermissionVP (expect failure) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, cooluser, policyAddr,
		permID,
	)
	if err := expectAuthorizationError("Step 3b", err); err != nil {
		return err
	}

	// Save results
	result := lib.JourneyResult{
		EcosystemID:     trIDStr,
		SchemaID:        csIDStr,
		DID:             did,
		PermissionID:    permIDStr,
		GroupID:         setup.GroupID,
		GroupPolicyAddr: policyAddr,
		DelegateAddr:    delegateAddr,
	}
	lib.SaveJourneyResult("journey302", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 302 completed successfully!")
	fmt.Println("StartPermissionVP and RenewPermissionVP tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
