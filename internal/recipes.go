package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/printer"
)

type Recipes map[string]Recipe

type Recipe struct {
	Args        []string          `json:"args"`
	Description string            `json:"description"`
	Example     string            `json:"example,omitempty"`
	JQ          string            `json:"jq,omitempty"`
	Validators  *RecipeValidators `json:"validators,omitempty"`
	name        string
}

type RecipeValidators struct {
	AtLeastArgs []string `json:"atLeastArgs,omitempty"`
}

func (r *RootCmd) NewRecipesCmd() *cobra.Command {
	recipes, recipesErr := readRecipes()

	cmd := &cobra.Command{
		Use:   "recipes",
		Short: "Run custom recipes",
	}
	if recipesErr != nil {
		cmd.RunE = func(*cobra.Command, []string) error { return recipesErr }
		return cmd
	}

	for name, recipe := range recipes {
		recipe.name = name
		printer := printer.NewPrinter(printer.Config{})
		var showDefinition bool
		recipeCmd := &cobra.Command{
			Use:     name,
			Short:   recipe.Description,
			Example: recipe.Example,
			Args:    recipesArgFunc(recipe),
			RunE: func(*cobra.Command, []string) error {
				if showDefinition {
					return printer.Print(Recipes{name: recipe})
				}
				return runRecipe(recipe)
			},
		}
		printer.MustRegisterFlags(recipeCmd)
		recipeCmd.Flags().BoolVarP(&showDefinition, "show-definition", "d", false, "Display recipe definition")
		cmd.AddCommand(recipeCmd)
	}

	return cmd
}

func runRecipe(recipe Recipe) error {
	args := recipe.Args
	if len(args) == 0 {
		return errors.New("empty arguments list for recipe")
	}
	// Program name is always the first argument when invoking sloctl cli.
	if args[0] != "sloctl" {
		args = slices.Insert(args, 0, "sloctl")
	}
	// 'sloctl', 'recipes', '<recipe_name>' == 3
	if len(os.Args) > 3 {
		args = append(args, os.Args[3:]...)
	}
	recipe.Args = args
	if err := recipeValidation.Validate(recipe); err != nil {
		return err
	}
	if len(recipe.JQ) > 0 {
		args = append(args, "--jq", recipe.JQ)
	}
	os.Args = args
	fmt.Fprintf(os.Stderr, "Running: %s\n", strings.Join(args, " "))
	return Execute()
}

func readRecipes() (Recipes, error) {
	configPath := os.Getenv("SLOCTL_RECIPES_PATH")
	if configPath == "" {
		defaultFilename := "sloctl-recipes.yaml"
		sdkConfigPath, err := sdk.GetDefaultConfigPath()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read default Nobl9 SDK config path")
		}
		configPath = filepath.Join(filepath.Dir(sdkConfigPath), defaultFilename)
	}
	data, err := os.ReadFile(configPath) // #nosec: G304
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read sloctl recipes located at: '%s'", configPath)
	}
	var recipes Recipes
	if err = yaml.Unmarshal(data, &recipes); err != nil {
		return nil, errors.Wrapf(err, "failed to decode sloctl recipes config located at: '%s'", configPath)
	}
	return recipes, nil
}

func recipesArgFunc(recipe Recipe) cobra.PositionalArgs {
	requiredArgs := recipe.Validators.AtLeastArgs
	if len(requiredArgs) == 0 {
		return nil
	}
	return func(_ *cobra.Command, args []string) error {
		if len(args) != len(requiredArgs) {
			return errors.Errorf("Expected at least %d arg(s), received %d, required arg(s): %v",
				len(requiredArgs), len(args), requiredArgs)
		}
		return nil
	}
}

var recipeValidation = govy.New(
	govy.For(govy.GetSelf[Recipe]()).
		Rules(govy.NewRule(func(r Recipe) error {
			if r.JQ != "" && (slices.Contains(r.Args, "--jq") || slices.Contains(r.Args, "-q")) {
				return errors.Errorf("jq expression cannot be defined both in 'jq' property and provided with 'args'")
			}
			return nil
		})),
	govy.ForSlice(func(r Recipe) []string { return r.Args }).
		WithName("cmd").
		Rules(rules.SliceMinLength[[]string](1)),
	govy.For(func(r Recipe) string { return r.Description }).
		WithName("description").
		Required(),
	govy.For(func(r Recipe) string { return r.JQ }).
		WithName("expr"),
).
	WithNameFunc(func(s Recipe) string { return "'" + s.name + "' recipe" })
