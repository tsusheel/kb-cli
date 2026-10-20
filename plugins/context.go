package plugins

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/plugins/api"
	"github.com/tsusheel/kb-cli/utils"
)

// NewPluginContext constructs a fully scoped PluginContext for a specific plugin.
func NewPluginContext(pluginID string, rootCmd *cobra.Command, httpServeMux *http.ServeMux, syncRegistry api.SyncRegistry) *api.PluginContext {
	return &api.PluginContext{
		PluginID:  pluginID,
		Config:    &scopedConfig{pluginID: pluginID},
		Store:     &scopedStore{pluginID: pluginID},
		Audit:     &scopedAudit{pluginID: pluginID},
		Revisions: &scopedRevisions{pluginID: pluginID},
		Sync:      syncRegistry,
		Secrets:   &scopedSecrets{pluginID: pluginID},
		Commands:  &scopedCommands{pluginID: pluginID, rootCmd: rootCmd},
		HTTP:      &scopedHTTP{pluginID: pluginID, mux: httpServeMux},
		MCP:       &scopedMCP{pluginID: pluginID},
	}
}

// --- Scoped Config ---

type scopedConfig struct {
	pluginID string
}

func (c *scopedConfig) key(k string) string {
	return fmt.Sprintf("plugins.%s.%s", c.pluginID, k)
}

func (c *scopedConfig) GetString(key string) string {
	return viper.GetString(c.key(key))
}

func (c *scopedConfig) GetInt(key string) int {
	return viper.GetInt(c.key(key))
}

func (c *scopedConfig) GetBool(key string) bool {
	return viper.GetBool(c.key(key))
}

func (c *scopedConfig) GetStringSlice(key string) []string {
	return viper.GetStringSlice(c.key(key))
}

func (c *scopedConfig) IsSet(key string) bool {
	return viper.IsSet(c.key(key))
}

// --- Scoped Store ---

type scopedStore struct {
	pluginID string
}

func (s *scopedStore) Create(ctx context.Context, entry api.Entry) error {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return err
	}
	entry.PluginID = s.pluginID

	m := toModelEntry(entry)
	actor := models.ActorContext{ActorType: "plugin", ActorID: s.pluginID}
	return db.CreateEntry(nil, &m, actor)
}

func (s *scopedStore) Get(ctx context.Context, pluginID, id string) (api.Entry, error) {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return api.Entry{}, err
	}
	if pluginID != s.pluginID {
		return api.Entry{}, fmt.Errorf("permission denied: cannot access entries belonging to other plugins")
	}

	m, err := db.GetEntry(s.pluginID, id)
	if err != nil {
		return api.Entry{}, err
	}
	return fromModelEntry(*m), nil
}

func (s *scopedStore) Update(ctx context.Context, entry api.Entry, expectedRevision int64) error {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return err
	}
	entry.PluginID = s.pluginID

	m := toModelEntry(entry)
	actor := models.ActorContext{ActorType: "plugin", ActorID: s.pluginID}
	return db.UpdateEntry(nil, &m, expectedRevision, actor)
}

func (s *scopedStore) Query(ctx context.Context, filter api.EntryFilter) ([]api.Entry, error) {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return nil, err
	}
	filter.PluginID = s.pluginID

	mf := models.EntryFilter{
		PluginID:   s.pluginID,
		EntityType: filter.EntityType,
		ParentID:   filter.ParentID,
		Status:     filter.Status,
		EntryDate:  filter.EntryDate,
		FromDate:   filter.FromDate,
		ToDate:     filter.ToDate,
		SearchText: filter.SearchText,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	modelsList, err := db.QueryEntries(mf)
	if err != nil {
		return nil, err
	}

	var results []api.Entry
	for _, m := range modelsList {
		results = append(results, fromModelEntry(m))
	}
	return results, nil
}

func (s *scopedStore) SoftDelete(ctx context.Context, pluginID, id, reason string) error {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return err
	}
	if pluginID != s.pluginID {
		return fmt.Errorf("permission denied: cannot delete entries belonging to other plugins")
	}

	actor := models.ActorContext{ActorType: "plugin", ActorID: s.pluginID}
	return db.SoftDeleteEntry(nil, s.pluginID, id, reason, actor)
}

