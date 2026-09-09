package lib

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

const (
	COUNCIL_MEMBER1_NAME = "council_member1"
	COUNCIL_MEMBER2_NAME = "council_member2"
)

func CouncilAuthority() string { return poatypes.CouncilAuthorityBech32(addressPrefix) }

func SubmitCouncilProposal(client cosmosclient.Client, ctx context.Context, proposer cosmosaccount.Account, title, summary string, msgs ...sdk.Msg) (uint64, error) {
	return SubmitGroupProposal(client, ctx, proposer, CouncilAuthority(), msgs, title, summary)
}

func VoteCouncilProposal(client cosmosclient.Client, ctx context.Context, voter cosmosaccount.Account, id uint64, tryExec bool) error {
	return VoteOnGroupProposal(client, ctx, voter, id, tryExec)
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
