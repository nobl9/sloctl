package internal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/sloctl/internal/printer"
)

const defaultProject = "default"

var (
	errWrongRenameSyntax = fmt.Errorf(`command "rename-context" requires exactly two arguments with names of your contexts
Example: sloctl config rename-context [oldContext] [newContext]`)

	errWrongDeleteSyntax = fmt.Errorf(`command "delete-context" requires exactly one argument with context name
Example: sloctl config delete-context [contextName]`)
)

type ConfigCmd struct {
	client  *sdk.Client
	config  *sdk.FileConfig
	printer *printer.Printer
	verbose bool
}

func (r *RootCmd) NewConfigCmd() *cobra.Command {
	configCmd := ConfigCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Manage configurations stored in configuration file.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configCmd.client = r.GetClient()
			return configCmd.loadFileConfig(r.Flags.ConfigFile)
		},
	}

	cmd.AddCommand(configCmd.AddContextCommand())
	cmd.AddCommand(configCmd.CurrentContextCommand())
	cmd.AddCommand(configCmd.CurrentUserCommand())
	cmd.AddCommand(configCmd.GetContextsCommand())
	cmd.AddCommand(configCmd.RenameContextCommand())
	cmd.AddCommand(configCmd.DeleteContextCommand())
	cmd.AddCommand(configCmd.SetDefaultContextCommand())

	return cmd
}

func (c *ConfigCmd) loadFileConfig(configFilePath string) error {
	fileConfig := c.client.Config.GetFileConfig()
	if fileConfig.ContextlessConfig == (sdk.ContextlessConfig{}) && len(fileConfig.Contexts) == 0 {
		if configFilePath == "" {
			var err error
			configFilePath, err = sdk.GetDefaultConfigPath()
			if err != nil {
				return err
			}
		}
		if err := fileConfig.Load(configFilePath); err != nil {
			return err
		}
	}
	c.config = &fileConfig
	return nil
}

// AddContextCommand returns cobra command add-context, allows to add context to your configuration file.
func (c *ConfigCmd) AddContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add-context",
		Short: "Add new sloctl configuration context",
		Long:  "Add new sloctl configuration context, an interactive command which collects parameters in wizard mode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if c.config.Contexts == nil {
				c.config.Contexts = make(map[string]sdk.ContextConfig)
			}

			scanner := bufio.NewScanner(os.Stdin)
			newConfigContext, contextName, scanningStop, err := scanContext(c.config, scanner)
			if scanningStop || err != nil {
				return err
			}

			ok, err := scanParams(&newConfigContext, scanner)
			if !ok {
				if err == nil {
					fmt.Println()
				}
				return err
			}

			if c.config.DefaultContext != contextName {
				fmt.Printf("Set \"%s\" as a default context? [y/N]: ", contextName)
				if !scanner.Scan() {
					err = scanner.Err()
					if err == nil {
						fmt.Println()
					}
					return nil
				}
				newContextName := scanDefaultContext(contextName, scanner)
				if newContextName != "" {
					c.config.DefaultContext = newContextName
				}
			}

			c.config.Contexts[contextName] = newConfigContext

			return c.config.Save(c.config.GetPath())
		},
	}
}

func scanContext(fileConfig *sdk.FileConfig, scanner *bufio.Scanner) (
	newConfigContext sdk.ContextConfig,
	contextName string, scanStop bool, err error,
) {
	newConfigContext = sdk.ContextConfig{}
	fmt.Print("New context name: ")
	if !scanner.Scan() {
		err = scanner.Err()
		if err == nil {
			fmt.Println()
		}
		return newConfigContext, "", true, err
	}
	contextName = strings.ToLower(strings.TrimSpace(scanner.Text()))
	isAllowedContextName := regexp.MustCompile(`^[a-zA-Z0-9\-]+$`).MatchString
	if !isAllowedContextName(contextName) {
		return newConfigContext, "", true, errors.New("Enter a valid context name." +
			" Use letters, numbers and `-` characters.")
	}

	if cc, ok := fileConfig.Contexts[contextName]; ok {
		fmt.Printf(
			"Context \"%s\" is already in the configuration file.\nDo you want to overwrite it? [y/N]: ",
			contextName)
		if !scanner.Scan() {
			err = scanner.Err()
			if err == nil {
				fmt.Println()
			}
			return newConfigContext, contextName, true, err
		}
		yesNo := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if yesNo == "y" {
			newConfigContext = cc
			newConfigContext.AccessToken = ""
		} else {
			fmt.Println("Please try to add a new context with a different name.")
			return newConfigContext, contextName, true, nil
		}
	}
	return newConfigContext, contextName, false, nil
}

