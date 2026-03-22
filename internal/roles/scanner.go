// Package roles scans Ansible role directories and parses their metadata.
package roles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Task is a single task parsed from a role's tasks file.
type Task struct {
	Name   string
	Module string   // best-guess module name
	Tags   []string
}

// Role holds the metadata for a single Ansible role.
type Role struct {
	Name     string
	Path     string
	Desc     string            // from meta/main.yml galaxy_info.description
	Tasks    []Task
	Defaults map[string]string // from defaults/main.yml
	Handlers []string          // handler names from handlers/main.yml
	Deps     []string          // role dependencies from meta/main.yml
}

// Scan finds all roles under rolesDir and parses their metadata.
func Scan(rolesDir string) ([]*Role, error) {
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		return nil, fmt.Errorf("read roles dir %s: %w", rolesDir, err)
	}

	var roles []*Role
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		rolePath := filepath.Join(rolesDir, e.Name())
		r := parseRole(rolePath, e.Name())
		roles = append(roles, r)
	}

	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})
	return roles, nil
}

// ─── Internal parsers ────────────────────────────────────────────────────────

func parseRole(path, name string) *Role {
	r := &Role{
		Name:     name,
		Path:     path,
		Defaults: make(map[string]string),
	}
	r.Tasks = parseTasks(filepath.Join(path, "tasks", "main.yml"))
	r.Defaults = parseDefaults(filepath.Join(path, "defaults", "main.yml"))
	r.Handlers = parseHandlerNames(filepath.Join(path, "handlers", "main.yml"))
	r.Desc, r.Deps = parseMeta(filepath.Join(path, "meta", "main.yml"))
	return r
}

// parseTasks parses tasks/main.yml into a slice of Task.
func parseTasks(path string) []Task {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw []map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var tasks []Task
	for _, item := range raw {
		t := Task{}
		if n, ok := item["name"].(string); ok {
			t.Name = n
		}
		t.Module = guessModule(item)
		t.Tags = extractStringSlice(item["tags"])
		tasks = append(tasks, t)
	}
	return tasks
}

// guessModule picks the Ansible module key from a task map.
func guessModule(task map[string]interface{}) string {
	skip := map[string]bool{
		"name": true, "tags": true, "when": true, "loop": true,
		"with_items": true, "register": true, "become": true,
		"become_user": true, "notify": true, "ignore_errors": true,
		"failed_when": true, "changed_when": true, "no_log": true,
		"vars": true, "environment": true, "delegate_to": true,
		"run_once": true, "any_errors_fatal": true, "block": true,
		"rescue": true, "always": true, "loop_control": true,
	}
	for k := range task {
		if !skip[k] {
			// Strip FQCN prefix (ansible.builtin.apt → apt).
			parts := strings.Split(k, ".")
			return parts[len(parts)-1]
		}
	}
	return ""
}

// parseDefaults parses defaults/main.yml into a string map.
func parseDefaults(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// parseHandlerNames extracts handler names from handlers/main.yml.
func parseHandlerNames(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw []map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	var names []string
	for _, item := range raw {
		if n, ok := item["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}

// metaDoc is a minimal representation of meta/main.yml.
type metaDoc struct {
	GalaxyInfo struct {
		Description string `yaml:"description"`
	} `yaml:"galaxy_info"`
	Dependencies []interface{} `yaml:"dependencies"`
}

// parseMeta extracts role description and dependencies from meta/main.yml.
func parseMeta(path string) (desc string, deps []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var doc metaDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return
	}
	desc = doc.GalaxyInfo.Description
	for _, d := range doc.Dependencies {
		switch v := d.(type) {
		case string:
			deps = append(deps, v)
		case map[string]interface{}:
			if role, ok := v["role"].(string); ok {
				deps = append(deps, role)
			}
		}
	}
	return
}

func extractStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []interface{}:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
