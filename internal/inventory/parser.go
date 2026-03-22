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

// inventoryCandidates is the ordered list of file names checked during auto-discovery.
var inventoryCandidates = []string{
	"inventory.yml", "inventory.yaml",
	"hosts.yml", "hosts.yaml", "hosts",
	"inventory", "inventory.ini",
}

// Discover finds inventory files by searching dir and its parent directory.
// It checks inventoryCandidates in both locations and also scans any inventories/
// sub-directory found in either place.
func Discover(dir string) []string {
	seen := map[string]bool{}
	var found []string

	add := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil || seen[abs] {
			return
		}
		if _, err := os.Stat(abs); err == nil {
			seen[abs] = true
			found = append(found, abs)
		}
	}

	searchDirs := []string{dir}
	if parent := filepath.Dir(dir); parent != dir {
		searchDirs = append(searchDirs, parent)
	}

	for _, searchDir := range searchDirs {
		for _, name := range inventoryCandidates {
			add(filepath.Join(searchDir, name))
		}
		// Also scan any inventories/ sub-directory.
		subdir := filepath.Join(searchDir, "inventories")
		if entries, err := os.ReadDir(subdir); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					add(filepath.Join(subdir, e.Name()))
				}
			}
		}
	}
	return found
}

// Parse auto-detects and parses an inventory file.
// After parsing it also merges any group_vars/ and host_vars/ directories
// found next to the inventory file or in its parent directory (matching
// standard Ansible lookup order).
func Parse(path string) (*core.Inventory, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var inv *core.Inventory
	var err error
	switch ext {
	case ".yaml", ".yml":
		inv, err = parseYAML(path)
	default:
		inv, err = parseINI(path)
	}
	if err != nil {
		return nil, err
	}

	// Load group_vars/ and host_vars/ from standard Ansible locations.
	inventoryDir := filepath.Dir(path)
	searchDirs := []string{inventoryDir}
	if parent := filepath.Dir(inventoryDir); parent != inventoryDir {
		searchDirs = append(searchDirs, parent)
	}
	for _, dir := range searchDirs {
		loadGroupVars(inv, filepath.Join(dir, "group_vars"))
		loadHostVars(inv, filepath.Join(dir, "host_vars"))
	}

	return inv, nil
}

// loadGroupVars merges YAML files from a group_vars directory into inv.Groups.
// Supports both flat files (group_vars/webservers.yml) and directory form
// (group_vars/webservers/main.yml or group_vars/webservers/*.yml).
func loadGroupVars(inv *core.Inventory, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory doesn't exist — silently skip
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			// Directory form: group_vars/<groupname>/ — load all YAML files inside.
			groupName := name
			g := findOrCreateGroup(inv, groupName)
			subEntries, _ := os.ReadDir(filepath.Join(dir, name))
			for _, se := range subEntries {
				if !se.IsDir() && isYAMLFile(se.Name()) {
					mergeYAMLFile(filepath.Join(dir, name, se.Name()), g.Vars)
				}
			}
		} else if isYAMLFile(name) {
			// Flat file: group_vars/webservers.yml → group name is stem.
			groupName := yamlStem(name)
			g := findOrCreateGroup(inv, groupName)
			mergeYAMLFile(filepath.Join(dir, name), g.Vars)
		}
	}
}

// loadHostVars merges YAML files from a host_vars directory into inv.Hosts.
func loadHostVars(inv *core.Inventory, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			hostName := name
			h := findOrCreateHost(inv, hostName)
			subEntries, _ := os.ReadDir(filepath.Join(dir, name))
			for _, se := range subEntries {
				if !se.IsDir() && isYAMLFile(se.Name()) {
					mergeYAMLFile(filepath.Join(dir, name, se.Name()), h.Vars)
				}
			}
		} else if isYAMLFile(name) {
			hostName := yamlStem(name)
			h := findOrCreateHost(inv, hostName)
			mergeYAMLFile(filepath.Join(dir, name), h.Vars)
		}
	}
}

func findOrCreateGroup(inv *core.Inventory, name string) *core.Group {
	if g, ok := inv.Groups[name]; ok {
		return g
	}
	g := &core.Group{Name: name, Vars: make(map[string]string)}
	inv.Groups[name] = g
	inv.OrderedGroups = append(inv.OrderedGroups, name)
	return g
}

func findOrCreateHost(inv *core.Inventory, name string) *core.Host {
	if h, ok := inv.Hosts[name]; ok {
		return h
	}
	h := &core.Host{Name: name, Vars: make(map[string]string)}
	inv.Hosts[name] = h
	return h
}

// mergeYAMLFile reads a YAML file and merges its top-level key/value pairs
// into dest. Non-scalar values are serialised to their YAML string form.
func mergeYAMLFile(path string, dest map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	for k, v := range raw {
		dest[k] = yamlValueString(v)
	}
}

// yamlValueString converts an arbitrary YAML value to a human-readable string.
func yamlValueString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	default:
		// For maps/slices, re-marshal to compact YAML.
		out, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return strings.TrimSpace(string(out))
	}
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yml" || ext == ".yaml"
}

func yamlStem(name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
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