func scanParams(config *sdk.ContextConfig, scanner *bufio.Scanner) (bool, error) {
	var existingClientID string
	if config.ClientID != "" {
		existingClientID = fmt.Sprintf(" [%s]", credentialPreview(config.ClientID))
	}
	fmt.Printf("Client ID%s: ", existingClientID)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	inputClientID := scanner.Text()
	if inputClientID != "" {
		config.ClientID = inputClientID
	}

	var existingClientSecret string
	if config.ClientSecret != "" {
		existingClientSecret = fmt.Sprintf(" [%s]", credentialPreview(config.ClientSecret))
	}
	fmt.Printf("Client Secret%s: ", existingClientSecret)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	inputClientSecret := scanner.Text()
	if inputClientSecret != "" {
		config.ClientSecret = inputClientSecret
	}

	if config.Project == "" {
		config.Project = defaultProject
	}

	fmt.Printf("Project [%s]: ", config.Project)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	if inputProject := scanner.Text(); inputProject != "" {
		config.Project = inputProject
	}

	return true, nil
}

func scanDefaultContext(contextName string, scanner *bufio.Scanner) string {
	yesNo := scanner.Text()
	yesNo = strings.ToLower(strings.TrimSpace(yesNo))
	if yesNo == "y" {
		return contextName
	}
	return ""
}

// SetDefaultContextCommand return cobra command to set current context in configuration file.
func (c *ConfigCmd) SetDefaultContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context [context name]",
		Short: "Set the default context",
		Long:  "Set a default context in the existing config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			scanner := bufio.NewScanner(os.Stdin)

			if len(c.config.Contexts) == 0 {
				return errors.New("You don't have any contexts in the current configuration file.\n" +
					"Add at least one context in the current configuration file and then set it as the default.\n" +
					"Run \"sloctl config add-context\" or indicate the path to the file using flag \"--config\".")
			}

			var contextName string
			if len(args) > 0 {
				contextName = strings.TrimSpace(args[0])
			} else {
				var names []string
				for existContextName := range c.config.Contexts {
					names = append(names, existContextName)
				}
				fmt.Printf("Select the default context from the existing contexts [%s]: ", strings.Join(names, ", "))
				if !scanner.Scan() {
					return nil
				}
				contextName = scanner.Text()
				contextName = strings.TrimSpace(contextName)
			}

			if _, exist := c.config.Contexts[contextName]; !exist {
				// nolint: revive
				return errors.Errorf(
					"there is no such context: \"%s\", please enter the correct name",
					contextName)
			}
			c.config.DefaultContext = contextName

			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Switched to context \"%s\"\n", contextName)
			return nil
		},
	}
}

func credentialPreview(val string) string {
	const forcedLen = 20
	const disclosedEndingLen = 4
	const anonymousChar = "*"
	const defaultIfEmpty = "None"
	if val == "" {
		return defaultIfEmpty
	}
	return anonymize(val, forcedLen, disclosedEndingLen, anonymousChar)
}

func anonymize(val string, forcedLen, disclosedEndingLen int, anonymousChar string) string {
	if len(val) < disclosedEndingLen {
		disclosedEndingLen = len(val)
	}
	return fmt.Sprintf("%s%s",
		strings.Repeat(anonymousChar, forcedLen-disclosedEndingLen),
		val[len(val)-disclosedEndingLen:])
}

// CurrentContextCommand returns cobra command current-context, prints current used context.
func (c *ConfigCmd) CurrentContextCommand() *cobra.Command {
	currentCtxCmd := &cobra.Command{
		Use:   "current-context",
		Short: "Display current context",
		Long:  "Display configuration for the current context set in the configuration file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if c.verbose {
				currentContext := buildContextString(c.config.DefaultContext,
					c.config.Contexts[c.config.DefaultContext],
					c.verbose)
				fmt.Print(currentContext)
				return nil
			}
			fmt.Println(c.config.DefaultContext)
			return nil
		},
	}

	registerVerboseFlag(currentCtxCmd, &c.verbose)

	return currentCtxCmd
}

func (c *ConfigCmd) CurrentUserCommand() *cobra.Command {
	currentUserCmd := &cobra.Command{
		Use:   "current-user",
		Short: "Display current user details",
		Long:  "Display extended details for the current user, which the access keys are associated with.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			userID, err := c.client.GetUser(ctx)
			if err != nil {
				return err
			}
			user, err := c.client.Users().V2().GetUser(ctx, userID)
			if err != nil {
				return err
			}
			if err = c.printer.Print(user); err != nil {
				return err
			}
			return nil
		},
	}

	c.printer.MustRegisterFlags(currentUserCmd)

	return currentUserCmd
}

