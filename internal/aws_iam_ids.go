package internal

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/sloctl/internal/printer"
)

type AwsIamIdsCmd struct {
	client       *sdk.Client
	printer      *printer.Printer
	resourceName string
}

func (r *RootCmd) NewAwsIamIds() *cobra.Command {
	awsIamIds := &AwsIamIdsCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}

	cobraCmd := &cobra.Command{
		Use:   "aws-iam-ids",
		Short: "Get AWS IAM role identifiers",
		Long:  "Get the AWS identifiers used to configure IAM roles for Direct data sources and data exports.",
	}
	awsIamIds.printer.MustRegisterFlags(cobraCmd)

	directCmd := &cobra.Command{
		Use:   "direct <direct-name>",
		Short: "Get IAM identifiers for a Direct data source",
		Long: "Return the AWS external ID and Nobl9 AWS account ID for the named\n" +
			"Direct data source in the active Project.\n" +
			"Use these values to configure a cross-account IAM role.\n" +
			"The response contains the `externalID` and `accountID` fields.",
		Example: `sloctl aws-iam-ids direct my-cloudwatch-source
sloctl aws-iam-ids direct my-cloudwatch-source --output json`,
		Args:             awsIamIds.arguments,
		PersistentPreRun: func(iamIdsCmd *cobra.Command, args []string) { awsIamIds.client = r.GetClient() },
		RunE:             func(iamIdsCmd *cobra.Command, args []string) error { return awsIamIds.Direct(iamIdsCmd) },
	}
	cobraCmd.AddCommand(directCmd)

	dataExportCmd := &cobra.Command{
		Use:   "dataexport",
		Short: "Get the AWS external ID for data exports",
		Long:  "Return the AWS external ID that Nobl9 uses to assume a data export IAM role.",
		Example: "sloctl aws-iam-ids dataexport\n" +
			"sloctl aws-iam-ids dataexport --output json",
		PersistentPreRun: func(iamIdsCmd *cobra.Command, args []string) { awsIamIds.client = r.GetClient() },
		RunE:             func(iamIdsCmd *cobra.Command, args []string) error { return awsIamIds.DataExport(iamIdsCmd) },
	}
	cobraCmd.AddCommand(dataExportCmd)

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
	return a.printer.Print(response)
}

func (a *AwsIamIdsCmd) DataExport(cmd *cobra.Command) error {
	ctx := cmd.Context()
	response, err := a.client.AuthData().V1().GetDataExportIAMRoleIDs(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to get AWS external ID")
	}
	return a.printer.Print(response)
}
