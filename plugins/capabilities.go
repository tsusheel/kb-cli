package plugins

import (
	"fmt"

	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/plugins/api"
)

// ErrCapabilityNotGranted is returned when a plugin attempts an action without authorization.
type ErrCapabilityNotGranted struct {
	PluginID   string
	Capability string
}

func (e *ErrCapabilityNotGranted) Error() string {
	return fmt.Sprintf("permission denied: plugin %q has not been granted capability %q", e.PluginID, e.Capability)
}

// CheckCapability verifies that a plugin has been explicitly granted a capability.
func CheckCapability(pluginID, capability string) error {
	granted, err := db.HasPluginCapability(pluginID, capability)
	if err != nil {
		return fmt.Errorf("failed verifying capability for plugin %s: %w", pluginID, err)
	}
	if !granted {
		return &ErrCapabilityNotGranted{PluginID: pluginID, Capability: capability}
	}
	return nil
}

// GrantCapability grants a capability to a plugin.
func GrantCapability(pluginID, capability string) error {
	return db.SetPluginCapability(pluginID, capability, true)
}

// RevokeCapability revokes a capability from a plugin.
func RevokeCapability(pluginID, capability string) error {
	return db.SetPluginCapability(pluginID, capability, false)
}

// GetGrantedCapabilities returns all capabilities granted to a plugin.
func GetGrantedCapabilities(pluginID string) ([]string, error) {
	return db.GetPluginCapabilities(pluginID)
}

// ValidateManifestCapabilities ensures all capabilities requested in a manifest are valid known capabilities.
func ValidateManifestCapabilities(m api.Manifest) error {
	validCaps := map[string]bool{
		api.CapStorageEntries: true,
		api.CapStorageDomain:  true,
		api.CapAudit:          true,
		api.CapSearch:         true,
		api.CapSync:           true,
		api.CapCLICommands:    true,
		api.CapMCPTools:       true,
		api.CapHTTPRoutes:     true,
		api.CapSecrets:        true,
	}

	for _, c := range m.Capabilities {
		if !validCaps[c] {
			return fmt.Errorf("unknown capability declared in manifest %q: %q", m.ID, c)
		}
	}
	return nil
}
