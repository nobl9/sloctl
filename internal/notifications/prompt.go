package notifications

import (
	_ "embed"
	"fmt"
	"text/template"

	huh "charm.land/huh/v2"

	"github.com/nobl9/sloctl/internal/huhform"
)

//go:embed prompt.tpl
var promptTemplate string

type updateAction string

const (
	updateActionRunUpgrade           updateAction = "run-upgrade"
	updateActionSkip                 updateAction = "skip"
	updateActionSkipUntilNextVersion updateAction = "skip-until-next-version"
)

func (n notifier) promptUpdate(release githubRelease, command updateCommand) (updateAction, error) {
	tpl, err := template.New("notification").Funcs(template.FuncMap{
		"releaseHighlights": releaseHighlights,
	}).Parse(promptTemplate)
	if err != nil {
		return updateActionSkip, fmt.Errorf("parse notification template: %w", err)
	}
	if err := tpl.ExecuteTemplate(n.stderr, "release", release); err != nil {
		return updateActionSkip, fmt.Errorf("render release notice: %w", err)
	}
	if !command.available() {
		return updateActionSkip, nil
	}

	action := updateActionSkip
	form := huhform.New(
		huh.NewGroup(
			huh.NewSelect[updateAction]().
				Title("Choose update action").
				Options(
					huh.NewOption(fmt.Sprintf("Update (runs %s)", command.display), updateActionRunUpgrade),
					huh.NewOption("Skip", updateActionSkip),
					huh.NewOption("Skip until next version", updateActionSkipUntilNextVersion),
				).
				Value(&action),
		),
	).
		WithInput(n.stdin).
		WithOutput(n.stderr)
	err = form.Run()
	return action, err
}
