package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

func TestDiscoverTemplateFolder(t *testing.T) {
	vaultDir := t.TempDir()

	// Set up .obsidian/templates.json with a configured folder
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"my-templates"}`),
		0644,
	)
	// Create the folder so it exists
	os.MkdirAll(filepath.Join(vaultDir, "my-templates"), 0755)

	folder, err := discoverTemplateFolder(vaultDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder != "my-templates" {
		t.Errorf("got %q, want %q", folder, "my-templates")
	}
}

func TestDiscoverTemplateFolderDefault(t *testing.T) {
	vaultDir := t.TempDir()

	// No .obsidian config, but create a templates/ folder
	os.MkdirAll(filepath.Join(vaultDir, "templates"), 0755)

	folder, err := discoverTemplateFolder(vaultDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder != "templates" {
		t.Errorf("got %q, want %q", folder, "templates")
	}
}

func TestTemplatesNoFolderError(t *testing.T) {
	vaultDir := t.TempDir()

	// No config, no templates/ folder
	_, err := discoverTemplateFolder(vaultDir)
	if err == nil {
		t.Fatal("expected error when no template folder configured or found")
	}
	if !strings.Contains(err.Error(), "no template folder configured or found") {
		t.Errorf("error message = %q, want to contain %q", err.Error(), "no template folder configured or found")
	}
}

func TestTemplateVariableSubstitution(t *testing.T) {
	now := time.Date(2026, 2, 19, 14, 30, 0, 0, time.UTC)
	input := "# {{title}}\nDate: {{date}}\nTime: {{time}}\n"

	got := substituteTemplateVars(input, "My Note", now)

	if !strings.Contains(got, "# My Note") {
		t.Errorf("title not substituted: %q", got)
	}
	if !strings.Contains(got, "Date: 2026-02-19") {
		t.Errorf("date not substituted: %q", got)
	}
	if !strings.Contains(got, "Time: 14:30") {
		t.Errorf("time not substituted: %q", got)
	}
}

func TestTemplateCustomDateFormat(t *testing.T) {
	now := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	input := "Created: {{date:YYYY-MM-DD}}\nYear: {{date:YYYY}}\nShort: {{date:MM/DD}}\n"

	got := substituteTemplateVars(input, "Test", now)

	if !strings.Contains(got, "Created: 2026-03-15") {
		t.Errorf("custom date YYYY-MM-DD not substituted: %q", got)
	}
	if !strings.Contains(got, "Year: 2026") {
		t.Errorf("custom date YYYY not substituted: %q", got)
	}
	if !strings.Contains(got, "Short: 03/15") {
		t.Errorf("custom date MM/DD not substituted: %q", got)
	}
}

func TestTemplateCustomTimeFormat(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 5, 0, 0, time.UTC)
	input := "Now: {{time:HH:mm}}\nFull: {{time:HH:mm:ss}}\n"

	got := substituteTemplateVars(input, "Test", now)

	if !strings.Contains(got, "Now: 09:05") {
		t.Errorf("custom time HH:mm not substituted: %q", got)
	}
	if !strings.Contains(got, "Full: 09:05:00") {
		t.Errorf("custom time HH:mm:ss not substituted: %q", got)
	}
}

func TestTemplateNoVariables(t *testing.T) {
	now := time.Now()
	input := "# Plain note\n\nNo variables here.\n"

	got := substituteTemplateVars(input, "Test", now)

	if got != input {
		t.Errorf("content changed: got %q, want %q", got, input)
	}
}

func TestTemplateUnknownVariable(t *testing.T) {
	now := time.Now()
	input := "# {{title}}\n\nUnknown: {{foo}}\nAnother: {{bar:baz}}\n"

	got := substituteTemplateVars(input, "Test", now)

	if !strings.Contains(got, "{{foo}}") {
		t.Errorf("unknown variable {{foo}} was removed: %q", got)
	}
	if !strings.Contains(got, "{{bar:baz}}") {
		t.Errorf("unknown variable {{bar:baz}} was removed: %q", got)
	}
	if !strings.Contains(got, "# Test") {
		t.Errorf("known variable {{title}} not substituted: %q", got)
	}
}

// ---------------------------------------------------------------------------
// Integration tests (real files, no mocks)
// ---------------------------------------------------------------------------

func TestTemplatesListIntegration(t *testing.T) {
	vaultDir := t.TempDir()

	// Set up template folder via config
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)

	// Create template files
	tmplDir := filepath.Join(vaultDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "Meeting Notes.md"), []byte("# {{title}}"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "Daily.md"), []byte("# {{date}}"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "not-a-template.txt"), []byte("skip me"), 0644)

	got := captureStdout(func() {
		if err := cmdTemplates(vaultDir, map[string]string{}, ""); err != nil {
			t.Fatalf("templates list: %v", err)
		}
	})

	// Should list .md files, sorted, not .txt
	if !strings.Contains(got, "Daily.md") {
		t.Errorf("missing Daily.md in output: %q", got)
	}
	if !strings.Contains(got, "Meeting Notes.md") {
		t.Errorf("missing Meeting Notes.md in output: %q", got)
	}
	if strings.Contains(got, "not-a-template.txt") {
		t.Errorf("non-md file listed: %q", got)
	}

	// Verify sorted order: Daily before Meeting Notes
	dailyIdx := strings.Index(got, "Daily.md")
	meetingIdx := strings.Index(got, "Meeting Notes.md")
	if dailyIdx > meetingIdx {
		t.Errorf("templates not sorted: Daily.md at %d, Meeting Notes.md at %d", dailyIdx, meetingIdx)
	}
}

func TestTemplatesApplyIntegration(t *testing.T) {
	vaultDir := t.TempDir()

	// Set up template folder
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)

	tmplDir := filepath.Join(vaultDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "Meeting Notes.md"),
		[]byte("---\ntype: meeting\n---\n# {{title}}\n\nDate: {{date}}\nTime: {{time}}\n\n## Attendees\n\n## Notes\n"),
		0644,
	)

	params := map[string]string{
		"template": "Meeting Notes",
		"name":     "Q1 Planning",
		"path":     "meetings/Q1 Planning.md",
	}

	if err := cmdTemplatesApply(vaultDir, params); err != nil {
		t.Fatalf("templates:apply: %v", err)
	}

	// Read the created note
	data, err := os.ReadFile(filepath.Join(vaultDir, "meetings", "Q1 Planning.md"))
	if err != nil {
		t.Fatalf("note not created: %v", err)
	}

	content := string(data)

	// Verify variable substitution
	if !strings.Contains(content, "# Q1 Planning") {
		t.Errorf("title not substituted: %q", content)
	}

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(content, "Date: "+today) {
		t.Errorf("date not substituted: %q", content)
	}

	// Time should be in HH:MM format
	if !strings.Contains(content, "Time: ") {
		t.Errorf("time not substituted: %q", content)
	}

	// Structure should be preserved
	if !strings.Contains(content, "## Attendees") {
		t.Errorf("template structure not preserved: %q", content)
	}
	if !strings.Contains(content, "type: meeting") {
		t.Errorf("frontmatter not preserved: %q", content)
	}
}

func TestTemplatesApplyExistingNote(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)

	tmplDir := filepath.Join(vaultDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "Simple.md"), []byte("# {{title}}"), 0644)

	// Create the target note first
	os.MkdirAll(filepath.Join(vaultDir, "notes"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "notes", "Existing.md"), []byte("# Existing"), 0644)

	params := map[string]string{
		"template": "Simple",
		"name":     "Existing",
		"path":     "notes/Existing.md",
	}

	err := cmdTemplatesApply(vaultDir, params)
	if err == nil {
		t.Fatal("expected error when applying to existing note")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want to contain 'already exists'", err.Error())
	}
}

func TestTemplatesApplyNotFound(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)
	os.MkdirAll(filepath.Join(vaultDir, "templates"), 0755)

	params := map[string]string{
		"template": "Nonexistent",
		"name":     "Test",
		"path":     "test.md",
	}

	err := cmdTemplatesApply(vaultDir, params)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
	if !strings.Contains(err.Error(), "template") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to mention template not found", err.Error())
	}
}

func TestTemplatesApplyCreatesDirectories(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)

	tmplDir := filepath.Join(vaultDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "Simple.md"), []byte("# {{title}}"), 0644)

	params := map[string]string{
		"template": "Simple",
		"name":     "Deep Note",
		"path":     "deeply/nested/dir/Deep Note.md",
	}

	if err := cmdTemplatesApply(vaultDir, params); err != nil {
		t.Fatalf("templates:apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "deeply", "nested", "dir", "Deep Note.md"))
	if err != nil {
		t.Fatalf("note not created at deep path: %v", err)
	}

	if !strings.Contains(string(data), "# Deep Note") {
		t.Errorf("title not substituted in deep note: %q", string(data))
	}
}

func TestTemplatesWithFormats(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "templates.json"),
		[]byte(`{"folder":"templates"}`),
		0644,
	)

	tmplDir := filepath.Join(vaultDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "Alpha.md"), []byte("# Alpha"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "Beta.md"), []byte("# Beta"), 0644)

	// Test JSON output
	jsonOut := captureStdout(func() {
		if err := cmdTemplates(vaultDir, map[string]string{}, "json"); err != nil {
			t.Fatalf("json format: %v", err)
		}
	})

	var items []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonOut)), &items); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, jsonOut)
	}
	if len(items) != 2 {
		t.Errorf("JSON: got %d items, want 2", len(items))
	}

	// Test CSV output
	csvOut := captureStdout(func() {
		if err := cmdTemplates(vaultDir, map[string]string{}, "csv"); err != nil {
			t.Fatalf("csv format: %v", err)
		}
	})
	if !strings.Contains(csvOut, "Alpha.md") || !strings.Contains(csvOut, "Beta.md") {
		t.Errorf("CSV output missing items: %q", csvOut)
	}

	// Test TSV output
	tsvOut := captureStdout(func() {
		if err := cmdTemplates(vaultDir, map[string]string{}, "tsv"); err != nil {
			t.Fatalf("tsv format: %v", err)
		}
	})
	if !strings.Contains(tsvOut, "Alpha.md") || !strings.Contains(tsvOut, "Beta.md") {
		t.Errorf("TSV output missing items: %q", tsvOut)
	}

	// Test YAML output
	yamlOut := captureStdout(func() {
		if err := cmdTemplates(vaultDir, map[string]string{}, "yaml"); err != nil {
			t.Fatalf("yaml format: %v", err)
		}
	})
	if !strings.Contains(yamlOut, "- Alpha.md") || !strings.Contains(yamlOut, "- Beta.md") {
		t.Errorf("YAML output missing items: %q", yamlOut)
	}
}

func TestTemplatesListNoFolderError(t *testing.T) {
	vaultDir := t.TempDir()

	// No config, no templates/ folder
	err := cmdTemplates(vaultDir, map[string]string{}, "")
	if err == nil {
		t.Fatal("expected error when no template folder configured or found")
	}
	if !strings.Contains(err.Error(), "no template folder configured or found") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "no template folder configured or found")
	}
}

func TestTemplatesApplyNoFolderError(t *testing.T) {
	vaultDir := t.TempDir()

	// No config, no templates/ folder
	params := map[string]string{
		"template": "Something",
		"name":     "Test",
		"path":     "test.md",
	}

	err := cmdTemplatesApply(vaultDir, params)
	if err == nil {
		t.Fatal("expected error when no template folder configured or found")
	}
	if !strings.Contains(err.Error(), "no template folder configured or found") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "no template folder configured or found")
	}
}
