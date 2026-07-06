package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	credentialschematypes "github.com/verana-labs/verana-node/x/cs/types"

	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// maxInt64AsUint64 is the highest uint64 that still fits in a signed int64.
// Used to guard narrowing conversions of fee/deposit amounts before they
// reach bank/sdk helpers that take int64.
const maxInt64AsUint64 uint64 = 1<<63 - 1

// uint64ToInt64 narrows a uint64 to int64 with an overflow guard. Returns an
// error when x does not fit, so the caller can abort the transaction rather
// than silently wrap to a negative amount.
func uint64ToInt64(x uint64, field string) (int64, error) {
	if x > maxInt64AsUint64 {
		return 0, fmt.Errorf("%s overflows int64: %d", field, x)
	}
	return int64(x), nil
}

// decToUint64 truncates a LegacyDec to uint64 with an overflow guard.
func decToUint64(d math.LegacyDec, field string) (uint64, error) {
	i := d.TruncateInt()
	if !i.IsUint64() {
		return 0, fmt.Errorf("%s overflows uint64: %s", field, i.String())
	}
	return i.Uint64(), nil
}

// [MOD-PP-MSG-10-2] Create or Update Participant Session precondition checks
func (ms msgServer) validateCreateOrUpdateParticipantSessionPreconditions(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, now time.Time) error {
	// if issuer_participant_id is null AND verifier_participant_id is null, MUST abort
	if msg.IssuerParticipantId == 0 && msg.VerifierParticipantId == 0 {
		return fmt.Errorf("at least one of issuer_participant_id or verifier_participant_id must be provided")
	}

	// id MUST be a valid uuid (already validated in ValidateBasic)
	// If an entry with id already exists, existing_entry.authority MUST equal authority AND existing_entry.vs_operator MUST equal operator
	if err := ms.validateSessionAccess(ctx, msg); err != nil {
		return err
	}

	var issuerParticipant, verifierParticipant types.Participant
	var hasIssuer, hasVerifier bool

	// if issuer_participant_id is not null
	if msg.IssuerParticipantId != 0 {
		var err error
		issuerParticipant, err = ms.Participant.Get(ctx, msg.IssuerParticipantId)
		if err != nil {
			return fmt.Errorf("issuer participant not found: %w", err)
		}
		hasIssuer = true

		// if issuer_participant.type is not ISSUER, abort
		if issuerParticipant.Role != types.ParticipantRole_ISSUER {
			return fmt.Errorf("issuer participant must be ISSUER type")
		}

		// if issuer_participant is not an active participant, abort
		if err := IsValidParticipant(issuerParticipant, now); err != nil {
			return fmt.Errorf("issuer participant is not valid: %w", err)
		}

		// if issuer_participant.vs_operator is not equal to operator, abort
		if issuerParticipant.VsOperator != msg.Operator {
			return fmt.Errorf("issuer participant vs_operator does not match operator")
		}

		// if issuer_participant.authority is not equal to authority, abort
		issuerCorpAcct, err := ms.corpAccountFromID(ctx, issuerParticipant.CorporationId)
		if err != nil {
			return err
		}
		if issuerCorpAcct != msg.Corporation {
			return fmt.Errorf("issuer participant authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// if verifier_participant_id is not null
	if msg.VerifierParticipantId != 0 {
		var err error
		verifierParticipant, err = ms.Participant.Get(ctx, msg.VerifierParticipantId)
		if err != nil {
			return fmt.Errorf("verifier participant not found: %w", err)
		}
		hasVerifier = true

		// if verifier_participant.type is not VERIFIER, abort
		if verifierParticipant.Role != types.ParticipantRole_VERIFIER {
			return fmt.Errorf("verifier participant must be VERIFIER type")
		}

		// if verifier_participant is not an active participant, abort
		if err := IsValidParticipant(verifierParticipant, now); err != nil {
			return fmt.Errorf("verifier participant is not valid: %w", err)
		}

		// if verifier_participant.vs_operator is not equal to operator, abort
		if verifierParticipant.VsOperator != msg.Operator {
			return fmt.Errorf("verifier participant vs_operator does not match operator")
		}

		// if verifier_participant.authority is not equal to authority, abort
		verifierCorpAcct, err := ms.corpAccountFromID(ctx, verifierParticipant.CorporationId)
		if err != nil {
			return err
		}
		if verifierCorpAcct != msg.Corporation {
			return fmt.Errorf("verifier participant authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// Define the primary participant: if verifier_participant is not null, participant = verifier_participant, else participant = issuer_participant
	var primaryParticipant types.Participant
	if hasVerifier {
		primaryParticipant = verifierParticipant
	} else if hasIssuer {
		primaryParticipant = issuerParticipant
	}

	// [AUTHZ-CHECK-3] MUST pass for the primary participant. Resolve co.id once and
	// run the record-based check; the record's existence + msg_type membership now
	// encodes whether the VS operator is authorized.
	if ms.delegationKeeper == nil {
		return fmt.Errorf("delegation keeper is required for VS operator authorization")
	}
	primaryCorpID, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	if err := ms.delegationKeeper.CheckVSOperatorAuthorizationOnParticipant(
		ctx,
		primaryCorpID,
		msg.Operator,
		primaryParticipant.Id,
		types.MsgCreateOrUpdateParticipantSessionTypeURL,
	); err != nil {
		return fmt.Errorf("VS operator authorization check failed: %w", err)
	}

	// agent_participant_id and wallet_agent_participant_id are optional; validate each only when set.

	// agent: Load agent_participant from agent_participant_id (if set)
	if msg.AgentParticipantId != 0 {
		agentParticipant, err := ms.Participant.Get(ctx, msg.AgentParticipantId)
		if err != nil {
			return fmt.Errorf("agent participant not found: %w", err)
		}

		// if agent_participant.type is not ISSUER, abort
		if agentParticipant.Role != types.ParticipantRole_ISSUER {
			return fmt.Errorf("agent participant must be ISSUER type")
		}

		// if agent_participant is not an active participant, abort
		if err := IsValidParticipant(agentParticipant, now); err != nil {
			return fmt.Errorf("agent participant is not valid: %w", err)
		}
	}

	// wallet_agent: Load wallet_agent_participant from wallet_agent_participant_id (if set)
	if msg.WalletAgentParticipantId != 0 {
		walletAgentParticipant, err := ms.Participant.Get(ctx, msg.WalletAgentParticipantId)
		if err != nil {
			return fmt.Errorf("wallet agent participant not found: %w", err)
		}

		// if wallet_agent_participant.type is not ISSUER, abort
		if walletAgentParticipant.Role != types.ParticipantRole_ISSUER {
			return fmt.Errorf("wallet agent participant must be ISSUER type")
		}

		// if wallet_agent_participant is not an active participant, abort
		if err := IsValidParticipant(walletAgentParticipant, now); err != nil {
			return fmt.Errorf("wallet agent participant is not valid: %w", err)
		}
	}

	return nil
}

// beneficiaryFee is the resolved on-chain settlement for one beneficiary,
// computed once so the fee check and the execution stay consistent.
type beneficiaryFee struct {
	participant        types.Participant
	feeDenom           string // denom of payeeFeesToAccount (native for A/B, pricing asset for C)
	payeeFeesToAccount uint64 // in feeDenom; native×(1-rate) for A/B, asset×(1-rate) for C, 0 for D
	payerTrustDeposit  uint64 // native; staked for the payer
	payeeTrustDeposit  uint64 // native; staked on behalf of the payee (equals payerTrustDeposit)
}

// sessionFeePlan is the full fee/deposit settlement for a session operation.
// required is the total balance the payer must hold across every denom; the
// native component is also what the spend limit is debited by.
type sessionFeePlan struct {
	beneficiaries     []beneficiaryFee
	userAgentReward   math.LegacyDec // accumulated native user-agent reward
	walletAgentReward math.LegacyDec // accumulated native wallet-agent reward
	required          sdk.Coins
}

// schemaForSession loads the credential schema backing the session operation,
// from the issuer participant if present, else the verifier participant.
func (ms msgServer) schemaForSession(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession) (credentialschematypes.CredentialSchema, error) {
	var schemaID uint64
	if msg.IssuerParticipantId != 0 {
		p, err := ms.Participant.Get(ctx, msg.IssuerParticipantId)
		if err != nil {
			return credentialschematypes.CredentialSchema{}, fmt.Errorf("issuer participant not found: %w", err)
		}
		schemaID = p.SchemaId
	} else if msg.VerifierParticipantId != 0 {
		p, err := ms.Participant.Get(ctx, msg.VerifierParticipantId)
		if err != nil {
			return credentialschematypes.CredentialSchema{}, fmt.Errorf("verifier participant not found: %w", err)
		}
		schemaID = p.SchemaId
	} else {
		return credentialschematypes.CredentialSchema{}, fmt.Errorf("at least one of issuer_participant_id or verifier_participant_id must be provided")
	}
	return ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, schemaID)
}

// [MOD-PP-MSG-10-3 / 10-4] buildSessionFeePlan resolves, per beneficiary, the
// fee distribution under the credential schema pricing asset (4-case model).
// Trust deposits and agent rewards are always native; payee fees follow the
// settlement denom. The returned plan drives both the balance precheck and the
// execution transfers, so the check always matches what execution spends.
func (ms msgServer) buildSessionFeePlan(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, foundParticipantSet []types.Participant) (*sessionFeePlan, error) {
	cs, err := ms.schemaForSession(ctx, msg)
	if err != nil {
		return nil, err
	}

	isVerification := msg.VerifierParticipantId != 0

	// Executor's discount, applied to every beneficiary fee (in pricing asset).
	var executorDiscount uint64
	if isVerification {
		executor, err := ms.Participant.Get(ctx, msg.VerifierParticipantId)
		if err != nil {
			return nil, fmt.Errorf("failed to get verifier participant: %w", err)
		}
		executorDiscount = executor.VerificationFeeDiscount
	} else {
		executor, err := ms.Participant.Get(ctx, msg.IssuerParticipantId)
		if err != nil {
			return nil, fmt.Errorf("failed to get issuer participant: %w", err)
		}
		executorDiscount = executor.IssuanceFeeDiscount
	}

	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)

	plan := &sessionFeePlan{
		userAgentReward:   math.LegacyZeroDec(),
		walletAgentReward: math.LegacyZeroDec(),
	}

	const discountScale = 10000 // 10000 = 1.0 = 100% discount

	for _, participant := range foundParticipantSet {
		var fee uint64
		if isVerification {
			fee = participant.VerificationFees
		} else {
			fee = participant.IssuanceFees
		}
		// beneficiary_fee_in_pricing_asset = participant.fee × (1 - discount).
		// Computed via math.Int to avoid uint64 overflow on large fees.
		if executorDiscount > 0 {
			fee = math.NewIntFromUint64(fee).MulRaw(int64(discountScale - executorDiscount)).QuoRaw(discountScale).Uint64()
		}
		if fee == 0 {
			continue
		}

		feeInDenom, feeDenom, nativeBasis, err := ms.resolvePricing(ctx, cs, fee)
		if err != nil {
			return nil, err
		}
		nativeBasisDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(nativeBasis))

		// Trust deposits are always native: rate applied to the native basis.
		payerTrustDeposit, err := decToUint64(nativeBasisDec.Mul(trustDepositRate), "payer_trust_deposit")
		if err != nil {
			return nil, err
		}
		// Payee fees settle in the fee denom: feeInDenom × (1 - trust_deposit_rate).
		payeeFeesToAccount, err := decToUint64(
			math.LegacyNewDecFromInt(math.NewIntFromUint64(feeInDenom)).Mul(math.LegacyOneDec().Sub(trustDepositRate)),
			"payee_fees_to_account",
		)
		if err != nil {
			return nil, err
		}

		plan.beneficiaries = append(plan.beneficiaries, beneficiaryFee{
			participant:        participant,
			feeDenom:           feeDenom,
			payeeFeesToAccount: payeeFeesToAccount,
			payerTrustDeposit:  payerTrustDeposit,
			payeeTrustDeposit:  payerTrustDeposit,
		})

		// Balance obligations: payer + payee trust deposits in native, payee fees in feeDenom.
		plan.required = plan.required.Add(sdk.NewCoin(types.BondDenom, math.NewIntFromUint64(payerTrustDeposit).MulRaw(2)))
		if payeeFeesToAccount > 0 {
			plan.required = plan.required.Add(sdk.NewCoin(feeDenom, math.NewIntFromUint64(payeeFeesToAccount)))
		}

		if msg.AgentParticipantId != 0 {
			plan.userAgentReward = plan.userAgentReward.Add(nativeBasisDec.Mul(userAgentRewardRate))
		}
		if msg.WalletAgentParticipantId != 0 {
			plan.walletAgentReward = plan.walletAgentReward.Add(nativeBasisDec.Mul(walletUserAgentRewardRate))
		}
	}

	// Agent rewards are paid in native; add their truncated totals to the obligation.
	if plan.userAgentReward.IsPositive() {
		plan.required = plan.required.Add(sdk.NewCoin(types.BondDenom, plan.userAgentReward.TruncateInt()))
	}
	if plan.walletAgentReward.IsPositive() {
		plan.required = plan.required.Add(sdk.NewCoin(types.BondDenom, plan.walletAgentReward.TruncateInt()))
	}

	return plan, nil
}

// [MOD-PP-MSG-10-3] Create or Update Participant Session fee checks
func (ms msgServer) validateCreateOrUpdateParticipantSessionFees(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession) (*sessionFeePlan, error) {
	// use "Find Beneficiaries" query method to get the set of beneficiary participant found_participant_set
	foundParticipantSet, err := ms.findBeneficiaries(ctx, msg.IssuerParticipantId, msg.VerifierParticipantId)
	if err != nil {
		return nil, fmt.Errorf("failed to find beneficiaries: %w", err)
	}

	plan, err := ms.buildSessionFeePlan(ctx, msg, foundParticipantSet)
	if err != nil {
		return nil, err
	}

	// corporation (payer) account MUST have sufficient available balance in every denom.
	authorityAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}
	for _, coin := range plan.required {
		if !ms.bankKeeper.HasBalance(ctx, authorityAddr, coin) {
			return nil, fmt.Errorf("insufficient funds: required %s", coin)
		}
	}

	return plan, nil
}

// [MOD-PP-MSG-10-4] Create or Update Participant Session execution
func (ms msgServer) executeCreateOrUpdateParticipantSession(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, plan *sessionFeePlan, now time.Time) error {
	isVerification := msg.VerifierParticipantId != 0
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)

	authorityAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Payer participant (issuer or verifier) accumulates its own trust deposit.
	var payerParticipant types.Participant
	if isVerification {
		payerParticipant, err = ms.Participant.Get(ctx, msg.VerifierParticipantId)
	} else {
		payerParticipant, err = ms.Participant.Get(ctx, msg.IssuerParticipantId)
	}
	if err != nil {
		return fmt.Errorf("failed to get payer participant: %w", err)
	}

	for _, bf := range plan.beneficiaries {
		participant := bf.participant
		participantCorpAcct, err := ms.corpAccountFromID(ctx, participant.CorporationId)
		if err != nil {
			return err
		}

		// Transfer payee fees to the beneficiary account in the settlement denom.
		if bf.payeeFeesToAccount > 0 {
			granteeAddr, err := sdk.AccAddressFromBech32(participantCorpAcct)
			if err != nil {
				return fmt.Errorf("invalid grantee address: %w", err)
			}
			amt, err := uint64ToInt64(bf.payeeFeesToAccount, "payee_fees_to_account")
			if err != nil {
				return err
			}
			if err := ms.bankKeeper.SendCoins(ctx, authorityAddr, granteeAddr, sdk.NewCoins(sdk.NewInt64Coin(bf.feeDenom, amt))); err != nil {
				return fmt.Errorf("failed to transfer beneficiary fees: %w", err)
			}
		}

		// Increase beneficiary trust deposit (native), funded by the payer.
		if bf.payeeTrustDeposit > 0 {
			td, err := uint64ToInt64(bf.payeeTrustDeposit, "payee_trust_deposit")
			if err != nil {
				return err
			}
			if err := ms.trustDeposit.AdjustTrustDepositOnBehalf(ctx, participantCorpAcct, authorityAddr, td); err != nil {
				return fmt.Errorf("failed to adjust grantee trust deposit: %w", err)
			}
			participant.Deposit += bf.payeeTrustDeposit
			if err := ms.Keeper.UpdateParticipant(ctx, participant); err != nil {
				return fmt.Errorf("failed to update grantee participant deposit: %w", err)
			}
		}

		// Increase payer's own trust deposit (native).
		if bf.payerTrustDeposit > 0 {
			td, err := uint64ToInt64(bf.payerTrustDeposit, "payer_trust_deposit")
			if err != nil {
				return err
			}
			if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Corporation, td, "csps_payer_trust_deposit"); err != nil {
				return fmt.Errorf("failed to adjust payer trust deposit: %w", err)
			}
			payerParticipant.Deposit += bf.payerTrustDeposit
			if err := ms.Keeper.UpdateParticipant(ctx, payerParticipant); err != nil {
				return fmt.Errorf("failed to update payer participant deposit: %w", err)
			}
		}
	}

	// Agent rewards are always paid in the native denom.
	if msg.AgentParticipantId != 0 && plan.userAgentReward.IsPositive() {
		if err := ms.distributeAgentReward(ctx, authorityAddr, msg.AgentParticipantId, plan.userAgentReward, trustDepositRate, "user_agent"); err != nil {
			return err
		}
	}
	if msg.WalletAgentParticipantId != 0 && plan.walletAgentReward.IsPositive() {
		if err := ms.distributeAgentReward(ctx, authorityAddr, msg.WalletAgentParticipantId, plan.walletAgentReward, trustDepositRate, "wallet_agent"); err != nil {
			return err
		}
	}

	if err := ms.createOrUpdateSession(ctx, msg, now); err != nil {
		return fmt.Errorf("failed to create/update session: %w", err)
	}

	// [MOD-PP-MSG-10] If this transaction is for issuance of a credential, persist
	// the digest SRI by calling [MOD-DI-MSG-1] keeper-to-keeper. Spec explicitly
	// lets participant invoke DI with no signer/AUTHZ check. Scoped to the issuance
	// path and only when the caller supplied a non-empty digest.
	if msg.Digest != "" && msg.IssuerParticipantId != 0 {
		if ms.digestKeeper == nil {
			return fmt.Errorf("digest keeper is required but not set")
		}
		if err := ms.digestKeeper.StoreDigestModuleCall(ctx, msg.Corporation, msg.Digest); err != nil {
			return fmt.Errorf("failed to persist credential digest: %w", err)
		}
	}

	return nil
}

