package internal

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceCommandUses(t *testing.T) {
	root := &RootCmd{}

	t.Run("delete requires at least one name", func(t *testing.T) {
		assertCommandUses(t, root.NewDeleteCmd(), map[string]string{
			"agents":            "agents <name> [name...]",
			"alertmethods":      "alertmethods <name> [name...]",
			"alertpolicies":     "alertpolicies <name> [name...]",
			"alertsilences":     "alertsilences <name> [name...]",
			"annotations":       "annotations <name> [name...]",
			"budgetadjustments": "budgetadjustments <name> [name...]",
			"dataexports":       "dataexports <name> [name...]",
			"directs":           "directs <name> [name...]",
			"projects":          "projects <name> [name...]",
			"reports":           "reports <name> [name...]",
			"rolebindings":      "rolebindings <name> [name...]",
			"services":          "services <name> [name...]",
			"slos":              "slos <name> [name...]",
		})
	})

	t.Run("get accepts optional names", func(t *testing.T) {
		assertCommandUses(t, root.NewGetCmd(), map[string]string{
			"agents":            "agents [name...]",
			"alertmethods":      "alertmethods [name...]",
			"alertpolicies":     "alertpolicies [name...]",
			"alerts":            "alerts [id...]",
			"alertsilences":     "alertsilences [name...]",
			"annotations":       "annotations [name...]",
			"budgetadjustments": "budgetadjustments [name...]",
			"dataexports":       "dataexports [name...]",
			"directs":           "directs [name...]",
			"projects":          "projects [name...]",
			"reports":           "reports [name...]",
			"rolebindings":      "rolebindings [name...]",
			"services":          "services [name...]",
			"slos":              "slos [name...]",
			"user":              "user [id...]",
			"usergroups":        "usergroups [name...]",
		})
	})

	t.Run("edit limits agents to one optional name", func(t *testing.T) {
		assertCommandUses(t, root.NewEditCmd(), map[string]string{
			"agents":            "agents [name]",
			"alertmethods":      "alertmethods [name...]",
			"alertpolicies":     "alertpolicies [name...]",
			"alertsilences":     "alertsilences [name...]",
			"annotations":       "annotations [name...]",
			"budgetadjustments": "budgetadjustments [name...]",
			"dataexports":       "dataexports [name...]",
			"directs":           "directs [name...]",
			"projects":          "projects [name...]",
			"reports":           "reports [name...]",
			"rolebindings":      "rolebindings [name...]",
			"services":          "services [name...]",
			"slos":              "slos [name...]",
		})
	})
}

