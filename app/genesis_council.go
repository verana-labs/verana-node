package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	icagenesistypes "github.com/cosmos/ibc-go/v11/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v11/modules/apps/27-interchain-accounts/types"
	"github.com/verana-labs/cosmos-group"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

// GenesisCouncil is the council of corporations seated at genesis (GCO).
type GenesisCouncil struct {
	Corporations       []GenesisCorporation
	Percentage         string
	VotingPeriod       time.Duration
	MinExecutionPeriod time.Duration
	Metadata           string
	MaxValidators      uint32
	UnbondingTime      time.Duration
}

// GenesisCorporation gets its own group and policy; the policy account is its policy_address.
// A zero VotingPeriod takes the council's.
type GenesisCorporation struct {
	Did                string
	Language           string
	Members            []string
	Threshold          string
	VotingPeriod       time.Duration
	MinExecutionPeriod time.Duration
	Funds              sdk.Coins
	Validator          *GenesisValidator
}

// GenesisValidator is operated by the corporation's policy address and bonded from the genesis supply.
type GenesisValidator struct {
	Moniker string
	PubKey  cryptotypes.PubKey
}

// ICA packets bypass the ante, so the host allow-list must not reach staking.
var icaHostAllowMessages = []string{"/cosmos.bank.v1beta1.MsgSend"}

func readState(cdc codec.JSONCodec, appState map[string]json.RawMessage, module string, out proto.Message) error {
	if len(appState[module]) == 0 {
		return nil
	}
	return cdc.UnmarshalJSON(appState[module], out)
}

// SetGenesisCouncil writes the council group, the corporations with their groups, policies and
// Corporation entries, and their genesis validators into appState.
func SetGenesisCouncil(cdc codec.Codec, appState map[string]json.RawMessage, c GenesisCouncil, now time.Time) error {
	n := len(c.Corporations)
	if n < poatypes.MinCouncilCorporations {
		return fmt.Errorf("the council needs at least %d corporations, got %d", poatypes.MinCouncilCorporations, n)
	}

	gg := group.NewGenesisState()
	if err := readState(cdc, appState, group.ModuleName, gg); err != nil {
		return err
	}
	if len(gg.Groups) > 0 || len(gg.GroupPolicies) > 0 {
		return fmt.Errorf("genesis already contains groups; the council must be seated on a fresh genesis")
	}

	policies := make([]string, n)
	for i := range c.Corporations {
		policies[i] = poatypes.GroupPolicyAddress(uint64(i + 2)).String()
	}
	if err := addGroup(gg, 1, CouncilAuthority, c.Metadata, policies, &group.PercentageDecisionPolicy{
		Percentage: c.Percentage,
		Windows:    &group.DecisionPolicyWindows{VotingPeriod: c.VotingPeriod, MinExecutionPeriod: c.MinExecutionPeriod},
	}, now); err != nil {
		return err
	}
	for i, co := range c.Corporations {
		threshold := co.Threshold
		if threshold == "" {
			threshold = "1"
		}
		voting := co.VotingPeriod
		if voting == 0 {
			voting = c.VotingPeriod
		}
		if err := addGroup(gg, uint64(i+2), policies[i], co.Did, co.Members, &group.ThresholdDecisionPolicy{
			Threshold: threshold,
			Windows:   &group.DecisionPolicyWindows{VotingPeriod: voting, MinExecutionPeriod: co.MinExecutionPeriod},
		}, now); err != nil {
			return fmt.Errorf("corporation %s: %w", co.Did, err)
		}
	}
	gg.GroupSeq = uint64(n + 1)
	gg.GroupPolicySeq = uint64(n + 1)
	if err := gg.Validate(); err != nil {
		return err
	}

	if err := addPolicyAccounts(cdc, appState, n+1); err != nil {
		return err
	}

	coGen := cotypes.DefaultGenesis()
	if err := readState(cdc, appState, cotypes.ModuleName, coGen); err != nil {
		return err
	}
	id := coGen.CorporationCounter
	for i, co := range c.Corporations {
		id++
		coGen.Corporations = append(coGen.Corporations, cotypes.Corporation{
			Id: id, PolicyAddress: policies[i], Did: co.Did, Language: co.Language,
			Created: now, Modified: now, ActiveVersion: 1,
		})
	}
	coGen.CorporationCounter = id
	if err := coGen.Validate(); err != nil {
		return err
	}

	staking := stakingtypes.DefaultGenesisState()
	if err := readState(cdc, appState, stakingtypes.ModuleName, staking); err != nil {
		return err
	}
	if c.MaxValidators > 0 {
		staking.Params.MaxValidators = c.MaxValidators
	}
	if c.UnbondingTime > 0 {
		staking.Params.UnbondingTime = c.UnbondingTime
	}
	bond := poatypes.ValidatorBond(staking.Params.BondDenom)
	bonded := sdk.NewCoins()
	for i, co := range c.Corporations {
		if co.Validator == nil {
			continue
		}
		v, d, err := genesisValidator(policies[i], co.Validator, bond, now)
		if err != nil {
			return fmt.Errorf("corporation %s: %w", co.Did, err)
		}
		staking.Validators = append(staking.Validators, v)
		staking.Delegations = append(staking.Delegations, d)
		bonded = bonded.Add(bond)
	}

	bank := banktypes.DefaultGenesisState()
	if err := readState(cdc, appState, banktypes.ModuleName, bank); err != nil {
		return err
	}
	credit(bank, authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String(), bonded)
	for i, co := range c.Corporations {
		credit(bank, policies[i], co.Funds)
	}

	slashing := slashingtypes.DefaultGenesisState()
	if err := readState(cdc, appState, slashingtypes.ModuleName, slashing); err != nil {
		return err
	}
	slashing.Params.SlashFractionDowntime = math.LegacyZeroDec()
	slashing.Params.SlashFractionDoubleSign = math.LegacyZeroDec()

	ica := icagenesistypes.DefaultGenesis()
	if err := readState(cdc, appState, icatypes.ModuleName, ica); err != nil {
		return err
	}
	ica.HostGenesisState.Params.AllowMessages = icaHostAllowMessages

	for module, state := range map[string]proto.Message{
		group.ModuleName: gg, cotypes.ModuleName: coGen, stakingtypes.ModuleName: staking,
		banktypes.ModuleName: bank, slashingtypes.ModuleName: slashing, icatypes.ModuleName: ica,
	} {
		bz, err := cdc.MarshalJSON(state)
		if err != nil {
			return err
		}
		appState[module] = bz
	}
	return nil
}

