// Package inventory provides parsers for Ansible inventory files.
package inventory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kocierik/lazyansible/internal/core"
	"gopkg.in/yaml.v3"
)

// Discover finds inventory files in the given directory.
// It looks for common names like inventory, hosts, inventory.ini, inventory.yaml/yml.
func Discover(dir string) []string {
	candidates := []string{
		"inventory", "hosts", "inventory.ini", "inventory.yaml", "inventory.yml",
	}
	var found []string
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	// Also check for inventories/ subdirectory.
	subdir := filepath.Join(dir, "inventories")
	if entries, err := os.ReadDir(subdir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				found = append(found, filepath.Join(subdir, e.Name()))
			}
		}
	}
	return found
}

// Parse auto-detects and parses an inventory file.
func Parse(path string) (*core.Inventory, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return parseYAML(path)
	default:
		return parseINI(path)
	}
}

// ─── INI parser ──────────────────────────────────────────────────────────────

func parseINI(path string) (*core.Inventory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open inventory: %w", err)
	}
	defer f.Close()

	inv := &core.Inventory{
		Hosts:  make(map[string]*core.Host),
		Groups: make(map[string]*core.Group),
	}

	// "all" group always exists.
	inv.Groups["all"] = &core.Group{
		Name: "all",
		Vars: make(map[string]string),
	}
	inv.OrderedGroups = []string{"all"}

	currentGroup := "ungrouped"
	inVars := false
	inChildren := false

	ensureGroup := func(name string) *core.Group {
		if g, ok := inv.Groups[name]; ok {
			return g
		}
		g := &core.Group{Name: name, Vars: make(map[string]string)}
		inv.Groups[name] = g
		inv.OrderedGroups = append(inv.OrderedGroups, name)
		return g
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header: [groupname], [groupname:vars], [groupname:children]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			inVars = false
			inChildren = false

			if strings.HasSuffix(section, ":vars") {
				currentGroup = strings.TrimSuffix(section, ":vars")
				inVars = true
				ensureGroup(currentGroup)
			} else if strings.HasSuffix(section, ":children") {
				currentGroup = strings.TrimSuffix(section, ":children")
				inChildren = true
				ensureGroup(currentGroup)
			} else {
				currentGroup = section
				ensureGroup(currentGroup)
			}
			continue
		}

		if inVars {
			key, val := splitKeyVal(line)
			if key != "" {
				inv.Groups[currentGroup].Vars[key] = val
			}
			continue
		}

		if inChildren {
			child := strings.Fields(line)[0]
			g := ensureGroup(currentGroup)
			g.Children = append(g.Children, child)
			ensureGroup(child)
			continue
		}

		// Host line: hostname [key=val ...]
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		hostName := fields[0]

		host, exists := inv.Hosts[hostName]
		if !exists {
			host = &core.Host{Name: hostName, Vars: make(map[string]string)}
			inv.Hosts[hostName] = host
		}
		host.Groups = appendUnique(host.Groups, currentGroup)

		// Inline variables.
		for _, kv := range fields[1:] {
			k, v := splitKeyVal(kv)
			if k != "" {
				host.Vars[k] = v
			}
		}

		// Add host to group.
		g := ensureGroup(currentGroup)
		g.Hosts = appendUnique(g.Hosts, hostName)

		// Every host belongs to "all".
		inv.Groups["all"].Hosts = appendUnique(inv.Groups["all"].Hosts, hostName)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan inventory: %w", err)
	}
	return inv, nil
}

func splitKeyVal(s string) (string, string) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// ─── YAML parser ─────────────────────────────────────────────────────────────

// yamlInventory matches the standard Ansible YAML inventory schema.
type yamlInventory struct {
	All struct {
		Hosts    map[string]map[string]interface{} `yaml:"hosts"`
		Vars     map[string]interface{}            `yaml:"vars"`
		Children map[string]yamlGroup              `yaml:"children"`
	} `yaml:"all"`
}

type yamlGroup struct {
	Hosts    map[string]map[string]interface{} `yaml:"hosts"`
	Vars     map[string]interface{}            `yaml:"vars"`
	Children map[string]yamlGroup              `yaml:"children"`
}

func parseYAML(path string) (*core.Inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yaml inventory: %w", err)
	}

	var raw yamlInventory
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse yaml inventory: %w", err)
	}

	inv := &core.Inventory{
		Hosts:  make(map[string]*core.Host),
		Groups: make(map[string]*core.Group),
	}

	allGroup := &core.Group{Name: "all", Vars: make(map[string]string)}
	inv.Groups["all"] = allGroup
	inv.OrderedGroups = []string{"all"}

	// Add hosts from the top-level "all" group.
	for hostName, hostVars := range raw.All.Hosts {
		host := &core.Host{
			Name:   hostName,
			Groups: []string{"all"},
			Vars:   toStringMap(hostVars),
		}
		inv.Hosts[hostName] = host
		allGroup.Hosts = appendUnique(allGroup.Hosts, hostName)
	}

	// Recurse into children.
	for groupName, groupData := range raw.All.Children {
		processYAMLGroup(inv, groupName, groupData)
	}

	return inv, nil
}

func processYAMLGroup(inv *core.Inventory, name string, data yamlGroup) {
	g, ok := inv.Groups[name]
	if !ok {
		g = &core.Group{Name: name, Vars: make(map[string]string)}
		inv.Groups[name] = g
		inv.OrderedGroups = append(inv.OrderedGroups, name)
	}

	for k, v := range data.Vars {
		g.Vars[k] = fmt.Sprintf("%v", v)
	}

	for hostName, hostVars := range data.Hosts {
		host, exists := inv.Hosts[hostName]
		if !exists {
			host = &core.Host{Name: hostName, Vars: make(map[string]string)}
			inv.Hosts[hostName] = host
		}
		host.Groups = appendUnique(host.Groups, name)

		for k, v := range toStringMap(hostVars) {
			host.Vars[k] = v
		}

		g.Hosts = appendUnique(g.Hosts, hostName)
		inv.Groups["all"].Hosts = appendUnique(inv.Groups["all"].Hosts, hostName)
	}

	for childName, childData := range data.Children {
		g.Children = appendUnique(g.Children, childName)
		processYAMLGroup(inv, childName, childData)
	}
}

func toStringMap(m map[string]interface{}) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
