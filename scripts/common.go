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

// Package scripts holds the environment setup/teardown routines that used to
// ship as standalone shell (and later standalone `package main`) scripts. They
// are exported functions here so the cmd package can call them directly instead
// of shelling out to a separate binary. The helpers below are the shared
// equivalents of the old scripts' inline bash plumbing.
package scripts

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getenv returns the value of key, or def when it is unset or empty.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// requireBinary returns an error if name isn't on PATH. The standalone scripts
// used to os.Exit here; as a library we surface the failure to the caller.
func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required but not found", name)
	}
	return nil
}

// run streams stdout/stderr straight through, equivalent to letting the old
// bash script's commands print directly to the terminal.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runQuiet captures output instead of streaming it, for the commands the
// original scripts silenced with `>/dev/null`. On failure the captured output
// is folded into the returned error.
func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\n%s", err, buf.String())
	}
	return nil
}

// capture runs a command and returns its stdout. On failure the returned error
// includes stderr, matching the old scripts' behaviour of surfacing the
// command's own diagnostics.
func capture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%w\n%s", err, errBuf.String())
	}
	return out.String(), nil
}

// runStdin feeds stdin to a command while streaming its stdout/stderr.
func runStdin(stdin []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// normalizeHelmVersion trims a Helm version and treats "latest" (any case) as
// empty so callers fall back to the chart's newest release instead of asking
// Helm for a literal "latest" version.
func normalizeHelmVersion(version string) string {
	version = strings.TrimSpace(version)
	if strings.EqualFold(version, "latest") {
		return ""
	}
	return version
}
