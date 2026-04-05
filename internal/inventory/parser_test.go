package inventory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLFlatGroups(t *testing.T) {
	dir := t.TempDir()
	content := `groupename:
  hosts:
    client1:
      ansible_host: 192.168.1.100
      flatpak_user: alice
    client2:
      ansible_host: client2.blackwall.lan
      flatpak_user: bob
`
	p := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	inv, err := Parse(p)
	if err != nil {
		t.Fatal("Parse error:", err)
	}

	if len(inv.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(inv.Hosts))
	}
	if _, ok := inv.Hosts["client1"]; !ok {
		t.Error("missing host client1")
	}
	if _, ok := inv.Hosts["client2"]; !ok {
		t.Error("missing host client2")
	}
	if _, ok := inv.Groups["groupename"]; !ok {
		t.Error("missing group groupename")
	}
	if h := inv.Hosts["client1"]; h != nil {
		if h.Vars["ansible_host"] != "192.168.1.100" {
			t.Errorf("client1 ansible_host = %q, want 192.168.1.100", h.Vars["ansible_host"])
		}
	}
}

func TestParseYAMLStandardAllFormat(t *testing.T) {
	dir := t.TempDir()
	content := `all:
  hosts:
    server1:
      ansible_host: 10.0.0.1
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 10.0.0.10
        web2:
          ansible_host: 10.0.0.11
    dbservers:
      hosts:
        db1:
          ansible_host: 10.0.0.20
`
	p := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	inv, err := Parse(p)
	if err != nil {
		t.Fatal("Parse error:", err)
	}

	if len(inv.Hosts) != 4 {
		t.Errorf("expected 4 hosts, got %d", len(inv.Hosts))
	}
	if _, ok := inv.Groups["webservers"]; !ok {
		t.Error("missing group webservers")
	}
	if _, ok := inv.Groups["dbservers"]; !ok {
		t.Error("missing group dbservers")
	}
	if _, ok := inv.Hosts["server1"]; !ok {
		t.Error("missing host server1")
	}
}

func TestParseYAMLMultipleFlatGroups(t *testing.T) {
	dir := t.TempDir()
	content := `webservers:
  hosts:
    web1:
      ansible_host: 10.0.0.10
    web2:
      ansible_host: 10.0.0.11
dbservers:
  hosts:
    db1:
      ansible_host: 10.0.0.20
`
	p := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	inv, err := Parse(p)
	if err != nil {
		t.Fatal("Parse error:", err)
	}

	if len(inv.Hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(inv.Hosts))
	}
	if _, ok := inv.Groups["webservers"]; !ok {
		t.Error("missing group webservers")
	}
	if _, ok := inv.Groups["dbservers"]; !ok {
		t.Error("missing group dbservers")
	}
}

func TestParseJSONFlatGroups(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "groupename": {
    "hosts": {
      "client1": {
        "ansible_host": "192.168.1.100",
        "flatpak_user": "alice"
      },
      "client2": {
        "ansible_host": "client2.blackwall.lan",
        "flatpak_user": "bob"
      }
    }
  }
}`
	p := filepath.Join(dir, "inventory.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	inv, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(inv.Hosts))
	}
	if _, ok := inv.Groups["groupename"]; !ok {
		t.Error("missing group groupename")
	}
}

func TestParseJSONStandardAllFormat(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "all": {
    "hosts": {
      "server1": { "ansible_host": "10.0.0.1" }
    },
    "children": {
      "webservers": {
        "hosts": {
          "web1": { "ansible_host": "10.0.0.10", "http_port": 8080 }
        }
      }
    }
  }
}`
	p := filepath.Join(dir, "inventory.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	inv, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(inv.Hosts))
	}
	if h := inv.Hosts["web1"]; h == nil || h.Vars["http_port"] != "8080" {
		var got string
		if h != nil {
			got = h.Vars["http_port"]
		}
		t.Errorf("web1 http_port = %q, want 8080", got)
	}
}