// distributeAgentReward pays an accumulated native agent reward: the trust
// deposit portion is staked on behalf of the agent, the remainder is sent to
// the agent's account. label distinguishes user vs wallet agent in errors.
func (ms msgServer) distributeAgentReward(ctx sdk.Context, payer sdk.AccAddress, agentParticipantID uint64, accumulated, trustDepositRate math.LegacyDec, label string) error {
	agentParticipant, err := ms.Participant.Get(ctx, agentParticipantID)
	if err != nil {
		return fmt.Errorf("failed to get %s participant: %w", label, err)
	}

	agentTrustDeposit, err := decToUint64(accumulated.Mul(trustDepositRate), label+" trust deposit")
	if err != nil {
		return err
	}
	total, err := decToUint64(accumulated, label+" reward")
	if err != nil {
		return err
	}
	feesToAccount := total - agentTrustDeposit

	agentCorpAcct, err := ms.corpAccountFromID(ctx, agentParticipant.CorporationId)
	if err != nil {
		return err
	}

	if feesToAccount > 0 {
		agentAddr, err := sdk.AccAddressFromBech32(agentCorpAcct)
		if err != nil {
			return fmt.Errorf("invalid %s grantee address: %w", label, err)
		}
		amt, err := uint64ToInt64(feesToAccount, label+" fees to account")
		if err != nil {
			return err
		}
		if err := ms.bankKeeper.SendCoins(ctx, payer, agentAddr, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, amt))); err != nil {
			return fmt.Errorf("failed to transfer %s reward: %w", label, err)
		}
	}

	if agentTrustDeposit > 0 {
		td, err := uint64ToInt64(agentTrustDeposit, label+" trust deposit")
		if err != nil {
			return err
		}
		if err := ms.trustDeposit.AdjustTrustDepositOnBehalf(ctx, agentCorpAcct, payer, td); err != nil {
			return fmt.Errorf("failed to adjust %s trust deposit: %w", label, err)
		}
		agentParticipant.Deposit += agentTrustDeposit
		if err := ms.Keeper.UpdateParticipant(ctx, agentParticipant); err != nil {
			return fmt.Errorf("failed to update %s participant deposit: %w", label, err)
		}
	}

	return nil
}

