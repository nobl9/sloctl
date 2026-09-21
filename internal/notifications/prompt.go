package notifications

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
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

func (n notifier) promptUpdate(
	release githubRelease,
	command updateCommand,
	showUpdateForm bool,
) (updateAction, error) {
	tpl, err := template.New("notification").Funcs(template.FuncMap{
		"releaseHighlights": releaseHighlights,
		"inc":               func(i int) int { return i + 1 },
	}).Parse(promptTemplate)
	if err != nil {
		return updateActionSkip, fmt.Errorf("parse notification template: %w", err)
	}
	if err := tpl.ExecuteTemplate(n.stderr, "release", release); err != nil {
		return updateActionSkip, fmt.Errorf("render release notice: %w", err)
	}
	if !showUpdateForm || !command.available() {
		return updateActionSkip, nil
	}

	options := updateActionOptions(command.display)
	if huhform.AccessibleMode() {
		return n.promptAccessibleUpdate(tpl, options)
	}
	action := updateActionSkip
	form := huhform.New(
		huh.NewGroup(
			huh.NewSelect[updateAction]().
				Title("Choose update action").
				Options(options...).
				Value(&action),
		),
	).
		WithInput(n.stdin).
		WithOutput(n.stderr)
	err = form.Run()
	return action, err
}

func (n notifier) promptAccessibleUpdate(
	tpl *template.Template,
	options []huh.Option[updateAction],
) (updateAction, error) {
	if err := tpl.ExecuteTemplate(n.stderr, "actions", options); err != nil {
		return updateActionSkip, fmt.Errorf("render update choices: %w", err)
	}
	// Huh's accessible selector treats EOF as a choice instead of a read failure.
	input, err := bufio.NewReader(n.stdin).ReadString('\n')
	_, _ = fmt.Fprintln(n.stderr)
	if err != nil {
		return updateActionSkip, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return updateActionSkip, nil
	}
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return updateActionSkip, fmt.Errorf("choose a number between 1 and %d", len(options))
	}
	return options[choice-1].Value, nil
}

func updateActionOptions(updateCommand string) []huh.Option[updateAction] {
	return []huh.Option[updateAction]{
		huh.NewOption(fmt.Sprintf("Update (runs %s)", updateCommand), updateActionRunUpgrade),
		huh.NewOption("Skip", updateActionSkip),
		huh.NewOption("Skip until next version", updateActionSkipUntilNextVersion),
	}
}