// credit adds coins to a genesis balance and, when the supply is set, to the supply.
func credit(bank *banktypes.GenesisState, addr string, coins sdk.Coins) {
	if coins.IsZero() {
		return
	}
	found := false
	for i := range bank.Balances {
		if bank.Balances[i].Address == addr {
			bank.Balances[i].Coins = bank.Balances[i].Coins.Add(coins...)
			found = true
		}
	}
	if !found {
		bank.Balances = append(bank.Balances, banktypes.Balance{Address: addr, Coins: coins})
	}
	if !bank.Supply.Empty() {
		bank.Supply = bank.Supply.Add(coins...)
	}
}

func addGroup(gg *group.GenesisState, id uint64, policy, metadata string, members []string, decision group.DecisionPolicy, now time.Time) error {
	if len(members) == 0 {
		return fmt.Errorf("group %d has no members", id)
	}
	seen := map[string]bool{}
	for _, m := range members {
		if _, err := sdk.AccAddressFromBech32(m); err != nil {
			return fmt.Errorf("member %q: %w", m, err)
		}
		if seen[m] {
			return fmt.Errorf("duplicate member %s", m)
		}
		seen[m] = true
		gg.GroupMembers = append(gg.GroupMembers, &group.GroupMember{GroupId: id, Member: &group.Member{Address: m, Weight: "1", AddedAt: now}})
	}
	gg.Groups = append(gg.Groups, &group.GroupInfo{
		Id: id, Admin: policy, Metadata: metadata, Version: 1, TotalWeight: strconv.Itoa(len(members)), CreatedAt: now,
	})
	info := &group.GroupPolicyInfo{Address: policy, GroupId: id, Admin: policy, Metadata: metadata, Version: 1, CreatedAt: now}
	if err := info.SetDecisionPolicy(decision); err != nil {
		return err
	}
	gg.GroupPolicies = append(gg.GroupPolicies, info)
	return nil
}

// addPolicyAccounts mirrors x/group, which gives every policy account its module credential.
func addPolicyAccounts(cdc codec.Codec, appState map[string]json.RawMessage, count int) error {
	auth := authtypes.GetGenesisStateFromAppState(cdc, appState)
	accs, err := authtypes.UnpackAccounts(auth.Accounts)
	if err != nil {
		return err
	}
	for seq := 1; seq <= count; seq++ {
		acc, err := authtypes.NewBaseAccountWithPubKey(poatypes.GroupPolicyCredential(uint64(seq)))
		if err != nil {
			return err
		}
		if accs.Contains(acc.GetAddress()) {
			return fmt.Errorf("genesis already has an account at policy address %s", acc.GetAddress())
		}
		accs = append(accs, acc)
	}
	packed, err := authtypes.PackAccounts(authtypes.SanitizeGenesisAccounts(accs))
	if err != nil {
		return err
	}
	auth.Accounts = packed
	bz, err := cdc.MarshalJSON(&auth)
	if err != nil {
		return err
	}
	appState[authtypes.ModuleName] = bz
	return nil
}

func genesisValidator(operator string, gv *GenesisValidator, bond sdk.Coin, now time.Time) (stakingtypes.Validator, stakingtypes.Delegation, error) {
	if gv.PubKey == nil {
		return stakingtypes.Validator{}, stakingtypes.Delegation{}, fmt.Errorf("validator pubkey is required")
	}
	pk, err := codectypes.NewAnyWithValue(gv.PubKey)
	if err != nil {
		return stakingtypes.Validator{}, stakingtypes.Delegation{}, err
	}
	acc, err := sdk.AccAddressFromBech32(operator)
	if err != nil {
		return stakingtypes.Validator{}, stakingtypes.Delegation{}, err
	}
	valoper := sdk.ValAddress(acc).String()
	shares := math.LegacyNewDecFromInt(bond.Amount)
	// unbonded so InitGenesis bonds it and the slashing hook records its signing info
	v := stakingtypes.Validator{
		OperatorAddress:   valoper,
		ConsensusPubkey:   pk,
		Status:            stakingtypes.Unbonded,
		Tokens:            bond.Amount,
		DelegatorShares:   shares,
		Description:       stakingtypes.NewDescription(gv.Moniker, "", "", "", ""),
		UnbondingTime:     time.Unix(0, 0).UTC(),
		Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		MinSelfDelegation: math.OneInt(),
	}
	v.Commission.UpdateTime = now
	return v, stakingtypes.NewDelegation(operator, valoper, shares), nil
}