func TestCommandHelpDescribesOperationalBehavior(t *testing.T) {
	root := &RootCmd{}

	applyCmd := root.NewApplyCmd()
	assert.Equal(t, "Apply Nobl9 resource definitions", applyCmd.Short)
	assert.Contains(t, applyCmd.Long, "Replay is skipped during `--dry-run`")
	assert.Contains(t, applyCmd.Flag("project").Usage, "definitions specifying another project are rejected")
	assert.Contains(t, applyCmd.Example, "--replay --from=")
	assert.Contains(t, applyCmd.Flag("yes").Usage, "filesPromptEnabled and filesPromptThreshold in the [sloctl] section")
	assert.Contains(t, applyCmd.Flag("yes").Usage, "more than 23 files")
	assert.NotContains(t, applyCmd.Flag("yes").Usage, "top-level")

	deleteCmd := root.NewDeleteCmd()
	assert.Equal(t, "Delete Nobl9 resources by name or definition file", deleteCmd.Short)
	assert.Contains(t, deleteCmd.Long, "use a resource subcommand")
	assert.Contains(t, deleteCmd.Example, "delete slos availability latency")

	getCmd := root.NewGetCmd()
	assert.Equal(t, "Get Nobl9 resources", getCmd.Short)
	assert.Contains(t, getCmd.Long, "YAML is the default")
	assert.Contains(t, getCmd.Example, "Read resource names from standard input")

	agentCmd := findSubcommand(t, getCmd, "agents")
	assert.Contains(t, agentCmd.Long, "one additional credential request")
	assert.Equal(t,
		"Include agent client_id and client_secret values. "+
			"This performs one additional credential request per returned agent.",
		agentCmd.Flag("with-keys").Usage,
	)

	alertCmd := findSubcommand(t, getCmd, "alerts")
	assert.Equal(t, "true", alertCmd.Flag("resolved").DefValue)
	assert.Equal(t, "true", alertCmd.Flag("triggered").DefValue)
	assert.Contains(t, alertCmd.Flag("resolved").Usage, "--resolved=false")
	assert.Contains(t, alertCmd.Flag("triggered").Usage, "--triggered=false")
	assert.Contains(t, alertCmd.Long, "same name does not restore its old alerts")
	assert.Contains(t, alertCmd.Long, "spec.conditions[].status.firstMetMetricTime")
	assert.Contains(t, alertCmd.Long, "spec.resolutionReason")
	assert.Contains(t, alertCmd.Long, "Results may be truncated by the API")
	assert.Contains(t, alertCmd.Example, "--resolved=false")
	assert.Contains(t, alertCmd.Example, "--triggered=false")
	assert.NotContains(t, alertCmd.Example, "max 1000")

	annotationCmd := findSubcommand(t, getCmd, "annotations")
	assert.Contains(t, annotationCmd.Long, "`Comment`, `ReviewNote`")
	assert.Contains(t, annotationCmd.Long, "`SloEdit`: created when an SLO definition changes")
	assert.Contains(t, annotationCmd.Long, "https://docs.nobl9.com/features/slo-annotations/#annotation-types")
	assert.Contains(t, annotationCmd.Flag("from").Usage, "spec.startTime is at or after")
	assert.Contains(t, annotationCmd.Flag("to").Usage, "spec.endTime is at or before")
	assert.Contains(t, annotationCmd.Example, "Require startTime at or after --from")
	assert.NotContains(t, annotationCmd.Example, "overlap")

	userCmd := findSubcommand(t, getCmd, "user")
	assert.Contains(t, userCmd.Long, "up to `--limit` users")
	assert.Contains(t, userCmd.Flag("limit").Usage, "default limit applies when no IDs are provided")

	editCmd := root.NewEditCmd()
	assert.Equal(t, "Edit Nobl9 resources in the configured editor", editCmd.Short)
	assert.Contains(t, editCmd.Long, "Removing a resource from the file does not delete it")
	assert.Contains(t, editCmd.Long, "preserves the temporary file")
	assert.Contains(t, editCmd.Example, "--dry-run")

	replayCmd := root.NewReplayCmd()
	assert.Contains(t, replayCmd.Long, "permanent and cannot")
	assert.Contains(t, replayCmd.Long, "several minutes to an hour")
	assert.Contains(t, replayCmd.Long, "does not change the running Replay")

	fullRoot := NewRootCmd()
	assert.Contains(t, fullRoot.Flag("config").Usage, "SLOCTL_CONFIG_FILE_PATH")
	assert.Contains(t, fullRoot.Flag("no-config-file").Usage, "SLOCTL_CLIENT_ID")
	assert.Contains(t, fullRoot.Flag("no-config-file").Usage, "SLOCTL_CLIENT_SECRET")

	eventsCmd := findSubcommand(t, findSubcommand(t, fullRoot, "budgetadjustments"), "events")
	eventsGetCmd := findSubcommand(t, eventsCmd, "get")
	assert.Contains(t, eventsGetCmd.Long, "limited to 250 events")
	assert.Contains(t, eventsGetCmd.Long, "ordered by event start time")
	for _, name := range []string{"update", "delete"} {
		cmd := findSubcommand(t, eventsCmd, name)
		assert.Contains(t, cmd.Long, "30 days")
		assert.Contains(t, cmd.Long, "recalculate affected SLO error budgets")
		assert.Contains(t, cmd.Long, "reliability reports")
	}

	moveSLOCmd := findSubcommand(t, findSubcommand(t, fullRoot, "move"), "slo")
	assert.Contains(t, moveSLOCmd.Long, "**Cross-Project moves:**")
	assert.Contains(t, moveSLOCmd.Long, "without access to the target Project")

	setStatusCmd := findSubcommand(t, findSubcommand(t, fullRoot, "review"), "set-status")
	assert.Contains(t, setStatusCmd.Long, "https://docs.nobl9.com/slo-oversight/reviews/#status-transitions")
	assert.Contains(t, findSubcommand(t, setStatusCmd, "not-started").Long, "has no review schedule")
	assert.Contains(t, findSubcommand(t, setStatusCmd, "reviewed").Long, "whether or not")
	for _, name := range []string{"to-review", "skipped", "overdue"} {
		assert.Contains(t, findSubcommand(t, setStatusCmd, name).Long, "has a review schedule")
	}

	mcpCmd := findSubcommand(t, fullRoot, "mcp")
	assert.Contains(t, mcpCmd.Long, "instead of a direct HTTP connection")
	assert.Contains(t, mcpCmd.Long, "https://docs.nobl9.com/tools-and-utilities/mcp-server")
	assert.NotContains(t, mcpCmd.Long, "experimental")

	awsDirectCmd := findSubcommand(t, findSubcommand(t, fullRoot, "aws-iam-ids"), "direct")
	assert.Contains(t, awsDirectCmd.Long, "`externalID` and `accountID`")
}

