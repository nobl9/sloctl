package internal

import (
	"github.com/pkg/errors"

	"github.com/nobl9/sloctl/internal/flags"
)

var (
	errReplayInvalidOptions = errors.New("you must either run 'sloctl replay' for a single SLO," +
		" providing its name as an argument, or provide configuration file using '-f' flag, but not both")
	errReplayDeleteInvalidOptions = errors.New("you must either run 'sloctl replay delete' for a single " +
		"SLO, providing its name as an argument, or use the '--all' flag to delete all queued replays, but not both")
	errReplayDeleteTooManyArgs = errors.New("'replay delete' command accepts only single SLO name." +
		" If you want to run it for multiple SLOs run it separately for each SLO or if you want to run it for all " +
		"queued SLOs use the '--all' flag.")
	errReplayCancelInvalidOptions = errors.New("you must run 'sloctl replay cancel' for a single " +
		"SLO, providing its name as an argument.")
	errReplayCancelTooManyArgs = errors.New("'replay cancel' command accepts only single SLO name." +
		" If you want to run it for multiple SLOs run it separately for each SLO.")
	errReplayTooManyArgs = errors.New("'replay' command accepts a single SLO name." +
		" If you want to run it for multiple SLOs provide a configuration file instead using '-f' flag")
	errReplayMissingFromArg = errors.Errorf("when running 'sloctl replay' for a single SLO,"+
		" you must provide Replay window start time (%s layout) with '--from' flag", flags.TimeLayoutName)
	errProjectWildcardIsNotAllowed = errors.New(
		"wildcard Project is not allowed, you must provide specific Project name(s)")
)
