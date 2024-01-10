package sloctl

import (
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/n9/internal/sloctl/printer"
)

type AwsIamIdsCmd struct {
	client          *sdk.Client
	fieldSeparator  string
	recordSeparator string
	outputFormat    string
	resourceName    string
}

func (r *RootCmd) NewAwsIamIds() *cobra.Command {
	awsIamIds := &AwsIamIdsCmd{}

	cobraCmd := &cobra.Command{
		Use:   "aws-iam-ids",
		Short: "Returns IAM IDs used in AWS integrations",
	}

	direct := &cobra.Command{
		Use:   "direct [direct-name]",
		Short: "Returns external ID and AWS account ID for given direct name",
		Long: "Returns external ID and AWS account ID that can be used to create cross-account IAM roles." +
			"\nMore details available at: https://docs.nobl9.com/Sources/Amazon_CloudWatch/#cross-account-iam-roles-new.",
		Args:             awsIamIds.arguments,
		PersistentPreRun: func(iamIdsCmd *cobra.Command, args []string) { awsIamIds.client = r.GetClient() },
		RunE:             func(iamIdsCmd *cobra.Command, args []string) error { return awsIamIds.Direct(iamIdsCmd) },
	}
	RegisterOutputFormatFlags(direct, &awsIamIds.outputFormat, &awsIamIds.fieldSeparator, &awsIamIds.recordSeparator)
	cobraCmd.AddCommand(direct)

	dataExport := &cobra.Command{
		Use: "dataexport",
		Short: "Returns AWS external ID, which will be used by Nobl9 to assume the IAM role when" +
			" performing data export",
		PersistentPreRun: func(iamIdsCmd *cobra.Command, args []string) { awsIamIds.client = r.GetClient() },
		RunE:             func(iamIdsCmd *cobra.Command, args []string) error { return awsIamIds.DataExport(iamIdsCmd) },
	}
	cobraCmd.AddCommand(dataExport)

	return cobraCmd
}

func (a *AwsIamIdsCmd) arguments(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		_ = cmd.Usage()
		if len(args) == 0 {
			return errors.New("Direct name must be provided")
		}
		return errors.New("command expects a single argument, Direct name")
	}
	a.resourceName = args[0]
	return nil
}

func (a *AwsIamIdsCmd) Direct(cmd *cobra.Command) error {
	ctx := cmd.Context()
	response, err := a.client.AuthData().V1().GetDirectIAMRoleIDs(ctx, a.client.Config.Project, a.resourceName)
	if err != nil {
		return errors.Wrap(err, "unable to get AWS IAM role auth external IDs")
	}

	p, err := printer.New(cmd.OutOrStdout(), printer.Format(a.outputFormat), a.fieldSeparator, a.recordSeparator)
	if err != nil {
		return err
	}
	if err = p.Print(response); err != nil {
		return err
	}
	return nil
}

func (a *AwsIamIdsCmd) DataExport(cmd *cobra.Command) error {
	ctx := cmd.Context()
	response, err := a.client.AuthData().V1().GetDataExportIAMRoleIDs(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to get AWS external ID")
	}

	p, err := printer.New(cmd.OutOrStdout(), printer.Format(a.outputFormat), a.fieldSeparator, a.recordSeparator)
	if err != nil {
		return err
	}
	if err = p.Print(response.ExternalID); err != nil {
		return err
	}
	return nil
}
