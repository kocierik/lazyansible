// Package galaxy wraps the ansible-galaxy CLI.
package galaxy

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// Item represents a role or collection returned by list/search.
type Item struct {
	Name        string
	Version     string
	Description string
}

// CheckBinary returns an error if ansible-galaxy is not in $PATH.
func CheckBinary() error {
	if _, err := exec.LookPath("ansible-galaxy"); err != nil {
		return errors.New("ansible-galaxy not found in PATH")
	}
	return nil
}

// runStdout runs a command and returns only its stdout.
// Stderr is captured separately so warnings never pollute the parsed output.
// Returns (stdout, stderr, error).
func runStdout(args ...string) ([]byte, string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.Bytes(), errBuf.String(), err
}

// isBenignError reports whether the error (and the stderr text) should be
// treated as "nothing installed" rather than a hard failure.
// ansible-galaxy exits with non-zero codes and prints "None of the provided
// paths were usable" when the roles/collections directory doesn't exist yet.
func isBenignError(err error, stderr string) bool {
	if err == nil {
		return true
	}
	if ee, ok := err.(*exec.ExitError); ok {
		code := ee.ExitCode()
		if code == 5 || code == 6 {
			return true
		}
	}
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "none of the provided paths") ||
		strings.Contains(lower, "no roles found") {
		return true
	}
	return false
}

// ListRoles returns installed roles (ansible-galaxy role list).
// Warnings printed to stderr are discarded; only stdout is parsed.
// A benign exit (empty roles path, exit 5/6) returns an empty list, not an error.
func ListRoles() ([]Item, error) {
	out, stderr, err := runStdout("ansible-galaxy", "role", "list")
	if err != nil && !isBenignError(err, stderr) {
		return nil, err
	}
	return parseRoleList(out), nil
}

func parseRoleList(data []byte) []Item {
	var items []Item
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		trimmed := strings.TrimSpace(sc.Text())
		// Skip blank lines and path comment headers ("# /home/…/roles").
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Role entries are formatted as:  "- namespace.role_name, 1.2.3"
		entry := strings.TrimPrefix(trimmed, "- ")
		parts := strings.SplitN(entry, ",", 2)
		item := Item{Name: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			item.Version = strings.TrimSpace(parts[1])
		}
		if item.Name != "" {
			items = append(items, item)
		}
	}
	return items
}

// ListCollections returns installed collections (ansible-galaxy collection list).
// Stderr warnings are discarded; only stdout is parsed.
func ListCollections() ([]Item, error) {
	out, stderr, err := runStdout("ansible-galaxy", "collection", "list")
	if err != nil && !isBenignError(err, stderr) {
		return nil, err
	}
	return parseCollectionList(out), nil
}

func parseCollectionList(data []byte) []Item {
	var items []Item
	sc := bufio.NewScanner(bytes.NewReader(data))
	inTable := false
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			inTable = false
			continue
		}
		// Header line starts with "# /path..."
		if strings.HasPrefix(trimmed, "#") {
			inTable = true
			continue
		}
		// Skip dashed separator lines and the "Collection Version" header.
		if strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "Collection") {
			continue
		}
		if !inTable {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 {
			items = append(items, Item{Name: fields[0], Version: fields[1]})
		}
	}
	return items
}

// InstallRole runs ansible-galaxy role install <name> and returns combined output.
func InstallRole(name string) (string, error) {
	cmd := exec.Command("ansible-galaxy", "role", "install", name)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// InstallCollection runs ansible-galaxy collection install <name> and returns combined output.
func InstallCollection(name string) (string, error) {
	cmd := exec.Command("ansible-galaxy", "collection", "install", name)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
