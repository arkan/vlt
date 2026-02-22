package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTasks(t *testing.T) {
	text := `# My Note

- [ ] Buy groceries
- [x] Review PR
- [X] Deploy changes
  - [ ] Nested task
Some random text
- [ ] Another task
`
	tasks := parseTasks(text)

	if len(tasks) != 5 {
		t.Fatalf("got %d tasks, want 5", len(tasks))
	}

	// First task
	if tasks[0].Text != "Buy groceries" || tasks[0].Done || tasks[0].Line != 3 {
		t.Errorf("task[0] = %+v, want Buy groceries, done=false, line=3", tasks[0])
	}

	// Second task (done)
	if tasks[1].Text != "Review PR" || !tasks[1].Done || tasks[1].Line != 4 {
		t.Errorf("task[1] = %+v, want Review PR, done=true, line=4", tasks[1])
	}

	// Third task (X uppercase)
	if !tasks[2].Done {
		t.Errorf("task[2] should be done (uppercase X)")
	}

	// Fourth task (nested)
	if tasks[3].Text != "Nested task" || tasks[3].Done {
		t.Errorf("task[3] = %+v, want Nested task, done=false", tasks[3])
	}
}

func TestParseTasks_Empty(t *testing.T) {
	tasks := parseTasks("# No tasks here\n\nJust text.\n")
	if len(tasks) != 0 {
		t.Errorf("got %d tasks, want 0", len(tasks))
	}
}

func TestFilterTasks(t *testing.T) {
	tasks := []task{
		{Text: "Done task", Done: true},
		{Text: "Pending task", Done: false},
		{Text: "Another done", Done: true},
	}

	done := filterTasks(tasks, true, false)
	if len(done) != 2 {
		t.Errorf("done filter: got %d, want 2", len(done))
	}

	pending := filterTasks(tasks, false, true)
	if len(pending) != 1 {
		t.Errorf("pending filter: got %d, want 1", len(pending))
	}

	all := filterTasks(tasks, false, false)
	if len(all) != 3 {
		t.Errorf("no filter: got %d, want 3", len(all))
	}
}

func TestCmdTasks_SingleFile(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Tasks.md"),
		[]byte("# Tasks\n\n- [ ] Do thing 1\n- [x] Done thing\n- [ ] Do thing 2\n"),
		0644,
	)

	params := map[string]string{"file": "Tasks"}
	flags := map[string]bool{}
	if err := cmdTasks(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks single file: %v", err)
	}
}

func TestCmdTasks_VaultWide(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "projects"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)

	os.WriteFile(
		filepath.Join(vaultDir, "Daily.md"),
		[]byte("- [ ] Buy groceries\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "projects", "Plan.md"),
		[]byte("- [x] Review PR\n- [ ] Deploy\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "hidden.md"),
		[]byte("- [ ] Should be skipped\n"),
		0644,
	)

	params := map[string]string{}
	flags := map[string]bool{}
	if err := cmdTasks(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks vault-wide: %v", err)
	}
}

func TestCmdTasks_FilterDone(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Tasks.md"),
		[]byte("- [ ] Pending\n- [x] Done\n"),
		0644,
	)

	params := map[string]string{"file": "Tasks"}
	flags := map[string]bool{"done": true}
	if err := cmdTasks(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks filter done: %v", err)
	}
}

func TestCmdTasks_FilterPending(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Tasks.md"),
		[]byte("- [ ] Pending\n- [x] Done\n"),
		0644,
	)

	params := map[string]string{"file": "Tasks"}
	flags := map[string]bool{"pending": true}
	if err := cmdTasks(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks filter pending: %v", err)
	}
}

func TestCmdTasks_PathFilter(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "projects"), 0755)

	os.WriteFile(
		filepath.Join(vaultDir, "Root.md"),
		[]byte("- [ ] Root task\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "projects", "Project.md"),
		[]byte("- [ ] Project task\n"),
		0644,
	)

	params := map[string]string{"path": "projects"}
	flags := map[string]bool{}
	if err := cmdTasks(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks path filter: %v", err)
	}
}

