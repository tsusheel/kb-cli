package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)

// completeNoteIDs returns active note IDs with titles as descriptions (e.g. "2d64a5a\tnew note")
func completeNoteIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	notes, err := db.ListNotes("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, n := range notes {
		shortID := utils.ShortID(n.ID)
		if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
			title := n.Note
			if len(title) > 35 {
				title = title[:32] + "..."
			}
			completions = append(completions, fmt.Sprintf("%s\t%s (%s)", shortID, title, n.Type))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeUnpromotedLogIDs returns unpromoted daily log IDs
func completeUnpromotedLogIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	logs, err := db.GetUnpromotedDailyLogs()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, l := range logs {
		shortID := utils.ShortID(l.ID)
		if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
			content := l.Content
			if len(content) > 35 {
				content = content[:32] + "..."
			}
			completions = append(completions, fmt.Sprintf("%s\t%s", shortID, content))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeNoteTypes returns available note types with descriptions
func completeNoteTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{
		"note\tGeneral atomic thought or note",
		"todo\tActionable task with optional due date",
		"project\tOngoing initiative or milestone",
		"idea\tEarly-stage spark or hypothesis",
		"concept\tMental model or definition",
		"decision\tArchitecture decision record (ADR)",
		"til\tToday I Learned discovery",
		"question\tOpen inquiry driving research",
		"resource\tCurated reference or tool",
		"experiment\tEmpirical test or log",
		"person\tCollaborator or expert reference",
	}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// completeNoteStatuses returns lifecycle statuses
func completeNoteStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statuses := []string{
		"active\tStandard active note",
		"raw\tFast fleeting unrefined capture",
		"refined\tStructured and tagged note",
		"in-progress\tCurrently active project or task",
		"completed\tAccomplished task or validated experiment",
		"archived\tHistorical archived knowledge",
	}
	return statuses, cobra.ShellCompDirectiveNoFileComp
}

// completeLinkTypes returns semantic link types
func completeLinkTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	linkTypes := []string{
		"related_to\tGeneral bidirectional relationship",
		"depends_on\tPrerequisite dependency",
		"part_of\tSub-component of a larger project",
		"inspired_by\tOrigin or inspiration source",
		"supports\tSupporting evidence or reasoning",
		"contradicts\tConflicting viewpoint or trade-off",
		"about\tSubject matter or conceptual focus",
		"created_by\tAttribution or author relationship",
	}
	return linkTypes, cobra.ShellCompDirectiveNoFileComp
}

// completeAreas returns standard areas
func completeAreas(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	areas := []string{
		"work\tWork, job, and career initiatives",
		"personal\tPersonal life and habits",
		"finance\tInvestments and financial planning",
	}
	return areas, cobra.ShellCompDirectiveNoFileComp
}

// completeConfigKeys returns available config keys for kb config set / get
func completeConfigKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keys := []string{
		"date_format\tCustom date formatting layout",
		"remote.postgres_url\tRemote PostgreSQL connection URL",
		"remote.enabled\tEnable/disable remote synchronization",
		"remote.provider\tRemote provider name (postgres)",
		"app_name\tApplication identifier",
		"base_path\tBase directory for database and assets",
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

// completeSecretKeys returns available secret keys for kb config set-secret
func completeSecretKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	secrets := []string{
		"postgres_password\tPostgreSQL database password",
		"db_password\tDatabase password alias",
	}
	return secrets, cobra.ShellCompDirectiveNoFileComp
}

// completeAuditIDs returns audit revision IDs for a specific note or log if first arg is given
func completeAuditIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeNoteIDs(cmd, args, toComplete)
	}

	id := args[0]
	var entityID string
	var entityType string
	if n, err := db.GetNote(id); err == nil {
		entityID = n.ID
		entityType = "note"
	} else if l, err := db.GetDailyLog(id); err == nil {
		entityID = l.ID
		entityType = "daily_log"
	} else {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, err := db.GetAuditHistory(entityType, entityID, 30)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, e := range entries {
		shortRev := utils.ShortID(e.ID)
		if strings.HasPrefix(shortRev, toComplete) || toComplete == "" {
			desc := fmt.Sprintf("[%s] %s (%s)", e.Action, e.ChangesSummary, e.CreatedAt.Format("15:04"))
			completions = append(completions, fmt.Sprintf("%s\t%s", shortRev, desc))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

