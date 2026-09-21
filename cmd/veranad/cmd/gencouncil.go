package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	icagenesistypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/spf13/cobra"

	"github.com/verana-labs/verana-node/app"
	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

const (
	flagPercentage    = "percentage"
	flagVotingPeriod  = "voting-period"
	flagMinExecPeriod = "min-execution-period"
	flagCouncilMeta   = "metadata"
	flagMaxValidators = "max-validators"
	flagUnbondingTime = "unbonding-time"
)

// ICA packets bypass the ante, so the host allow-list must not reach staking.
var icaHostAllowMessages = []string{
	"/cosmos.bank.v1beta1.MsgSend",
}

func unmarshalIfSet(cdc codec.Codec, raw []byte, out proto.Message) error {
	if len(raw) == 0 {
		return nil
	}
	return cdc.UnmarshalJSON(raw, out)
}

func AddCouncilCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-council [member][,member...]",
		Short: "Seat the Verana Council in genesis",
		Long: `Seat the Verana Council in genesis: one weight-1 x/group member per address or key name,
a self-administered 2/3 decision policy, slashing fractions set to 0 and staking max_validators,
so every seated validator keeps exactly one unit of consensus power. Run before gentx.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			kb, err := keyring.New(sdk.KeyringServiceName(), keyringBackend, clientCtx.HomeDir, bufio.NewReader(cmd.InOrStdin()), cdc)
			if err != nil {
				return err
			}
			var members []string
			seen := map[string]bool{}
			for _, raw := range strings.Split(args[0], ",") {
				raw = strings.TrimSpace(raw)
				addr, err := sdk.AccAddressFromBech32(raw)
				if err != nil {
					info, kerr := kb.Key(raw)
					if kerr != nil {
						return fmt.Errorf("member %q is neither an address nor a key: %w", raw, kerr)
					}
					if addr, err = info.GetAddress(); err != nil {
						return err
					}
				}
				if seen[addr.String()] {
					return fmt.Errorf("duplicate member %s", addr)
				}
				seen[addr.String()] = true
				members = append(members, addr.String())
			}

			percentage, _ := cmd.Flags().GetString(flagPercentage)
			votingPeriod, _ := cmd.Flags().GetDuration(flagVotingPeriod)
			minExec, _ := cmd.Flags().GetDuration(flagMinExecPeriod)
			metadata, _ := cmd.Flags().GetString(flagCouncilMeta)
			maxValidators, _ := cmd.Flags().GetUint32(flagMaxValidators)
			unbondingTime, _ := cmd.Flags().GetDuration(flagUnbondingTime)

			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			var existing group.GenesisState
			if err := unmarshalIfSet(cdc, appState[group.ModuleName], &existing); err != nil {
				return err
			}
			if len(existing.Groups) > 0 || len(existing.GroupPolicies) > 0 {
				return fmt.Errorf("genesis already contains groups; add-council must run on a fresh genesis")
			}

			gs, err := poatypes.CouncilGenesis(app.CouncilAuthority, members, metadata, percentage, votingPeriod, minExec, genDoc.GenesisTime)
			if err != nil {
				return err
			}
			if err := gs.Validate(); err != nil {
				return err
			}
			appState[group.ModuleName] = cdc.MustMarshalJSON(gs)

			staking := stakingtypes.DefaultGenesisState()
			if err := unmarshalIfSet(cdc, appState[stakingtypes.ModuleName], staking); err != nil {
				return err
			}
			staking.Params.MaxValidators = maxValidators
			if unbondingTime > 0 {
				staking.Params.UnbondingTime = unbondingTime
			}
			appState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(staking)

			slashing := slashingtypes.DefaultGenesisState()
			if err := unmarshalIfSet(cdc, appState[slashingtypes.ModuleName], slashing); err != nil {
				return err
			}
			slashing.Params.SlashFractionDowntime = math.LegacyZeroDec()
			slashing.Params.SlashFractionDoubleSign = math.LegacyZeroDec()
			appState[slashingtypes.ModuleName] = cdc.MustMarshalJSON(slashing)

			ica := icagenesistypes.DefaultGenesis()
			if err := unmarshalIfSet(cdc, appState[icatypes.ModuleName], ica); err != nil {
				return err
			}
			ica.HostGenesisState.Params.AllowMessages = icaHostAllowMessages
			appState[icatypes.ModuleName] = cdc.MustMarshalJSON(ica)

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return err
			}
			genDoc.AppState = appStateJSON
			if err := genutil.ExportGenesisFile(genDoc, genFile); err != nil {
				return err
			}
			cmd.Printf("council policy %s with %d member(s)\n", app.CouncilAuthority, len(members))
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(flagPercentage, poatypes.CouncilPercentage, "yes-vote share of seated members required to pass")
	cmd.Flags().Duration(flagVotingPeriod, 14*24*time.Hour, "ballot window")
	cmd.Flags().Duration(flagMinExecPeriod, 48*time.Hour, "earliest execution after submission")
	cmd.Flags().String(flagCouncilMeta, "Verana Council", "group and policy metadata")
	cmd.Flags().Uint32(flagMaxValidators, 25, "staking max_validators")
	cmd.Flags().Duration(flagUnbondingTime, 0, "staking unbonding_time; 0 keeps the default")
	return cmd
}