func (s *scopedStore) Restore(ctx context.Context, pluginID, id string, expectedRevision int64) error {
	if err := CheckCapability(s.pluginID, api.CapStorageEntries); err != nil {
		return err
	}
	if pluginID != s.pluginID {
		return fmt.Errorf("permission denied: cannot restore entries belonging to other plugins")
	}

	actor := models.ActorContext{ActorType: "plugin", ActorID: s.pluginID}
	return db.RestoreEntry(nil, s.pluginID, id, expectedRevision, actor)
}

func toModelEntry(e api.Entry) models.PluginEntry {
	dataStr := string(e.Data)
	if strings.TrimSpace(dataStr) == "" {
		dataStr = "{}"
	}
	return models.PluginEntry{
		ID:            e.ID,
		PluginID:      e.PluginID,
		EntityType:    e.EntityType,
		SchemaVersion: e.SchemaVersion,
		ParentID:      e.ParentID,
		Status:        e.Status,
		OccurredAt:    e.OccurredAt,
		EntryDate:     e.EntryDate,
		NumVal:        e.NumVal,
		SearchText:    e.SearchText,
		Data:          dataStr,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		DeletedAt:     e.DeletedAt,
		DeletedNote:   e.DeletedNote,
		Revision:      e.Revision,
	}
}

func fromModelEntry(m models.PluginEntry) api.Entry {
	return api.Entry{
		ID:            m.ID,
		PluginID:      m.PluginID,
		EntityType:    m.EntityType,
		SchemaVersion: m.SchemaVersion,
		ParentID:      m.ParentID,
		Status:        m.Status,
		OccurredAt:    m.OccurredAt,
		EntryDate:     m.EntryDate,
		NumVal:        m.NumVal,
		SearchText:    m.SearchText,
		Data:          []byte(m.Data),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
		DeletedNote:   m.DeletedNote,
		Revision:      m.Revision,
	}
}

// --- Scoped Audit ---

type scopedAudit struct {
	pluginID string
}