func TestCmdTasks_JSONOutput(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Tasks.md"),
		[]byte("- [ ] Buy groceries\n- [x] Review PR\n"),
		0644,
	)

	params := map[string]string{"file": "Tasks"}
	flags := map[string]bool{"--json": true}

	got := captureStdout(func() {
		if err := cmdTasks(vaultDir, params, flags); err != nil {
			t.Fatalf("tasks json: %v", err)
		}
	})

	if got == "" {
		t.Fatal("expected json output, got empty")
	}
	if got[0] != '[' {
		t.Errorf("expected json array, got: %q", got[:20])
	}
}

// --- Parsing tests ---

func TestParseTaskMeta_Dataview(t *testing.T) {
	clean, meta, isEmoji := parseTaskMeta("Buy groceries [due:: 2024-01-15] [priority:: high]")
	if clean != "Buy groceries" {
		t.Errorf("cleanText = %q, want %q", clean, "Buy groceries")
	}
	if meta.Due != "2024-01-15" {
		t.Errorf("due = %q, want 2024-01-15", meta.Due)
	}
	if meta.Priority != "high" {
		t.Errorf("priority = %q, want high", meta.Priority)
	}
	if isEmoji {
		t.Error("expected isEmoji=false for Dataview format")
	}
}

func TestParseTaskMeta_Emoji(t *testing.T) {
	clean, meta, isEmoji := parseTaskMeta("Buy groceries 📅 2024-01-15 ⏫")
	if clean != "Buy groceries" {
		t.Errorf("cleanText = %q, want %q", clean, "Buy groceries")
	}
	if meta.Due != "2024-01-15" {
		t.Errorf("due = %q, want 2024-01-15", meta.Due)
	}
	if meta.Priority != "high" {
		t.Errorf("priority = %q, want high", meta.Priority)
	}
	if !isEmoji {
		t.Error("expected isEmoji=true for emoji format")
	}
}

func TestParseTaskMeta_Empty(t *testing.T) {
	clean, meta, isEmoji := parseTaskMeta("Just a plain task")
	if clean != "Just a plain task" {
		t.Errorf("cleanText = %q, want unchanged", clean)
	}
	if meta.Due != "" || meta.Priority != "" {
		t.Errorf("expected empty meta, got due=%q priority=%q", meta.Due, meta.Priority)
	}
	if isEmoji {
		t.Error("expected isEmoji=false for plain text")
	}
}

func TestParseTaskMeta_AllDataviewFields(t *testing.T) {
	raw := "Task [due:: 2024-01-15] [scheduled:: 2024-01-10] [start:: 2024-01-01] [created:: 2024-01-01] [priority:: medium] [repeat:: every day] [id:: abc123] [dependsOn:: xyz]"
	clean, meta, _ := parseTaskMeta(raw)
	if clean != "Task" {
		t.Errorf("cleanText = %q, want Task", clean)
	}
	if meta.Due != "2024-01-15" {
		t.Errorf("due = %q", meta.Due)
	}
	if meta.Scheduled != "2024-01-10" {
		t.Errorf("scheduled = %q", meta.Scheduled)
	}
	if meta.Start != "2024-01-01" {
		t.Errorf("start = %q", meta.Start)
	}
	if meta.Created != "2024-01-01" {
		t.Errorf("created = %q", meta.Created)
	}
	if meta.Priority != "medium" {
		t.Errorf("priority = %q", meta.Priority)
	}
	if meta.Repeat != "every day" {
		t.Errorf("repeat = %q", meta.Repeat)
	}
	if meta.ID != "abc123" {
		t.Errorf("id = %q", meta.ID)
	}
	if meta.DependsOn != "xyz" {
		t.Errorf("dependsOn = %q", meta.DependsOn)
	}
}

// --- Serialization tests ---