func TestPublicCommandsHaveLongDescriptions(t *testing.T) {
	root := NewRootCmd()
	root.InitDefaultCompletionCmd()
	assertCommandLongDescriptions(t, root)
}

func TestGenericResourceCommandsDoNotAddExamples(t *testing.T) {
	root := &RootCmd{}

	for _, parent := range []*cobra.Command{root.NewDeleteCmd(), root.NewEditCmd()} {
		for _, cmd := range parent.Commands() {
			assert.Empty(t, cmd.Example, cmd.CommandPath())
		}
	}

	getCmd := root.NewGetCmd()
	for _, cmd := range getCmd.Commands() {
		if cmd.Name() == "alerts" || cmd.Name() == "annotations" {
			assert.NotEmpty(t, cmd.Example, cmd.CommandPath())
			continue
		}
		assert.Empty(t, cmd.Example, cmd.CommandPath())
	}
}

func TestCompletionHelpUsesMarkdown(t *testing.T) {
	completion := findSubcommand(t, NewRootCmd(), "completion")
	assert.Contains(t, completion.Long, "Bash, fish, PowerShell, or Zsh")

	tests := map[string][]string{
		"bash":       {"~~~bash", "`bash-completion`", "**Load completions in the current shell:**"},
		"fish":       {"~~~fish", "**Load completions in the current shell:**"},
		"powershell": {"~~~powershell", "**Load completions in the current shell:**"},
		"zsh":        {"~~~zsh", "`~/.zshrc`", "**Load completions in the current shell:**"},
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := findSubcommand(t, completion, name)
			for _, fragment := range expected {
				assert.Contains(t, cmd.Long, fragment)
			}
			assert.NotContains(t, cmd.Long, "####")
		})
	}
}

func assertCommandUses(t *testing.T, parent *cobra.Command, expected map[string]string) {
	t.Helper()

	require.Len(t, parent.Commands(), len(expected))
	for name, expectedUse := range expected {
		cmd := findSubcommand(t, parent, name)
		assert.Equal(t, expectedUse, cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
	}
}

func findSubcommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()

	cmd, _, err := parent.Find([]string{name})
	require.NoError(t, err)
	return cmd
}

func assertCommandLongDescriptions(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	assert.NotEmpty(t, cmd.Long, cmd.CommandPath())
	for _, child := range cmd.Commands() {
		if child.IsAvailableCommand() {
			assertCommandLongDescriptions(t, child)
		}
	}
}
