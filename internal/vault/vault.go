// Package vault provides helpers for detecting and working with Ansible Vault files.
package vault

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const vaultHeader = "$ANSIBLE_VAULT;"

// IsEncrypted reports whether the file at path is an Ansible Vault file.
func IsEncrypted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if sc.Scan() {
		return strings.HasPrefix(sc.Text(), vaultHeader)
	}
	return false
}

// FindEncryptedFiles scans a directory tree for vault-encrypted files.
// It returns the relative paths of any encrypted files found.
func FindEncryptedFiles(root string) []string {
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" && ext != ".json" && ext != "" {
			return nil
		}
		if IsEncrypted(path) {
			rel, _ := filepath.Rel(root, path)
			found = append(found, rel)
		}
		return nil
	})
	return found
}

// WriteTempPassword writes a vault password to a temporary file and returns
// the file path. The caller is responsible for deleting the file.
func WriteTempPassword(password string) (string, error) {
	f, err := os.CreateTemp("", "lazyansible-vault-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(password + "\n")
	return f.Name(), err
}
