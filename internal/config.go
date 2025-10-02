package internal

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
	"github.com/nobl9/sloctl/internal/form"
	"github.com/nobl9/sloctl/internal/printer"
)

// runHuhFormFunc is a global variable so that the unit test can inject their logic inside.
var runHuhFormFunc = func(ctx context.Context, form *huh.Form) error {
	return form.RunWithContext(ctx)
}

type clientGetter interface {
	GetClient() *sdk.Client
}

type ConfigCmd struct {
	clientGetter clientGetter
	config       *sdk.FileConfig
	printer      *printer.Printer
	verbose      bool
}

func (r *RootCmd) NewConfigCmd() *cobra.Command {
	configCmd := &ConfigCmd{
		clientGetter: r,
		printer: printer.NewPrinter(printer.Config{SupportedFromats: []printer.Format{
			printer.TOMLFormat,
			printer.YAMLFormat,
			printer.JSONFormat,
			printer.CSVFormat,
		}}),
	}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Manage configurations stored in configuration file.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configCmd.loadFileConfig(r.Flags.ConfigFile)
		},
	}

	cmd.AddCommand(configCmd.AddContextCommand())
	cmd.AddCommand(configCmd.CurrentContextCommand())
	cmd.AddCommand(configCmd.CurrentUserCommand())
	cmd.AddCommand(configCmd.GetContextsCommand())
	cmd.AddCommand(configCmd.RenameContextCommand())
	cmd.AddCommand(configCmd.DeleteContextCommand())
	cmd.AddCommand(configCmd.UseContextCommand())

	return cmd
}

func (c *ConfigCmd) AddContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add-context",
		Short: "Add new sloctl configuration context",
		Long:  "Add new sloctl configuration context, an interactive command which collects parameters in wizard mode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				contextName      string
				setAsDefault     bool
				platformInstance = sdk.PlatformInstanceDefault
				config           = new(sdk.ContextConfig)
				orgURL           *url.URL
				authConfig       sdk.PlatformInstanceAuthConfig
			)
			config.Project = "default"
			authInstances := sdk.GetPlatformInstanceAuthConfigs()

			form := form.New(
				huh.NewGroup(
					huh.NewInput().
						Title("Provide context name").
						Value(&contextName).
						Validate(validateMultipleHuhValues(validateStringNotEmpty, c.validateContextIsInUse)),
					huh.NewInput().
						Title("Provide client ID").
						Value(&config.ClientID).
						Validate(validateStringNotEmpty),
					huh.NewInput().
						Title("Provide client secret").
						Value(&config.ClientSecret).
						EchoMode(huh.EchoModePassword).
						Validate(validateStringNotEmpty),
					huh.NewSelect[sdk.PlatformInstance]().
						Title("Select Nobl9 instance").
						Options(huh.NewOptions(sdk.GetPlatformInstances()...)...).
						Validate(func(pi sdk.PlatformInstance) error {
							if pi == sdk.PlatformInstanceCustom {
								return nil
							}
							authConfig = authInstances[pi]
							return nil
						}).
						Value(&platformInstance),
				),
				huh.NewGroup(
					huh.NewInput().
						Title("Set organization url").
						Description("Example: "+authInstances[sdk.PlatformInstanceDefault].URL.String()).
						// Value(&config.OktaOrgURL). -- Value is set in Validate below.
						Validate(validateMultipleHuhValues(validateStringNotEmpty, func(s string) error {
							var err error
							orgURL, err = url.Parse(s)
							if err != nil {
								return err
							}
							authConfig = authInstances[platformInstance]
							authConfig.URL = orgURL
							return nil
						})),
					huh.NewInput().
						Title("Set auth server id").
						Description("Example: "+authInstances[sdk.PlatformInstanceDefault].AuthServer).
						Value(&authConfig.AuthServer).
						Validate(validateStringNotEmpty),
				).
					WithHideFunc(func() bool { return platformInstance != sdk.PlatformInstanceCustom }),
				huh.NewGroup(
					huh.NewInput().
						Title("Provide default project").
						Value(&config.Project).
						Validate(validateStringNotEmpty),
					huh.NewConfirm().
						Title("Set context as default?").
						Value(&setAsDefault),
				),
			)

			if err := runHuhFormFunc(cmd.Context(), form); err != nil {
				return errors.Wrap(err, "failed to run context addition form")
			}

			switch platformInstance {
			case sdk.PlatformInstanceDefault:
			// Do nothing, auth config defaults are not required to be set in config.toml.
			default:
				config.OktaOrgURL = authConfig.URL.String()
				config.OktaAuthServer = authConfig.AuthServer
			}
			c.config.Contexts[contextName] = *config
			if setAsDefault {
				c.config.DefaultContext = contextName
			}
			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Added context %q.\n", contextName)
			return nil
		},
	}
}

