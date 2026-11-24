package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
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

type councilFile struct {
	Corporations []struct {
		Did                string   `json:"did"`
		Language           string   `json:"language"`
		Members            []string `json:"members"`
		Threshold          string   `json:"threshold"`
		VotingPeriod       string   `json:"voting_period"`
		MinExecutionPeriod string   `json:"min_execution_period"`
		Funds              string   `json:"funds"`
		Validator          *struct {
			Moniker string          `json:"moniker"`
			PubKey  json.RawMessage `json:"pubkey"`
		} `json:"validator"`
	} `json:"corporations"`
}

func AddCouncilCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-council [council.json]",
		Short: "Seat the council of corporations in genesis",
		Long: `Seat the council in genesis from a JSON file listing at least three corporations:
{"corporations":[{"did":"did:...","language":"en","members":["key-or-address"],"threshold":"1",
"voting_period":"72h","min_execution_period":"24h","funds":"10000000uvna",
"validator":{"moniker":"node1","pubkey":<output of 'veranad comet show-validator'>}}]}
Each corporation gets its own group and policy, whose account is its policy_address and holds the given funds,
and a Corporation entry. Threshold defaults to 1, voting_period to the council's, min_execution_period to 0.
A corporation with a validator operates it from its policy address, bonded from the genesis supply; no gentx is used.
The council is a self-administered 2/3 group of the corporations' policy addresses; slashing fractions are set to 0.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			raw, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var file councilFile
			if err := json.Unmarshal(raw, &file); err != nil {
				return fmt.Errorf("parse %s: %w", args[0], err)
			}

			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			kb, err := keyring.New(sdk.KeyringServiceName(), keyringBackend, clientCtx.HomeDir, bufio.NewReader(cmd.InOrStdin()), cdc)
			if err != nil {
				return err
			}
			resolve := func(raw string) (string, error) {
				if addr, err := sdk.AccAddressFromBech32(raw); err == nil {
					return addr.String(), nil
				}
				info, err := kb.Key(raw)
				if err != nil {
					return "", fmt.Errorf("member %q is neither an address nor a key: %w", raw, err)
				}
				addr, err := info.GetAddress()
				if err != nil {
					return "", err
				}
				return addr.String(), nil
			}

			spec := app.GenesisCouncil{}
			spec.Percentage, _ = cmd.Flags().GetString(flagPercentage)
			spec.VotingPeriod, _ = cmd.Flags().GetDuration(flagVotingPeriod)
			spec.MinExecutionPeriod, _ = cmd.Flags().GetDuration(flagMinExecPeriod)
			spec.Metadata, _ = cmd.Flags().GetString(flagCouncilMeta)
			spec.MaxValidators, _ = cmd.Flags().GetUint32(flagMaxValidators)
			spec.UnbondingTime, _ = cmd.Flags().GetDuration(flagUnbondingTime)
			for _, c := range file.Corporations {
				gc := app.GenesisCorporation{Did: c.Did, Language: c.Language, Threshold: c.Threshold}
				if gc.VotingPeriod, err = optionalDuration(c.VotingPeriod); err != nil {
					return fmt.Errorf("corporation %s: voting_period: %w", c.Did, err)
				}
				if gc.MinExecutionPeriod, err = optionalDuration(c.MinExecutionPeriod); err != nil {
					return fmt.Errorf("corporation %s: min_execution_period: %w", c.Did, err)
				}
				if c.Funds != "" {
					if gc.Funds, err = sdk.ParseCoinsNormalized(c.Funds); err != nil {
						return fmt.Errorf("corporation %s: funds: %w", c.Did, err)
					}
				}
				for _, m := range c.Members {
					addr, err := resolve(m)
					if err != nil {
						return err
					}
					gc.Members = append(gc.Members, addr)
				}
				if c.Validator != nil {
					var pk cryptotypes.PubKey
					if err := cdc.UnmarshalInterfaceJSON(c.Validator.PubKey, &pk); err != nil {
						return fmt.Errorf("corporation %s: validator pubkey: %w", c.Did, err)
					}
					gc.Validator = &app.GenesisValidator{Moniker: c.Validator.Moniker, PubKey: pk}
				}
				spec.Corporations = append(spec.Corporations, gc)
			}

			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}
			if err := app.SetGenesisCouncil(cdc, appState, spec, genDoc.GenesisTime); err != nil {
				return err
			}
			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return err
			}
			genDoc.AppState = appStateJSON
			if err := genutil.ExportGenesisFile(genDoc, genFile); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "council %s\n", app.CouncilAuthority)
			for i, c := range spec.Corporations {
				fmt.Fprintf(out, "corporation %s %s\n", c.Did, poatypes.GroupPolicyAddress(uint64(i+2)))
			}
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(flagPercentage, poatypes.CouncilPercentage, "yes-vote share of seated corporations required to pass")
	cmd.Flags().Duration(flagVotingPeriod, 14*24*time.Hour, "ballot window of the council and of the genesis corporations")
	cmd.Flags().Duration(flagMinExecPeriod, 48*time.Hour, "earliest council execution after submission")
	cmd.Flags().String(flagCouncilMeta, "Verana Council", "council group and policy metadata")
	cmd.Flags().Uint32(flagMaxValidators, 25, "staking max_validators")
	cmd.Flags().Duration(flagUnbondingTime, 0, "staking unbonding_time; 0 keeps the default")
	return cmd
}

func optionalDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