func (ms msgServer) validateSessionAccess(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession) error {
	existingSession, err := ms.ParticipantSession.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil // New session case
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// existing_entry.corporation MUST be equal to corporation
	msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	if existingSession.CorporationId != msgCorpId {
		return fmt.Errorf("session corporation does not match: expected %d, got %s", existingSession.CorporationId, msg.Corporation)
	}

	// existing_entry.vs_operator MUST be equal to operator
	if existingSession.VsOperator != msg.Operator {
		return fmt.Errorf("session vs_operator does not match: expected %s, got %s", existingSession.VsOperator, msg.Operator)
	}

	return nil
}

func (ms msgServer) createOrUpdateSession(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, now time.Time) error {
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	session := &types.ParticipantSession{
		Id:            msg.Id,
		CorporationId: corporationId,
		VsOperator:    msg.Operator,
		Modified:      &now,
	}

	existingSession, err := ms.ParticipantSession.Get(ctx, msg.Id)
	if err == nil {
		// Update existing session
		session = &existingSession
		session.Modified = &now
	} else if errors.Is(err, collections.ErrNotFound) {
		// New session
		session.Created = &now
	} else {
		return err
	}

	// Create ParticipantSessionRecord with its own uint64 id (sequential within
	// the session). agent_participant_id now lives on the record per spec v4-rc2.
	record := &types.ParticipantSessionRecord{
		Id:                       uint64(len(session.SessionRecords) + 1),
		Created:                  &now,
		IssuerParticipantId:      msg.IssuerParticipantId,
		VerifierParticipantId:    msg.VerifierParticipantId,
		WalletAgentParticipantId: msg.WalletAgentParticipantId,
		AgentParticipantId:       msg.AgentParticipantId,
	}

	// Add the record to session.session_records
	session.SessionRecords = append(session.SessionRecords, record)

	return ms.ParticipantSession.Set(ctx, msg.Id, *session)
}

