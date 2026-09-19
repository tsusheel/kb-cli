package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

func TestCompleteStaticEnums(t *testing.T) {
	dummyCmd := &cobra.Command{}

	// 1. Note Types
	types, dir := completeNoteTypes(dummyCmd, nil, "to")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp")
	}
	hasTodo := false
	for _, typ := range types {
		if strings.HasPrefix(typ, "todo") {
			hasTodo = true
			break
		}
	}
	if !hasTodo {
		t.Errorf("completeNoteTypes missing 'todo': %v", types)
	}

	// 2. Note Statuses
	statuses, _ := completeNoteStatuses(dummyCmd, nil, "")
	if len(statuses) < 5 {
		t.Errorf("expected at least 5 statuses, got %d", len(statuses))
	}

	// 3. Link Types
	linkTypes, _ := completeLinkTypes(dummyCmd, nil, "")
	if len(linkTypes) < 5 {
		t.Errorf("expected at least 5 link types, got %d", len(linkTypes))
	}

	// 4. Areas
	areas, _ := completeAreas(dummyCmd, nil, "")
	if len(areas) < 3 {
		t.Errorf("expected at least 3 areas, got %d", len(areas))
	}

	// 5. Secret Keys
	secrets, _ := completeSecretKeys(dummyCmd, nil, "")
	if len(secrets) == 0 {
		t.Errorf("expected secret keys to be returned")
	}

	// 6. Config Keys
	configKeys, _ := completeConfigKeys(dummyCmd, nil, "remote")
	hasRemoteURL := false
	for _, k := range configKeys {
		if strings.HasPrefix(k, "remote.postgres_url") {
			hasRemoteURL = true
			break
		}
	}
	if !hasRemoteURL {
		t.Errorf("completeConfigKeys missing 'remote.postgres_url': %v", configKeys)
	}
}

func TestCompleteNoteIDs(t *testing.T) {
	setupCmdTestDB(t)

	n := &models.Note{
		Note:   "Completion Test Note",
		Type:   models.Decision,
		Status: models.Active,
	}
	if err := db.CreateNote(n); err != nil {
		t.Fatalf("failed creating test note: %v", err)
	}

	dummyCmd := &cobra.Command{}
	completions, dir := completeNoteIDs(dummyCmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp")
	}
	if len(completions) != 1 {
		t.Fatalf("expected 1 completion, got %d", len(completions))
	}
	if !strings.Contains(completions[0], "Completion Test Note") {
		t.Errorf("completion does not contain note title: %s", completions[0])
	}
}
