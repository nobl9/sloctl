package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

type Spinner struct {
	bar  *progressbar.ProgressBar
	stop chan struct{}
}

func NewSpinner(description string) *Spinner {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionOnCompletion(func() { _, _ = fmt.Fprint(os.Stderr, "\n") }),
		progressbar.OptionSpinnerType(14))
	return &Spinner{bar: bar, stop: make(chan struct{})}
}

func (s Spinner) Go() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				_ = s.bar.Add(1)
			case <-s.stop:
				return
			}
		}
	}()
}

func (s Spinner) Stop() {
	s.stop <- struct{}{}
	_ = s.bar.Finish()
}