// findBeneficiaries gets the set of participants that should receive fees
func (ms msgServer) findBeneficiaries(ctx sdk.Context, issuerParticipantId, verifierParticipantId uint64) ([]types.Participant, error) {
	var foundParticipants []types.Participant

	// Helper function to check if a participant is already in the slice
	containsParticipant := func(id uint64) bool {
		for _, p := range foundParticipants {
			if p.Id == id {
				return true
			}
		}
		return false
	}

	if issuerParticipantId == 0 && verifierParticipantId == 0 {
		return nil, fmt.Errorf("at least one of issuer_participant_id or verifier_participant_id must be provided")
	}

	// MOD-PP-QRY-4-3 has no OPEN-mode special case: self-created OPEN participants
	// carry validator_participant_id = ECOSYSTEM, so the walks below include it.

	// Process issuer participant hierarchy if provided.
	if issuerParticipantId != 0 {
		issuerParticipant, err := ms.Participant.Get(ctx, issuerParticipantId)
		if err != nil {
			return nil, fmt.Errorf("issuer participant not found: %w", err)
		}

		// Follow the validator chain up
		if issuerParticipant.ValidatorParticipantId != 0 {
			currentParticipantID := issuerParticipant.ValidatorParticipantId
			visited := map[uint64]bool{}
			for currentParticipantID != 0 && !visited[currentParticipantID] {
				visited[currentParticipantID] = true
				currentParticipant, err := ms.Participant.Get(ctx, currentParticipantID)
				if err != nil {
					return nil, fmt.Errorf("failed to get participant: %w", err)
				}

				// Add to set if valid and not already included
				if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil && !containsParticipant(currentParticipantID) {
					foundParticipants = append(foundParticipants, currentParticipant)
				}

				// Move up
				currentParticipantID = currentParticipant.ValidatorParticipantId
			}
		}
	}

	// Process verifier participant hierarchy if provided
	if verifierParticipantId != 0 {
		// First add issuer participant to the set if provided
		if issuerParticipantId != 0 {
			issuerParticipant, err := ms.Participant.Get(ctx, issuerParticipantId)
			if err == nil && issuerParticipant.Revoked == nil && !containsParticipant(issuerParticipantId) {
				foundParticipants = append(foundParticipants, issuerParticipant)
			}
		}

		// Then process verifier's validator chain
		verifierParticipant, err := ms.Participant.Get(ctx, verifierParticipantId)
		if err != nil {
			return nil, fmt.Errorf("verifier participant not found: %w", err)
		}

		if verifierParticipant.ValidatorParticipantId != 0 {
			currentParticipantID := verifierParticipant.ValidatorParticipantId
			visited := map[uint64]bool{}
			for currentParticipantID != 0 && !visited[currentParticipantID] {
				visited[currentParticipantID] = true
				currentParticipant, err := ms.Participant.Get(ctx, currentParticipantID)
				if err != nil {
					return nil, fmt.Errorf("failed to get participant: %w", err)
				}

				// Add to set if valid and not already included
				if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil && !containsParticipant(currentParticipantID) {
					foundParticipants = append(foundParticipants, currentParticipant)
				}

				// Move up
				currentParticipantID = currentParticipant.ValidatorParticipantId
			}
		}
	}

	return foundParticipants, nil
}
