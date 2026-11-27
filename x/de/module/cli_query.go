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
	cmd.AddCommand(CmdGetSponsorship())
	cmd.AddCommand(CmdListSponsorships())

	return cmd
}

// CmdGetDelegateAuthorization returns a cobra command for the
// [MOD-DE-QRY-3] GetDelegateAuthorization query.
func CmdGetDelegateAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-delegate-authorization [id]",
		Short: "Get a delegate authorization by id",
		Long:  "[MOD-DE-QRY-3] Get a single delegate authorization by its id.",
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
			res, err := queryClient.GetDelegateAuthorization(cmd.Context(), &types.QueryGetDelegateAuthorizationRequest{Id: id})
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
		Use:   "list-delegate-authorizations",
		Short: "List delegate authorizations with optional filters",
		Long:  "[MOD-DE-QRY-1] List delegate authorizations. Optionally filter by corporation and/or delegate address.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, _ := cmd.Flags().GetUint64("corporation-id")
			delegate, _ := cmd.Flags().GetString("delegate")
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListDelegateAuthorizations(cmd.Context(), &types.QueryListDelegateAuthorizationsRequest{
				CorporationId:   corporationID,
				Delegate:        delegate,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint64("corporation-id", 0, "filter by the corporation id that granted the authorization")
	cmd.Flags().String("delegate", "", "filter by the delegate account that received the authorization")
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

// CmdGetSponsorship returns a cobra command for the [MOD-DE-QRY-5]
// GetSponsorship query.
func CmdGetSponsorship() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-sponsorship [sponsor-corporation-id] [sponsored-corporation-id]",
		Short: "Get the sponsorship of a sponsor and a sponsored corporation",
		Long:  "[MOD-DE-QRY-5] Get the Sponsorship entry keyed by the sponsor and the sponsored corporation ids.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			sponsor, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sponsor corporation id: %w", err)
			}
			sponsored, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sponsored corporation id: %w", err)
			}
			res, err := types.NewQueryClient(clientCtx).GetSponsorship(cmd.Context(), &types.QueryGetSponsorshipRequest{
				SponsorCorporationId: sponsor, SponsoredCorporationId: sponsored,
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

// CmdListSponsorships returns a cobra command for the [MOD-DE-QRY-6]
// ListSponsorships query.
func CmdListSponsorships() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-sponsorships [sponsor-corporation-id]",
		Short: "List the sponsorships of a sponsor",
		Long:  "[MOD-DE-QRY-6] List the sponsorships granted by a sponsor corporation, each with whether it is active.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			sponsor, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sponsor corporation id: %w", err)
			}
			limit, _ := cmd.Flags().GetUint32("limit")
			res, err := types.NewQueryClient(clientCtx).ListSponsorships(cmd.Context(), &types.QueryListSponsorshipsRequest{
				SponsorCorporationId: sponsor, ResponseMaxSize: limit,
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