func (a *scopedAudit) Record(ctx context.Context, event api.AuditEvent) error {
	if err := CheckCapability(a.pluginID, api.CapAudit); err != nil {
		return err
	}
	event.PluginID = a.pluginID

	diffStr := string(event.DiffJSON)
	if diffStr == "" {
		diffStr = "{}"
	}

	query := `INSERT INTO audit_logs (
		id, plugin_id, entity_type, entity_id, revision, action, actor_type, actor_id, request_id,
		changes_summary, diff_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.DB.ExecContext(ctx, query,
		event.ID, a.pluginID, event.EntityType, event.EntityID, event.Revision, event.Action,
		event.ActorType, event.ActorID, event.RequestID, event.ChangesSummary, diffStr, event.CreatedAt)
	return err
}

// --- Scoped Revisions ---

type scopedRevisions struct {
	pluginID string
}

func (r *scopedRevisions) GetRevision(ctx context.Context, pluginID, entityType, entityID string, revision int64) ([]byte, error) {
	if err := CheckCapability(r.pluginID, api.CapStorageEntries); err != nil {
		return nil, err
	}
	if pluginID != r.pluginID {
		return nil, fmt.Errorf("permission denied: cannot read revisions of other plugins")
	}

	var snapJSON string
	err := db.DB.QueryRowContext(ctx, `SELECT snapshot_json FROM entity_revisions WHERE plugin_id = ? AND entity_type = ? AND entity_id = ? AND revision = ?`,
		r.pluginID, entityType, entityID, revision).Scan(&snapJSON)
	if err != nil {
		return nil, err
	}
	return []byte(snapJSON), nil
}

func (r *scopedRevisions) ListRevisions(ctx context.Context, pluginID, entityType, entityID string, limit int) ([]api.RevisionInfo, error) {
	if err := CheckCapability(r.pluginID, api.CapStorageEntries); err != nil {
		return nil, err
	}
	if pluginID != r.pluginID {
		return nil, fmt.Errorf("permission denied: cannot list revisions of other plugins")
	}

	modelsRevs, err := db.GetEntityRevisions(r.pluginID, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}

	var results []api.RevisionInfo
	for _, m := range modelsRevs {
		results = append(results, api.RevisionInfo{
			RevisionID:    m.RevisionID,
			Revision:      m.Revision,
			SchemaVersion: m.SchemaVersion,
			CreatedAt:     m.CreatedAt,
		})
	}
	return results, nil
}

func (r *scopedRevisions) RestoreRevision(ctx context.Context, pluginID, entityType, entityID string, revision int64, expectedCurrentRevision int64) error {
	if err := CheckCapability(r.pluginID, api.CapStorageEntries); err != nil {
		return err
	}
	if pluginID != r.pluginID {
		return fmt.Errorf("permission denied")
	}

	actor := models.ActorContext{ActorType: "plugin", ActorID: r.pluginID}
	_, err := db.RestoreEntryToRevision(r.pluginID, entityType, entityID, revision, expectedCurrentRevision, actor)
	return err
}

// --- Scoped Secrets ---

type scopedSecrets struct {
	pluginID string
}

func (s *scopedSecrets) key(k string) string {
	return fmt.Sprintf("plugin_%s_%s", s.pluginID, k)
}

func (s *scopedSecrets) Get(key string) (string, error) {
	if err := CheckCapability(s.pluginID, api.CapSecrets); err != nil {
		return "", err
	}
	return utils.GetSecret(s.key(key))
}

func (s *scopedSecrets) Set(key string, val string) error {
	if err := CheckCapability(s.pluginID, api.CapSecrets); err != nil {
		return err
	}
	return utils.SetSecret(s.key(key), val)
}

func (s *scopedSecrets) Delete(key string) error {
	if err := CheckCapability(s.pluginID, api.CapSecrets); err != nil {
		return err
	}
	return utils.DeleteSecret(s.key(key))
}

// --- Scoped Commands ---

type scopedCommands struct {
	pluginID string
	rootCmd  *cobra.Command
}

func (c *scopedCommands) AddCommand(cmd *cobra.Command) {
	if err := CheckCapability(c.pluginID, api.CapCLICommands); err != nil {
		// Log error or ignore unpermitted command registration
		return
	}
	if c.rootCmd != nil && cmd != nil {
		c.rootCmd.AddCommand(cmd)
	}
}

// --- Scoped HTTP ---

type scopedHTTP struct {
	pluginID string
	mux      *http.ServeMux
}

func (h *scopedHTTP) HandleFunc(pattern string, handler http.HandlerFunc) {
	if err := CheckCapability(h.pluginID, api.CapHTTPRoutes); err != nil {
		return
	}
	if h.mux == nil {
		return
	}

	// Mount under namespaced route: /api/plugins/<plugin_id>/<pattern>
	trimmed := strings.TrimPrefix(pattern, "/")
	mountPath := fmt.Sprintf("/api/plugins/%s/%s", h.pluginID, trimmed)
	h.mux.HandleFunc(mountPath, handler)
}

// --- Scoped MCP ---

type scopedMCP struct {
	pluginID string
}

var mcpToolRegistry = make(map[string]api.MCPToolDef)

func (m *scopedMCP) RegisterTool(tool api.MCPToolDef) error {
	if err := CheckCapability(m.pluginID, api.CapMCPTools); err != nil {
		return err
	}
	if tool.Name == "" {
		return fmt.Errorf("MCP tool name cannot be empty")
	}

	mcpToolRegistry[tool.Name] = tool
	return nil
}

// GetRegisteredMCPTools returns all tools registered by plugins.
func GetRegisteredMCPTools() []api.MCPToolDef {
	var list []api.MCPToolDef
	for _, t := range mcpToolRegistry {
		list = append(list, t)
	}
	return list
}

// DispatchMCPTool calls a plugin-registered MCP tool with actor attribution.
func DispatchMCPTool(ctx context.Context, toolName string, argsJSON []byte) (any, error) {
	tool, ok := mcpToolRegistry[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %q not found in plugin registry", toolName)
	}
	return tool.Handler(ctx, argsJSON)
}
