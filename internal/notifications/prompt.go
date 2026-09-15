package notifications

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	huh "charm.land/huh/v2"

	"github.com/nobl9/sloctl/internal/huhform"
)

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
	_, _ = fmt.Fprintf(n.stderr, "New sloctl version %s is available!\n\n", release.TagName)
	if highlights := releaseHighlights(release.Body); highlights != "" {
		_, _ = fmt.Fprintf(n.stderr, "%s\n\n", highlights)
	}
	_, _ = fmt.Fprintf(n.stderr, "📜 %s\n\n", release.HTMLURL)
	if !showUpdateForm || !command.available() {
		return updateActionSkip, nil
	}

	options := updateActionOptions(command.display)
	if huhform.AccessibleMode() {
		return n.promptAccessibleUpdate(options)
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
	err := form.Run()
	return action, err
}

func (n notifier) promptAccessibleUpdate(options []huh.Option[updateAction]) (updateAction, error) {
	_, _ = fmt.Fprintln(n.stderr, "Choose update action")
	for i, option := range options {
		_, _ = fmt.Fprintf(n.stderr, "%d. %s\n", i+1, option.Key)
	}
	_, _ = fmt.Fprintf(n.stderr, "Enter a number between 1 and %d [2]: ", len(options))
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