func TestBuildTaskLine_Dataview(t *testing.T) {
	meta := taskMeta{Due: "2024-01-15", Priority: "high"}
	got := buildTaskLine("", false, "Buy groceries", meta, false)
	want := "- [ ] Buy groceries [due:: 2024-01-15] [priority:: high]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildTaskLine_Emoji(t *testing.T) {
	meta := taskMeta{Due: "2024-01-15", Priority: "high"}
	got := buildTaskLine("", false, "Buy groceries", meta, true)
	if !strings.Contains(got, "- [ ] Buy groceries") {
		t.Errorf("missing checkbox prefix: %q", got)
	}
	if !strings.Contains(got, "⏫") {
		t.Errorf("missing high priority emoji: %q", got)
	}
	if !strings.Contains(got, "📅 2024-01-15") {
		t.Errorf("missing due date: %q", got)
	}
}

func TestBuildTaskLine_Done(t *testing.T) {
	meta := taskMeta{Completion: "2024-01-20"}
	got := buildTaskLine("", true, "Finished task", meta, false)
	if !strings.HasPrefix(got, "- [x]") {
		t.Errorf("expected [x] checkbox: %q", got)
	}
	if !strings.Contains(got, "[completion:: 2024-01-20]") {
		t.Errorf("missing completion date: %q", got)
	}
}

func TestBuildTaskLine_Indented(t *testing.T) {
	got := buildTaskLine("  ", false, "Nested task", taskMeta{}, false)
	if !strings.HasPrefix(got, "  - [ ]") {
		t.Errorf("expected indentation: %q", got)
	}
}

// --- Resolution tests ---

func TestResolveTask_ByID(t *testing.T) {
	lines := strings.Split("# Tasks\n- [ ] Task A [id:: abc]\n- [ ] Task B [id:: def]", "\n")
	tk, idx, err := resolveTask(lines, map[string]string{"id": "def"})
	if err != nil {
		t.Fatalf("resolveTask: %v", err)
	}
	if idx != 2 {
		t.Errorf("lineIdx = %d, want 2", idx)
	}
	if tk.Meta.ID != "def" {
		t.Errorf("id = %q, want def", tk.Meta.ID)
	}
}

func TestResolveTask_ByLine(t *testing.T) {
	lines := strings.Split("- [ ] First\n- [ ] Second\n- [ ] Third", "\n")
	tk, idx, err := resolveTask(lines, map[string]string{"line": "2"})
	if err != nil {
		t.Fatalf("resolveTask: %v", err)
	}
	if idx != 1 {
		t.Errorf("lineIdx = %d, want 1", idx)
	}
	if tk.CleanText != "Second" {
		t.Errorf("text = %q, want Second", tk.CleanText)
	}
}

func TestResolveTask_ByMatch(t *testing.T) {
	lines := strings.Split("- [ ] Buy groceries\n- [ ] Review PR\n- [ ] Call dentist", "\n")
	tk, _, err := resolveTask(lines, map[string]string{"match": "review"})
	if err != nil {
		t.Fatalf("resolveTask: %v", err)
	}
	if tk.CleanText != "Review PR" {
		t.Errorf("text = %q, want Review PR", tk.CleanText)
	}
}

func TestResolveTask_NotFound(t *testing.T) {
	lines := strings.Split("- [ ] Task A\n- [ ] Task B", "\n")
	_, _, err := resolveTask(lines, map[string]string{"id": "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestResolveTask_NoIdentifier(t *testing.T) {
	lines := strings.Split("- [ ] Task A", "\n")
	_, _, err := resolveTask(lines, map[string]string{})
	if err == nil {
		t.Fatal("expected error when no identifier given")
	}
}

// --- Command tests ---

func TestCmdTasksAdd_EndOfFile(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("# My Note\n\nSome text\n"), 0644)

	params := map[string]string{"file": "Note", "content": "New task"}
	flags := map[string]bool{}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "- [ ] New task") {
		t.Errorf("task not found in file: %s", content)
	}
	// Should have auto-created date
	if !strings.Contains(content, "[created::") {
		t.Errorf("missing auto-created date: %s", content)
	}
}

func TestCmdTasksAdd_WithHeading(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("# My Note\n\n## TODO\n\nSome existing content\n\n## Done\n\nFinished stuff\n"), 0644)

	params := map[string]string{"file": "Note", "content": "New task", "heading": "## TODO", "section": "start"}
	flags := map[string]bool{}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add heading: %v", err)
	}

	data, _ := os.ReadFile(note)
	lines := strings.Split(string(data), "\n")
	// Task should be right after "## TODO" heading
	found := false
	for i, line := range lines {
		if strings.Contains(line, "## TODO") && i+1 < len(lines) {
			if strings.Contains(lines[i+1], "- [ ] New task") {
				found = true
			}
			break
		}
	}
	if !found {
		t.Errorf("task not inserted after heading: %s", string(data))
	}
}

