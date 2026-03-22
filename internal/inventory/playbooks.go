// Package inventory also handles playbook discovery.
package inventory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kocierik/lazyansible/internal/core"
	"gopkg.in/yaml.v3"
)

// DiscoverPlaybooks walks dir and returns all files that look like Ansible playbooks.
func DiscoverPlaybooks(dir string) ([]*core.Playbook, error) {
	var playbooks []*core.Playbook

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") ||
				name == "roles" || name == "collections" || name == "files" ||
				name == "templates" || name == "vars" || name == "defaults" ||
				name == "handlers" || name == "meta" || name == "tasks" ||
				name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}

		pb, ok := looksLikePlaybook(path)
		if ok {
			playbooks = append(playbooks, pb)
		}
		return nil
	})

	return playbooks, err
}

// ParseSinglePlaybook attempts to parse a single file as an Ansible playbook.
// It is exported so callers (e.g. loadPlaybooksCmd) can probe specific paths.
func ParseSinglePlaybook(path string) (*core.Playbook, bool) {
	return looksLikePlaybook(path)
}

// looksLikePlaybook returns a Playbook if the file is a valid Ansible playbook.
func looksLikePlaybook(path string) (*core.Playbook, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	// Parse as a generic YAML list to detect the playbook shape.
	var raw []interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil || len(raw) == 0 {
		return nil, false
	}

	pb := &core.Playbook{
		Name: displayName(path),
		Path: path,
	}

	isPlaybook := false
	tagSet := make(map[string]bool)

	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if _, hasHosts := m["hosts"]; hasHosts {
			isPlaybook = true
			if h, ok := m["hosts"].(string); ok {
				pb.Hosts = pbAppendUnique(pb.Hosts, h)
			}
		}
		if _, hasImport := m["import_playbook"]; hasImport {
			isPlaybook = true
		}
		// Collect all tags recursively.
		collectTags(m, tagSet)
	}

	if !isPlaybook {
		return nil, false
	}

	for tag := range tagSet {
		pb.Tags = append(pb.Tags, tag)
	}
	sort.Strings(pb.Tags)

	return pb, true
}

// collectTags recurses through a YAML map/list and collects all "tags" values.
func collectTags(node interface{}, tagSet map[string]bool) {
	switch v := node.(type) {
	case map[string]interface{}:
		if tags, ok := v["tags"]; ok {
			addTags(tags, tagSet)
		}
		// Recurse into task lists.
		for _, key := range []string{"tasks", "pre_tasks", "post_tasks", "block", "rescue", "always", "roles"} {
			if sub, ok := v[key]; ok {
				collectTags(sub, tagSet)
			}
		}
	case []interface{}:
		for _, item := range v {
			collectTags(item, tagSet)
		}
	}
}

func addTags(tags interface{}, tagSet map[string]bool) {
	switch t := tags.(type) {
	case string:
		if t != "" {
			tagSet[t] = true
		}
	case []interface{}:
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				tagSet[s] = true
			}
		}
	}
}

func pbAppendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func displayName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".yml"), ".yaml")
}
