package lib

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"github.com/verana-labs/cosmos-group"

	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

const (
	COUNCIL_MEMBER1_NAME = "council_member1"
	COUNCIL_MEMBER2_NAME = "council_member2"
)

func CouncilAuthority() string { return poatypes.CouncilAuthorityBech32(addressPrefix) }

// corporationOf returns the council seat whose corporation group has account as a member.
func corporationOf(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account) (string, error) {
	addr, err := account.Address(addressPrefix)
	if err != nil {
		return "", err
	}
	council, err := poatypes.NewQueryClient(client.Context()).Council(ctx, &poatypes.QueryCouncilRequest{})
	if err != nil {
		return "", err
	}
	seats := map[string]bool{}
	for _, m := range council.Members {
		seats[m] = true
	}
	gq := group.NewQueryClient(client.Context())
	groups, err := gq.GroupsByMember(ctx, &group.QueryGroupsByMemberRequest{Address: addr})
	if err != nil {
		return "", err
	}
	// a corporation's group is administered by its own policy
	for _, g := range groups.Groups {
		if seats[g.Admin] {
			return g.Admin, nil
		}
	}
	return "", fmt.Errorf("%s is a member of no council corporation", addr)
}

// asCorporation has account's corporation execute msgs at once through its threshold-1 group.
// It returns the proposal ids submitted in the tx, the corporation's own first.
func asCorporation(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, msgs ...sdk.Msg) ([]uint64, error) {
	corp, err := corporationOf(client, ctx, account)
	if err != nil {
		return nil, err
	}
	addr, err := account.Address(addressPrefix)
	if err != nil {
		return nil, err
	}
	anyMsgs := make([]*codectypes.Any, len(msgs))
	for i, m := range msgs {
		if anyMsgs[i], err = codectypes.NewAnyWithValue(m); err != nil {
			return nil, err
		}
	}
	txResp, err := client.BroadcastTx(ctx, account, &group.MsgSubmitProposal{
		GroupPolicyAddress: corp, Proposers: []string{addr}, Messages: anyMsgs,
		Exec: group.Exec_EXEC_TRY, Title: "council act", Summary: "executed by the corporation",
	})
	if err != nil {
		return nil, fmt.Errorf("corporation proposal: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return nil, fmt.Errorf("corporation proposal failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	var ids []uint64
	for _, ev := range txResp.TxResponse.Events {
		attrs := map[string]string{}
		for _, a := range ev.Attributes {
			attrs[a.Key] = strings.Trim(a.Value, "\"")
		}
		switch ev.Type {
		case "cosmos.group.v1.EventSubmitProposal":
			id, err := strconv.ParseUint(attrs["proposal_id"], 10, 64)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		case "cosmos.group.v1.EventExec":
			if len(ids) > 0 && attrs["proposal_id"] == strconv.FormatUint(ids[0], 10) && attrs["result"] != group.PROPOSAL_EXECUTOR_RESULT_SUCCESS.String() {
				return nil, fmt.Errorf("corporation %s could not execute: %s", corp, attrs["logs"])
			}
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("corporation proposal id not found in events")
	}
	return ids, nil
}

// SubmitCouncilProposal submits msgs to the council on behalf of proposer's corporation.
func SubmitCouncilProposal(client cosmosclient.Client, ctx context.Context, proposer cosmosaccount.Account, title, summary string, msgs ...sdk.Msg) (uint64, error) {
	corp, err := corporationOf(client, ctx, proposer)
	if err != nil {
		return 0, err
	}
	submit := &group.MsgSubmitProposal{GroupPolicyAddress: CouncilAuthority(), Proposers: []string{corp}, Title: title, Summary: summary}
	if err := submit.SetMsgs(msgs); err != nil {
		return 0, err
	}
	ids, err := asCorporation(client, ctx, proposer, submit)
	if err != nil {
		return 0, err
	}
	if len(ids) < 2 {
		return 0, fmt.Errorf("council proposal id not found in events")
	}
	fmt.Printf("✅ Submitted council proposal %d\n", ids[len(ids)-1])
	return ids[len(ids)-1], nil
}

// VoteCouncilProposal votes yes on behalf of voter's corporation.
func VoteCouncilProposal(client cosmosclient.Client, ctx context.Context, voter cosmosaccount.Account, id uint64, tryExec bool) error {
	corp, err := corporationOf(client, ctx, voter)
	if err != nil {
		return err
	}
	vote := &group.MsgVote{ProposalId: id, Voter: corp, Option: group.VOTE_OPTION_YES}
	if tryExec {
		vote.Exec = group.Exec_EXEC_TRY
	}
	if _, err := asCorporation(client, ctx, voter, vote); err != nil {
		return err
	}
	fmt.Printf("✅ %s voted YES on council proposal %d\n", corp, id)
	return nil
}

func ExecCouncilProposal(client cosmosclient.Client, ctx context.Context, executor cosmosaccount.Account, id uint64) error {
	return ExecGroupProposal(client, ctx, executor, id)
}

// nil means the proposal was pruned after a successful execution.
func QueryCouncilProposal(client cosmosclient.Client, ctx context.Context, id uint64) (*group.Proposal, error) {
	res, err := group.NewQueryClient(client.Context()).Proposal(ctx, &group.QueryProposalRequest{ProposalId: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	return res.Proposal, nil
}

func PassCouncilProposal(client cosmosclient.Client, ctx context.Context, id uint64) error {
	cooluser := GetAccount(client, COOLUSER_NAME)
	if err := VoteCouncilProposal(client, ctx, cooluser, id, false); err != nil {
		return err
	}
	if err := VoteCouncilProposal(client, ctx, GetAccount(client, COUNCIL_MEMBER1_NAME), id, false); err != nil {
		return err
	}
	for attempt := 0; attempt < 20; attempt++ {
		if err := ExecCouncilProposal(client, ctx, cooluser, id); err != nil {
			return err
		}
		p, err := QueryCouncilProposal(client, ctx, id)
		if err != nil {
			return err
		}
		if p == nil || p.ExecutorResult == group.PROPOSAL_EXECUTOR_RESULT_SUCCESS {
			fmt.Printf("✅ Council proposal %d executed\n", id)
			return nil
		}
		if p.Status != group.PROPOSAL_STATUS_ACCEPTED {
			return fmt.Errorf("proposal %d not accepted: status %s executor %s", id, p.Status, p.ExecutorResult)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("proposal %d accepted but never executed", id)
}