func (c *ConfigCmd) UseContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context [context name]",
		Short: "Set the default context",
		Long:  "Set a default context in the existing config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(c.config.Contexts) == 0 {
				return errors.New("You don't have any contexts in the current configuration file.\n" +
					"Add at least one context in the current configuration file and then set it as the default.\n" +
					"Run \"sloctl config add-context\" or indicate the path to the file using \"--config\" flag.")
			}

			var contextName string
			switch {
			case len(args) == 0:
				form := form.New(huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select the new context:").
						Options(huh.NewOptions(c.getContextNames()...)...).
						Value(&contextName),
				))
				if err := form.RunWithContext(cmd.Context()); err != nil {
					return errors.Wrap(err, "failed to run context selection prompt")
				}
			default:
				contextName = strings.TrimSpace(args[0])
			}

			if err := c.validateContextExists(contextName); err != nil {
				return err
			}

			c.config.DefaultContext = contextName
			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Switched to context \"%s\".\n", contextName)
			return nil
		},
	}
}

func (c *ConfigCmd) CurrentContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-context",
		Short: "Display current context",
		Long:  "Display configuration for the current context set in the configuration file.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return requireFlagsIfFlagIsSet(
				cmd,
				flagVerbose,
				printer.OutputFlagName,
				csv.RecordSeparatorFlag,
				csv.FieldSeparatorFlag,
			)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if c.verbose {
				conf, err := c.config.GetCurrentContextConfig()
				if err != nil {
					return err
				}
				conf = sanitizeContextConfig(conf)
				return c.printer.Print(conf)
			}
			fmt.Println(c.config.DefaultContext)
			return nil
		},
	}

	registerVerboseFlag(cmd, &c.verbose)
	c.printer.MustRegisterFlags(cmd)
	return cmd
}

func (c *ConfigCmd) CurrentUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-user",
		Short: "Display current user details",
		Long:  "Display extended details for the user associated with the current context's access key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := c.clientGetter.GetClient()
			ctx := cmd.Context()
			userID, err := client.GetUser(ctx)
			if err != nil {
				return err
			}
			user, err := client.Users().V2().GetUser(ctx, userID)
			if err != nil {
				return err
			}
			if err = c.printer.Print(user); err != nil {
				return err
			}
			return nil
		},
	}

	c.printer.MustRegisterFlags(cmd)
	return cmd
}

func (c *ConfigCmd) GetContextsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-contexts",
		Short: "Display all available contexts",
		Long:  "Display all available contexts in the configuration file.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return requireFlagsIfFlagIsSet(
				cmd,
				flagVerbose,
				printer.OutputFlagName,
				csv.RecordSeparatorFlag,
				csv.FieldSeparatorFlag,
			)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var names []string
			switch len(args) {
			case 0:
				names = c.getContextNames()
			default:
				names = args
			}
			for _, name := range names {
				if err := c.validateContextExists(name); err != nil {
					return err
				}
			}

			switch c.verbose {
			case true:
				configs := make(map[string]sdk.ContextConfig, len(names))
				for _, name := range names {
					configs[name] = sanitizeContextConfig(c.config.Contexts[name])
				}
				return c.printer.Print(configs)
			default:
				slices.Sort(names)
				fmt.Println(strings.Join(names, "\n"))
				return nil
			}
		},
	}

	registerVerboseFlag(cmd, &c.verbose)
	c.printer.MustRegisterFlags(cmd)
	return cmd
}

func (c *ConfigCmd) RenameContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename-context",
		Short: "Rename chosen context",
		Long: "Rename one of the contexts in the configuration file.\n" +
			"If no arguments are provided, the command displays an interactive prompt.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				oldContext string
				newContext string
			)
			switch len(args) {
			case 0:
				form := form.New(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select context to rename").
							Options(huh.NewOptions(c.getContextNames()...)...).
							Value(&oldContext),
						huh.NewInput().
							Title("New context name").
							Value(&newContext).
							Validate(validateMultipleHuhValues(validateStringNotEmpty, c.validateContextIsInUse)),
					),
				)
				if err := form.RunWithContext(cmd.Context()); err != nil {
					return errors.Wrap(err, "failed to run context rename form")
				}
			case 2:
				oldContext, newContext = args[0], args[1]
			default:
				return errors.Errorf(
					"either provide new and old context names or no arguments at all, received %d arguments",
					len(args))
			}

			if err := validateStringNotEmpty(newContext); err != nil {
				return errors.Errorf("new context cannot be empty")
			}
			if err := c.validateContextExists(oldContext); err != nil {
				return err
			}
			if err := c.validateContextIsInUse(newContext); err != nil {
				return err
			}

			if c.config.DefaultContext == oldContext {
				fmt.Printf("Renamed context was set as default. Changing default context to %q.\n", newContext)
				c.config.DefaultContext = newContext
			}

			c.config.Contexts[newContext] = c.config.Contexts[oldContext]
			delete(c.config.Contexts, oldContext)

			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}
			fmt.Printf("Renamed context %q to %q.\n", oldContext, newContext)
			return nil
		},
	}

	return cmd
}

