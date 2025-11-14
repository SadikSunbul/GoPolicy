package policy

import (
	"fmt"
	"os"
	"path/filepath"
)

// PolicyState represents policy states.
type PolicyState int

const (
	PolicyStateNotConfigured PolicyState = 0
	PolicyStateDisabled      PolicyState = 1
	PolicyStateEnabled       PolicyState = 2
	PolicyStateUnknown       PolicyState = 3
)

// String returns the readable value of a PolicyState.
func (ps PolicyState) String() string {
	switch ps {
	case PolicyStateNotConfigured:
		return "Not Configured"
	case PolicyStateDisabled:
		return "Disabled"
	case PolicyStateEnabled:
		return "Enabled"
	default:
		return "Unknown"
	}
}

// GetPolPath returns the path to the pol file for a given section
func GetPolPath(section AdmxPolicySection) (string, error) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = "C:\\Windows"
	}

	basePath := filepath.Join(systemRoot, "System32", "GroupPolicy")

	switch section {
	case User:
		return filepath.Join(basePath, "User", "Registry.pol"), nil
	case Machine:
		return filepath.Join(basePath, "Machine", "Registry.pol"), nil
	default:
		return "", fmt.Errorf("invalid section: %d", section)
	}
}
