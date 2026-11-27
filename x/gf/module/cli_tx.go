package gf

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/verana-labs/verana-node/x/gf/types"
)

const flagEcosystemID = "ecosystem-id"

// GetTxCmd implements the autocli.HasCustomTxCommand interface.
func (am AppModule) GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Transactions commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(CmdAddGovernanceFrameworkDocument())
	cmd.AddCommand(CmdIncreaseActiveGovernanceFrameworkVersion())
	return cmd
}

// CmdAddGovernanceFrameworkDocument returns the [MOD-GF-MSG-1] command.
func CmdAddGovernanceFrameworkDocument() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-governance-framework-document [corporation] [signer] [doc-language] [doc-url] [doc-digest-sri] [version]",
		Short: "[MOD-GF-MSG-1] Add a governance framework document",
		Long:  "Add or replace a GovernanceFrameworkDocument for a draft GovernanceFrameworkVersion owned by either an Ecosystem (set --ecosystem-id) or the signing Corporation's own CGF (omit --ecosystem-id).",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			version, err := strconv.ParseUint(args[5], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid version: %w", err)
			}
			ecosystemID, err := cmd.Flags().GetUint64(flagEcosystemID)
			if err != nil {
				return err
			}
			msg := &types.MsgAddGovernanceFrameworkDocument{
				Corporation:  args[0],
				Signer:       args[1],
				EcosystemId:  ecosystemID,
				DocLanguage:  args[2],
				DocUrl:       args[3],
				DocDigestSri: args[4],
				Version:      uint32(version),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64(flagEcosystemID, 0, "Target Ecosystem ID (omit for Corporation-targeted CGF).")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdIncreaseActiveGovernanceFrameworkVersion returns the [MOD-GF-MSG-2] command.
func CmdIncreaseActiveGovernanceFrameworkVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "increase-active-gf-version [corporation] [signer]",
		Short: "[MOD-GF-MSG-2] Activate the next governance framework version",
		Long:  "Activate the next governance framework version for either an Ecosystem (set --ecosystem-id) or the signing Corporation's own CGF (omit --ecosystem-id). Requires the next version to exist and contain a document in the subject's default language.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			ecosystemID, err := cmd.Flags().GetUint64(flagEcosystemID)
			if err != nil {
				return err
			}
			msg := &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: args[0],
				Signer:      args[1],
				EcosystemId: ecosystemID,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64(flagEcosystemID, 0, "Target Ecosystem ID (omit for Corporation-targeted CGF).")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
