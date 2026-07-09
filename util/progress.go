/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// isTerminal reports whether stdout is an interactive terminal. When it
// isn't (piped output, a log file, CI), carriage-return animation just
// litters the output with escape codes, so Spinner falls back to plain
// sequential lines instead.
func isTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Spinner renders a single, self-overwriting status line in place of a
// scroll of per-resource log lines, collapsing to one ✔/❌ line once the
// step finishes. On a non-terminal it degrades to plain "label: text"
// lines with no animation or overwriting.
type Spinner struct {
	mu    sync.Mutex
	label string
	text  string
	live  bool
	stop  chan struct{}
	done  chan struct{}
}

// NewSpinner starts a spinner for the given step label (e.g. a manifest
// path). Call Update as work progresses within the step, then Done or
// Fail exactly once to stop it and print the final line.
func NewSpinner(label string) *Spinner {
	s := &Spinner{label: label, live: isTerminal()}
	if !s.live {
		fmt.Printf("→ %s\n", label)
		return s
	}
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	frame := 0
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			fmt.Printf("\r\033[K%s %s %s", spinnerFrames[frame%len(spinnerFrames)], s.label, s.text)
			s.mu.Unlock()
			frame++
		}
	}
}

// Update sets the trailing status text shown next to the spinner, e.g.
// the specific resource currently being applied or removed.
func (s *Spinner) Update(text string) {
	if s == nil {
		return
	}
	if !s.live {
		fmt.Printf("  %s\n", text)
		return
	}
	s.mu.Lock()
	s.text = text
	s.mu.Unlock()
}

func (s *Spinner) stopAnimation() {
	if !s.live {
		return
	}
	close(s.stop)
	<-s.done
	fmt.Print("\r\033[K")
}

// Done stops the spinner and prints a final ✔ line for this step.
func (s *Spinner) Done(msg string) {
	if s == nil {
		return
	}
	s.stopAnimation()
	if msg == "" {
		fmt.Printf("✔ %s\n", s.label)
		return
	}
	fmt.Printf("✔ %s — %s\n", s.label, msg)
}

// Fail stops the spinner and prints a final ❌ line for this step.
func (s *Spinner) Fail(err error) {
	if s == nil {
		return
	}
	s.stopAnimation()
	fmt.Printf("❌ %s: %v\n", s.label, err)
}