// GetContextsCommand returns cobra command to prints all available contexts.
func (c *ConfigCmd) GetContextsCommand() *cobra.Command {
	getContextsCmd := &cobra.Command{
		Use:   "get-contexts",
		Short: "Display all available contexts",
		Long:  "Display all available contexts in the configuration file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var names []string
			for name := range c.config.Contexts {
				names = append(names, name)
			}
			sort.Strings(names)
			var fullConfig string
			if len(args) == 0 && c.verbose {
				for _, name := range names {
					singleConfig := buildContextString(name, c.config.Contexts[name], true)
					fullConfig += singleConfig + "\n"
				}
				fmt.Printf("[%s]\n%s", strings.Join(names, ", "), fullConfig)
				return nil
			}
			for _, name := range args {
				if _, ok := c.config.Contexts[name]; !ok {
					fullConfig += fmt.Sprintf("Missing context: %s\n\n", name)
					continue
				}
				singleConfig := buildContextString(name, c.config.Contexts[name], true)
				fullConfig += singleConfig + "\n"
			}
			fmt.Printf("[%s]\n%s", strings.Join(names, ", "), fullConfig)
			return nil
		},
	}

	registerVerboseFlag(getContextsCmd, &c.verbose)

	return getContextsCmd
}

func buildContextString(name string, config sdk.ContextConfig, verbose bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Context: %s\n", name))
	if !verbose {
		return sb.String()
	}
	configuration := []struct {
		Name  string
		Value string
	}{
		{Name: "client ID", Value: config.ClientID},
		{Name: "client secret", Value: censorField(config.ClientSecret)},
		{Name: "project", Value: config.Project},
		{Name: "url", Value: config.URL},
		{Name: "oktaOrgURL", Value: config.OktaOrgURL},
		{Name: "oktaAuthServer", Value: config.OktaAuthServer},
		{Name: "disable okta", Value: func() string {
			if config.DisableOkta != nil {
				return strconv.FormatBool(*config.DisableOkta)
			}
			return ""
		}()},
		{Name: "timeout", Value: func() string {
			if config.Timeout != nil {
				return config.Timeout.String()
			}
			return ""
		}()},
	}
	for _, field := range configuration {
		if field.Value != "" {
			sb.WriteString(fmt.Sprintf("\t%s: %s\n", field.Name, field.Value))
		}
	}
	return sb.String()
}

func censorField(field string) (censored string) {
	if len(field) > 3 {
		return field[:2] + "***" + field[len(field)-2:]
	} else if len(field) != 0 {
		return generateMissingSecretMessage()
	}
	return censored
}

// RenameContextCommand return cobra command to rename one of contexts in configuration file.
func (c *ConfigCmd) RenameContextCommand() *cobra.Command {
	renameContextCmd := &cobra.Command{
		Use:     "rename-context",
		Short:   "Rename chosen context",
		Long:    "Rename one of the contexts in the configuration file.",
		Example: "	sloctl config rename-context [oldContext] [newContext]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errWrongRenameSyntax
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			oldContext, newContext := args[0], args[1]
			if _, ok := c.config.Contexts[oldContext]; !ok {
				return errors.Errorf("selected context \"%s\" doesn't exists", oldContext)
			}
			if _, ok := c.config.Contexts[newContext]; ok {
				return errors.Errorf("selected context name \"%s\" is already in use", newContext)
			}

			if c.config.DefaultContext == oldContext {
				fmt.Printf("Selected context was set as default. Changing default context to %s.\n", newContext)
				c.config.DefaultContext = newContext
			}

			c.config.Contexts[newContext] = c.config.Contexts[oldContext]
			delete(c.config.Contexts, oldContext)

			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Renaming: \"%s\" to \"%s\"\n", oldContext, newContext)
			return nil
		},
	}

	return renameContextCmd
}

// DeleteContextCommand return cobra command to delete context from configuration file.
func (c *ConfigCmd) DeleteContextCommand() *cobra.Command {
	delContextCmd := &cobra.Command{
		Use:     "delete-context",
		Short:   "Delete chosen context",
		Long:    "Delete one of the contexts in the configuration file.",
		Example: "	sloctl config delete-context [context-name]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errWrongDeleteSyntax
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			toDeleteCtx := args[0]
			if _, ok := c.config.Contexts[toDeleteCtx]; !ok {
				return errors.Errorf("selected context \"%s\" doesn't exists", toDeleteCtx)
			}

			if toDeleteCtx == c.config.DefaultContext {
				return errors.Errorf("cannot remove context currently set as default")
			}

			delete(c.config.Contexts, toDeleteCtx)

			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Context \"%s\" has been deleted.\n", toDeleteCtx)
			return nil
		},
	}

	return delContextCmd
}

func generateMissingSecretMessage() string {
	secretMessages := map[string]struct{}{
		"who needs security anyway?":                                {},
		"this secret could be guessed by any PC in less than 0.01s": {},
		"I know it is easier to remember":                           {},
	}
	for key := range secretMessages {
		return key
	}
	return ""
}
