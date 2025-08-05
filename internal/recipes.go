package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/nobl9/sloctl/internal/printer"
	"github.com/nobl9/sloctl/internal/yamlenc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Recipes map[string]Recipe

type Recipe struct {
	Args        []string         `json:"args"`
	Description string           `json:"description"`
	Example     string           `json:"example,omitempty"`
	JQ          string           `json:"jq,omitempty"`
	Validators  RecipeValidators `json:"validators,omitempty"`
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

	printer := printer.NewPrinter(printer.Config{})
	configGroup := &cobra.Group{
		ID:    "config",
		Title: "Config commands:",
	}
	addCommand := &cobra.Command{
		GroupID: configGroup.ID,
		Use:     "add",
		Short:   "Add new recipe",
		Long:    "Provide a ",
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	}
	listCommand := &cobra.Command{
		GroupID: configGroup.ID,
		Use:     "list",
		Short:   "List existing recipes",
		RunE: func(*cobra.Command, []string) error {
			return printer.Print(recipes)
		},
	}
	printer.MustRegisterFlags(listCommand)
	removeCommand := &cobra.Command{
		GroupID: configGroup.ID,
		Use:     "remove",
		Short:   "Remove recipe by name",
		Args:    recipesArgFunc([]string{"name"}),
		RunE: func(_ *cobra.Command, args []string) error {
			for _, name := range args {
				delete(recipes, name)
			}
			return saveRecipes(recipes)
		},
	}
	cmd.AddCommand(addCommand, listCommand, removeCommand)

	recipesGroup := &cobra.Group{
		ID:    "recipes",
		Title: "Recipes:",
	}
	for name, recipe := range recipes {
		recipe.name = name
		recipeCmd := &cobra.Command{
			GroupID: recipesGroup.ID,
			Use:     name,
			Short:   recipe.Description,
			Example: recipe.Example,
			Args:    recipesArgFunc(recipe.Validators.AtLeastArgs),
			RunE: func(*cobra.Command, []string) error {
				return runRecipe(recipe)
			},
		}
		cmd.AddCommand(recipeCmd)
	}

	cmd.AddGroup(configGroup, recipesGroup)

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
	configPath, err := getRecipesConfigPath()
	if err != nil {
		return nil, err
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

func saveRecipes(recipes Recipes) error {
	configPath, err := getRecipesConfigPath()
	if err != nil {
		return err
	}
	tmpFileName, err := writeRecipesToTempFile(configPath, recipes)
	if err != nil {
		return errors.Wrapf(err, "failed to create and write a temporary recipes file used for saving the changes")
	}
	if err = os.Rename(tmpFileName, configPath); err != nil {
		return err
	}
	return nil
}

type fileEncoder interface {
	Encode(v any) error
}

func writeRecipesToTempFile(path string, recipes Recipes) (tmpFileName string, err error) {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return "", err
	}
	defer func() { _ = tmpFile.Close() }()
	var enc fileEncoder
	switch filepath.Ext(path) {
	case ".json":
		enc = json.NewEncoder(tmpFile)
	default:
		enc = yamlenc.NewEncoder(tmpFile)
	}
	if err = enc.Encode(recipes); err != nil {
		return "", err
	}
	if err = tmpFile.Sync(); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

func getRecipesConfigPath() (string, error) {
	configPath := os.Getenv("SLOCTL_RECIPES_PATH")
	if configPath == "" {
		defaultFilename := "sloctl-recipes.yaml"
		sdkConfigPath, err := sdk.GetDefaultConfigPath()
		if err != nil {
			return "", errors.Wrapf(err, "failed to read default Nobl9 SDK config path")
		}
		configPath = filepath.Join(filepath.Dir(sdkConfigPath), defaultFilename)
	}
	return configPath, nil
}

func recipesArgFunc(requiredArgs []string) cobra.PositionalArgs {
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