func (c *ConfigCmd) DeleteContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-context",
		Short: "Delete chosen context(s)",
		Long: "Delete one or more of the contexts from the configuration file.\n" +
			"Each argument is treated as a context name, " +
			"when no arguments are provided a multi-selection prompt is desplayed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var contextNames []string
			switch len(args) {
			case 0:
				contexts := c.getContextNames()
				contexts = slices.DeleteFunc(contexts, func(name string) bool { return name == c.config.DefaultContext })
				form := form.New(huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Select context(s) for deletion").
						Options(huh.NewOptions(contexts...)...).
						Value(&contextNames),
				))
				if err := form.RunWithContext(cmd.Context()); err != nil {
					return errors.Wrap(err, "failed to run context selection prompt")
				}
			default:
				for _, arg := range args {
					contextNames = append(contextNames, strings.TrimSpace(arg))
				}
			}

			for _, name := range contextNames {
				if err := c.validateContextExists(name); err != nil {
					return err
				}
				if name == c.config.DefaultContext {
					return errors.Errorf("cannot remove context currently set as default")
				}
				delete(c.config.Contexts, name)
			}

			if err := c.config.Save(c.config.GetPath()); err != nil {
				return err
			}

			// Wrap in quotes for presentation.
			for i, name := range contextNames {
				contextNames[i] = "\"" + name + "\""
			}
			switch len(contextNames) {
			case 0:
				fmt.Println("No contexts provided for deletion.")
			case 1:
				fmt.Printf("Context %s has been deleted.\n", contextNames[0])
			default:
				fmt.Printf("Contexts %s have been deleted.\n", strings.Join(contextNames, ", "))
			}
			return nil
		},
	}

	return cmd
}

func (c *ConfigCmd) loadFileConfig(configPath string) error {
	if configPath == "" {
		if v, ok := os.LookupEnv(sdkEnvPrefix + "CONFIG_FILE_PATH"); ok {
			configPath = strings.TrimSpace(v)
		} else {
			var err error
			configPath, err = sdk.GetDefaultConfigPath()
			if err != nil {
				return err
			}
		}
	}
	c.config = new(sdk.FileConfig)
	return c.config.Load(configPath)
}

func (c *ConfigCmd) validateContextIsInUse(name string) error {
	if _, exist := c.config.Contexts[name]; exist {
		return errors.Errorf("selected context name %q is already in use", name)
	}
	return nil
}

func (c *ConfigCmd) validateContextExists(name string) error {
	if _, exist := c.config.Contexts[name]; !exist {
		return errors.Errorf("selected context %q does not exists", name)
	}
	return nil
}

func (c *ConfigCmd) getContextNames() []string {
	return slices.Sorted(maps.Keys(c.config.Contexts))
}

func sanitizeContextConfig(conf sdk.ContextConfig) sdk.ContextConfig {
	conf.ClientSecret = censorField(conf.ClientSecret)
	conf.AccessToken = ""
	return conf
}

func censorField(field string) (censored string) {
	if len(field) > 3 {
		return field[:2] + "***" + field[len(field)-2:]
	} else if len(field) != 0 {
		return generateMissingSecretMessage()
	}
	return censored
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

type huhValidationFunc[T any] func(v T) error

func validateMultipleHuhValues[T any](funcs ...huhValidationFunc[T]) huhValidationFunc[T] {
	return func(v T) error {
		var err error
		for _, f := range funcs {
			if err = f(v); err != nil {
				return err
			}
		}
		return nil
	}
}

func validateStringNotEmpty(s string) error {
	s = strings.TrimSpace(s)
	if err := huh.ValidateMinLength(1)(s); err != nil {
		return fmt.Errorf("input cannot be empty")
	}
	return nil
}
