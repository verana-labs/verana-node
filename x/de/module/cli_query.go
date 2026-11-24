package de

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GetQueryCmd implements the autocli.HasCustomQueryCommand interface.
// This is needed because autocli's amino JSON encoder cannot properly render
// gogo proto types with extensions (stdtime, stdduration, castrepeated) used
// in DelegateAuthorization. The custom command uses the gogo proto codec directly.
func (am AppModule) GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
	}

	cmd.AddCommand(CmdListDelegateAuthorizations())
	cmd.AddCommand(CmdListVSOperatorAuthorizations())
	cmd.AddCommand(CmdGetDelegateAuthorization())
	cmd.AddCommand(CmdGetVSOperatorAuthorization())

	return cmd
}

// CmdGetDelegateAuthorization returns a cobra command for the
// [MOD-DE-QRY-3] GetDelegateAuthorization query.
func CmdGetDelegateAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-delegate-authorization [corporation-id] [delegate]",
		Short: "Get the delegate authorization of a corporation and a delegate",
		Long:  "[MOD-DE-QRY-3] Get the delegate authorization keyed by (corporation_id, delegate).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid corporation id: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GetDelegateAuthorization(cmd.Context(), &types.QueryGetDelegateAuthorizationRequest{
				CorporationId: corporationID,
				Delegate:      args[1],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdGetVSOperatorAuthorization returns a cobra command for the
// [MOD-DE-QRY-4] GetVSOperatorAuthorization query.
func CmdGetVSOperatorAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-vs-operator-authorization [id]",
		Short: "Get a VS operator authorization by id",
		Long:  "[MOD-DE-QRY-4] Get a single VS operator authorization (with its records) by its id.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GetVSOperatorAuthorization(cmd.Context(), &types.QueryGetVSOperatorAuthorizationRequest{Id: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdListDelegateAuthorizations returns a cobra command for the
// [MOD-DE-QRY-1] ListDelegateAuthorizations query.
func CmdListDelegateAuthorizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-delegate-authorizations [corporation-id]",
		Short: "List the delegate authorizations of a corporation",
		Long:  "[MOD-DE-QRY-1] List the delegate authorizations granted by a corporation.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid corporation id: %w", err)
			}
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListDelegateAuthorizations(cmd.Context(), &types.QueryListDelegateAuthorizationsRequest{
				CorporationId:   corporationID,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint32("limit", 64, "maximum number of results (1-1024, default 64)")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// CmdListVSOperatorAuthorizations returns a cobra command for the
// [MOD-DE-QRY-2] ListVSOperatorAuthorizations query.
func CmdListVSOperatorAuthorizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-vs-operator-authorizations",
		Short: "List VS operator authorizations with optional filters",
		Long:  "[MOD-DE-QRY-2] List VS operator authorizations. Optionally filter by corporation and/or vs_operator address.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, _ := cmd.Flags().GetUint64("corporation-id")
			vsOperator, _ := cmd.Flags().GetString("vs-operator")
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListVSOperatorAuthorizations(cmd.Context(), &types.QueryListVSOperatorAuthorizationsRequest{
				CorporationId:   corporationID,
				VsOperator:      vsOperator,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint64("corporation-id", 0, "filter by the corporation id that granted the authorization")
	cmd.Flags().String("vs-operator", "", "filter by the VS operator account")
	cmd.Flags().Uint32("limit", 64, "maximum number of results (1-1024, default 64)")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