func TestCmdTasksAdd_WithHeadingSectionEnd(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("# My Note\n\n## TODO\n\n- [ ] Existing task\n\n## Done\n"), 0644)

	params := map[string]string{"file": "Note", "content": "End task", "heading": "## TODO", "section": "end"}
	flags := map[string]bool{}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add heading end: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	// Task should be before "## Done"
	todoIdx := strings.Index(content, "End task")
	doneIdx := strings.Index(content, "## Done")
	if todoIdx < 0 || doneIdx < 0 || todoIdx > doneIdx {
		t.Errorf("task not inserted at section end: %s", content)
	}
}

func TestCmdTasksAdd_AtLine(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("Line 1\nLine 2\nLine 3\n"), 0644)

	params := map[string]string{"file": "Note", "content": "Inserted task", "line": "2"}
	flags := map[string]bool{}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add at line: %v", err)
	}

	data, _ := os.ReadFile(note)
	lines := strings.Split(string(data), "\n")
	if !strings.Contains(lines[1], "- [ ] Inserted task") {
		t.Errorf("task not at line 2: %v", lines)
	}
}

func TestCmdTasksAdd_WithMetadata(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("# Tasks\n"), 0644)

	params := map[string]string{
		"file": "Note", "content": "Important task",
		"due": "2024-06-15", "priority": "high",
	}
	flags := map[string]bool{}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add metadata: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "[due:: 2024-06-15]") {
		t.Errorf("missing due date: %s", content)
	}
	if !strings.Contains(content, "[priority:: high]") {
		t.Errorf("missing priority: %s", content)
	}
}

func TestCmdTasksAdd_Emoji(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("# Tasks\n"), 0644)

	params := map[string]string{
		"file": "Note", "content": "Emoji task",
		"due": "2024-06-15", "priority": "high",
	}
	flags := map[string]bool{"--emoji": true}
	if err := cmdTasksAdd(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:add emoji: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "⏫") {
		t.Errorf("missing priority emoji: %s", content)
	}
	if !strings.Contains(content, "📅 2024-06-15") {
		t.Errorf("missing due emoji: %s", content)
	}
}

func TestCmdTasksEdit_ChangeText(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Old text\n- [ ] Keep this\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1", "content": "New text"}
	flags := map[string]bool{}
	if err := cmdTasksEdit(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:edit: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "- [ ] New text") {
		t.Errorf("text not updated: %s", content)
	}
	if !strings.Contains(content, "- [ ] Keep this") {
		t.Errorf("second task changed: %s", content)
	}
}

func TestCmdTasksEdit_ChangeMetadata(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Task [due:: 2024-01-15] [priority:: low]\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1", "due": "2024-06-01", "priority": "high"}
	flags := map[string]bool{}
	if err := cmdTasksEdit(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:edit metadata: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "[due:: 2024-06-01]") {
		t.Errorf("due not updated: %s", content)
	}
	if !strings.Contains(content, "[priority:: high]") {
		t.Errorf("priority not updated: %s", content)
	}
}

func TestCmdTasksEdit_ByID(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Task A [id:: aaa]\n- [ ] Task B [id:: bbb]\n"), 0644)

	params := map[string]string{"file": "Note", "id": "bbb", "content": "Updated B"}
	flags := map[string]bool{}
	if err := cmdTasksEdit(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:edit by id: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "Updated B") {
		t.Errorf("task B not updated: %s", content)
	}
	if !strings.Contains(content, "Task A") {
		t.Errorf("task A changed: %s", content)
	}
}

func TestCmdTasksEdit_ClearField(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Task [due:: 2024-01-15] [priority:: high]\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1", "due": "-"}
	flags := map[string]bool{}
	if err := cmdTasksEdit(vaultDir, params, flags); err != nil {
		t.Fatalf("tasks:edit clear: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if strings.Contains(content, "[due::") {
		t.Errorf("due not cleared: %s", content)
	}
	if !strings.Contains(content, "[priority:: high]") {
		t.Errorf("priority should remain: %s", content)
	}
}

func TestCmdTasksRemove_ByLine(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Keep\n- [ ] Remove me\n- [ ] Also keep\n"), 0644)

	params := map[string]string{"file": "Note", "line": "2"}
	if err := cmdTasksRemove(vaultDir, params); err != nil {
		t.Fatalf("tasks:remove: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if strings.Contains(content, "Remove me") {
		t.Errorf("task not removed: %s", content)
	}
	if !strings.Contains(content, "Keep") || !strings.Contains(content, "Also keep") {
		t.Errorf("wrong tasks removed: %s", content)
	}
}

func TestCmdTasksRemove_ByID(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Task A [id:: aaa]\n- [ ] Task B [id:: bbb]\n"), 0644)

	params := map[string]string{"file": "Note", "id": "bbb"}
	if err := cmdTasksRemove(vaultDir, params); err != nil {
		t.Fatalf("tasks:remove by id: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if strings.Contains(content, "Task B") {
		t.Errorf("task B not removed: %s", content)
	}
	if !strings.Contains(content, "Task A") {
		t.Errorf("task A removed: %s", content)
	}
}

func TestCmdTasksDone_MarksComplete(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Pending task\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1"}
	if err := cmdTasksDone(vaultDir, params); err != nil {
		t.Fatalf("tasks:done: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "- [x]") {
		t.Errorf("task not marked done: %s", content)
	}
	if !strings.Contains(content, "[completion::") {
		t.Errorf("missing completion date: %s", content)
	}
}

func TestCmdTasksDone_AlreadyDone(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [x] Already done\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1"}
	if err := cmdTasksDone(vaultDir, params); err != nil {
		t.Fatalf("tasks:done already done: %v", err)
	}

	// File should be unchanged
	data, _ := os.ReadFile(note)
	if string(data) != "- [x] Already done\n" {
		t.Errorf("file changed for already done task: %s", string(data))
	}
}

func TestCmdTasksToggle_PendingToDone(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [ ] Pending\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1"}
	if err := cmdTasksToggle(vaultDir, params); err != nil {
		t.Fatalf("tasks:toggle: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "- [x]") {
		t.Errorf("task not toggled to done: %s", content)
	}
	if !strings.Contains(content, "[completion::") {
		t.Errorf("missing completion date: %s", content)
	}
}

func TestCmdTasksToggle_DoneToPending(t *testing.T) {
	vaultDir := t.TempDir()
	note := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(note, []byte("- [x] Done task [completion:: 2024-01-15]\n"), 0644)

	params := map[string]string{"file": "Note", "line": "1"}
	if err := cmdTasksToggle(vaultDir, params); err != nil {
		t.Fatalf("tasks:toggle: %v", err)
	}

	data, _ := os.ReadFile(note)
	content := string(data)
	if !strings.Contains(content, "- [ ]") {
		t.Errorf("task not toggled to pending: %s", content)
	}
	if strings.Contains(content, "[completion::") {
		t.Errorf("completion date not cleared: %s", content)
	}
}

// --- MergeMeta test ---

func TestMergeMeta_ClearField(t *testing.T) {
	meta := taskMeta{Due: "2024-01-15", Priority: "high"}
	mergeMeta(&meta, map[string]string{"due": "-"})
	if meta.Due != "" {
		t.Errorf("due not cleared: %q", meta.Due)
	}
	if meta.Priority != "high" {
		t.Errorf("priority changed: %q", meta.Priority)
	}
}

func TestMergeMeta_UpdateField(t *testing.T) {
	meta := taskMeta{Due: "2024-01-15"}
	mergeMeta(&meta, map[string]string{"due": "2024-06-01", "priority": "low"})
	if meta.Due != "2024-06-01" {
		t.Errorf("due = %q, want 2024-06-01", meta.Due)
	}
	if meta.Priority != "low" {
		t.Errorf("priority = %q, want low", meta.Priority)
	}
}
