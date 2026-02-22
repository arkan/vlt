package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// taskMeta holds parsed metadata from Dataview inline fields or Tasks emoji format.
type taskMeta struct {
	Due          string `json:"due,omitempty"`
	Scheduled    string `json:"scheduled,omitempty"`
	Start        string `json:"start,omitempty"`
	Created      string `json:"created,omitempty"`
	Completion   string `json:"completion,omitempty"`
	Cancelled    string `json:"cancelled,omitempty"`
	Priority     string `json:"priority,omitempty"`
	Repeat       string `json:"repeat,omitempty"`
	OnCompletion string `json:"onCompletion,omitempty"`
	ID           string `json:"id,omitempty"`
	DependsOn    string `json:"dependsOn,omitempty"`
}

// task represents a parsed checkbox item from a note.
type task struct {
	Text      string   `json:"text"`                // task text after the checkbox (raw, with metadata)
	CleanText string   `json:"cleanText,omitempty"` // text without metadata annotations
	Done      bool     `json:"done"`                // true if [x] or [X]
	Line      int      `json:"line"`                // 1-based line number
	File      string   `json:"file"`                // relative path (when searching vault-wide)
	Meta      taskMeta `json:"meta,omitempty"`       // parsed metadata
	isEmoji   bool     // detected format (unexported)
	indent    string   // leading whitespace (unexported)
}

// taskPattern matches markdown checkboxes: - [ ] text or - [x] text
// Allows leading whitespace/tabs for nesting.
var taskPattern = regexp.MustCompile(`(?m)^[\t ]*- \[([ xX])\] (.+)$`)

// dataviewFieldPattern matches Dataview inline fields: [key:: value]
var dataviewFieldPattern = regexp.MustCompile(`\[(\w+)::\s*([^\]]*)\]`)

// emojiDatePattern matches emoji signifiers followed by a value.
var emojiDatePattern = regexp.MustCompile(
	`([\x{2795}\x{23f3}\x{1f6eb}\x{1f4c5}\x{2705}\x{274c}\x{1f501}\x{1f3c1}\x{1f194}\x{26d4}])\s*(\S+)`,
)

// emojiPriorityPattern matches standalone priority emoji signifiers.
var emojiPriorityPattern = regexp.MustCompile(
	`[\x{23ec}\x{1f53d}\x{1f53c}\x{23eb}\x{1f53a}]`,
)

// emojiToField maps emoji signifiers to taskMeta field names.
var emojiToField = map[string]string{
	"\u2795":     "created",      // ➕
	"\u23f3":     "scheduled",    // ⏳
	"\U0001f6eb": "start",        // 🛫
	"\U0001f4c5": "due",          // 📅
	"\u2705":     "completion",   // ✅
	"\u274c":     "cancelled",    // ❌
	"\U0001f501": "repeat",       // 🔁
	"\U0001f3c1": "onCompletion", // 🏁
	"\U0001f194": "id",           // 🆔
	"\u26d4":     "dependsOn",    // ⛔
}

// emojiToPriorityMap maps priority emojis to priority names.
var emojiToPriorityMap = map[string]string{
	"\u23ec":     "lowest",  // ⏬
	"\U0001f53d": "low",     // 🔽
	"\U0001f53c": "medium",  // 🔼
	"\u23eb":     "high",    // ⏫
	"\U0001f53a": "highest", // 🔺
}

// priorityToEmojiMap maps priority names to emojis.
var priorityToEmojiMap = map[string]string{
	"lowest":  "\u23ec",     // ⏬
	"low":     "\U0001f53d", // 🔽
	"medium":  "\U0001f53c", // 🔼
	"high":    "\u23eb",     // ⏫
	"highest": "\U0001f53a", // 🔺
}

// parseTasks extracts all checkbox items from text.
func parseTasks(text string) []task {
	lines := strings.Split(text, "\n")
	var tasks []task

	for i, line := range lines {
		m := taskPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rawText := m[2]
		cleanText, meta, isEmoji := parseTaskMeta(rawText)

		// Detect leading indentation
		indent := ""
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent += string(ch)
			} else {
				break
			}
		}

		tasks = append(tasks, task{
			Text:      rawText,
			CleanText: cleanText,
			Done:      m[1] == "x" || m[1] == "X",
			Line:      i + 1,
			Meta:      meta,
			isEmoji:   isEmoji,
			indent:    indent,
		})
	}
	return tasks
}

// parseTaskMeta extracts metadata from the text after the checkbox.
// Tries Dataview format first ([key:: value]), then emoji format.
// Returns the clean text (without metadata), the parsed meta, and whether emoji format was detected.
func parseTaskMeta(rawText string) (string, taskMeta, bool) {
	meta := taskMeta{}
	clean := rawText

	// Try Dataview format: [key:: value]
	dvMatches := dataviewFieldPattern.FindAllStringSubmatch(rawText, -1)
	if len(dvMatches) > 0 {
		for _, m := range dvMatches {
			setMetaField(&meta, m[1], strings.TrimSpace(m[2]))
			clean = strings.Replace(clean, m[0], "", 1)
		}
		clean = strings.TrimSpace(clean)
		// Collapse multiple spaces left by removal
		for strings.Contains(clean, "  ") {
			clean = strings.ReplaceAll(clean, "  ", " ")
		}
		return clean, meta, false
	}

	// Try emoji format
	found := false

	// Extract priority emojis (no value, just emoji)
	if loc := emojiPriorityPattern.FindStringIndex(rawText); loc != nil {
		emoji := rawText[loc[0]:loc[1]]
		if p, ok := emojiToPriorityMap[emoji]; ok {
			meta.Priority = p
			clean = clean[:loc[0]] + clean[loc[1]:]
			found = true
		}
	}

	// Extract emoji+value fields (emoji followed by value)
	for {
		eLoc := emojiDatePattern.FindStringSubmatchIndex(clean)
		if eLoc == nil {
			break
		}
		emoji := clean[eLoc[2]:eLoc[3]]
		value := clean[eLoc[4]:eLoc[5]]
		if field, ok := emojiToField[emoji]; ok {
			setMetaField(&meta, field, value)
			clean = clean[:eLoc[0]] + clean[eLoc[1]:]
			found = true
		} else {
			break
		}
	}

	if found {
		clean = strings.TrimSpace(clean)
		for strings.Contains(clean, "  ") {
			clean = strings.ReplaceAll(clean, "  ", " ")
		}
		return clean, meta, true
	}

	return rawText, meta, false
}

// setMetaField maps a key name to the corresponding taskMeta field.
func setMetaField(m *taskMeta, key, value string) {
	switch strings.ToLower(key) {
	case "due":
		m.Due = value
	case "scheduled":
		m.Scheduled = value
	case "start":
		m.Start = value
	case "created":
		m.Created = value
	case "completion":
		m.Completion = value
	case "cancelled":
		m.Cancelled = value
	case "priority":
		m.Priority = value
	case "repeat":
		m.Repeat = value
	case "oncompletion":
		m.OnCompletion = value
	case "id":
		m.ID = value
	case "dependson":
		m.DependsOn = value
	}
}

// cmdTasks lists tasks (checkboxes) from one note or across the vault.
// Supports filters: done (only completed), pending (only incomplete).
// Supports path= to limit search to a subfolder.
func cmdTasks(vaultDir string, params map[string]string, flags map[string]bool) error {
	format := outputFormat(flags)
	filterDone := flags["done"]
	filterPending := flags["pending"]

	title := params["file"]
	pathFilter := params["path"]

	// Single file mode
	if title != "" {
		path, err := resolveNote(vaultDir, title)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(vaultDir, path)
		tasks := parseTasks(string(data))
		tasks = filterTasks(tasks, filterDone, filterPending)

		for i := range tasks {
			tasks[i].File = relPath
		}

		outputTasks(tasks, format)
		return nil
	}

	// Vault-wide mode
	searchRoot := vaultDir
	if pathFilter != "" {
		searchRoot = filepath.Join(vaultDir, pathFilter)
		if _, err := os.Stat(searchRoot); os.IsNotExist(err) {
			return fmt.Errorf("path filter %q not found in vault", pathFilter)
		}
	}

	var allTasks []task

	err := filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == ".trash") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(name, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(vaultDir, path)
		tasks := parseTasks(string(data))

		for i := range tasks {
			tasks[i].File = relPath
		}

		allTasks = append(allTasks, tasks...)
		return nil
	})

	if err != nil {
		return err
	}

	allTasks = filterTasks(allTasks, filterDone, filterPending)
	outputTasks(allTasks, format)
	return nil
}

// filterTasks applies done/pending filters.
func filterTasks(tasks []task, done, pending bool) []task {
	if !done && !pending {
		return tasks
	}

	var result []task
	for _, t := range tasks {
		if done && t.Done {
			result = append(result, t)
		}
		if pending && !t.Done {
			result = append(result, t)
		}
	}
	return result
}

// outputTasks prints tasks in the requested format.
func outputTasks(tasks []task, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(tasks)
		fmt.Println(string(data))
	case "csv":
		fmt.Println("done,text,line,file")
		for _, t := range tasks {
			done := "false"
			if t.Done {
				done = "true"
			}
			fmt.Printf("%s,%q,%d,%s\n", done, t.Text, t.Line, t.File)
		}
	case "yaml":
		for _, t := range tasks {
			fmt.Printf("- text: %s\n  done: %v\n  line: %d\n  file: %s\n", yamlEscapeValue(t.Text), t.Done, t.Line, t.File)
		}
	default:
		for _, t := range tasks {
			check := " "
			if t.Done {
				check = "x"
			}
			fmt.Printf("- [%s] %s (%s:%d)\n", check, t.Text, t.File, t.Line)
		}
	}
}

// buildTaskLine constructs a full markdown task line.
// If emoji is true, writes emoji format; otherwise writes Dataview format.
func buildTaskLine(indent string, done bool, text string, meta taskMeta, emoji bool) string {
	check := " "
	if done {
		check = "x"
	}
	var sb strings.Builder
	sb.WriteString(indent)
	sb.WriteString("- [")
	sb.WriteString(check)
	sb.WriteString("] ")
	sb.WriteString(text)

	if emoji {
		appendEmojiMeta(&sb, meta)
	} else {
		appendDataviewMeta(&sb, meta)
	}

	return sb.String()
}

// appendDataviewMeta appends [key:: value] fields to the string builder.
func appendDataviewMeta(sb *strings.Builder, m taskMeta) {
	fields := []struct{ key, val string }{
		{"due", m.Due}, {"scheduled", m.Scheduled}, {"start", m.Start},
		{"created", m.Created}, {"completion", m.Completion}, {"cancelled", m.Cancelled},
		{"priority", m.Priority}, {"repeat", m.Repeat},
		{"onCompletion", m.OnCompletion}, {"id", m.ID}, {"dependsOn", m.DependsOn},
	}
	for _, f := range fields {
		if f.val != "" {
			sb.WriteString(" [")
			sb.WriteString(f.key)
			sb.WriteString(":: ")
			sb.WriteString(f.val)
			sb.WriteString("]")
		}
	}
}

// appendEmojiMeta appends emoji-format metadata to the string builder.
func appendEmojiMeta(sb *strings.Builder, m taskMeta) {
	// Priority emoji (no value, just the emoji)
	if m.Priority != "" {
		if e, ok := priorityToEmojiMap[m.Priority]; ok {
			sb.WriteString(" ")
			sb.WriteString(e)
		}
	}
	// Date/value fields
	dateFields := []struct{ emoji, val string }{
		{"\U0001f4c5", m.Due},          // 📅
		{"\u23f3", m.Scheduled},        // ⏳
		{"\U0001f6eb", m.Start},        // 🛫
		{"\u2795", m.Created},          // ➕
		{"\u2705", m.Completion},       // ✅
		{"\u274c", m.Cancelled},        // ❌
		{"\U0001f501", m.Repeat},       // 🔁
		{"\U0001f3c1", m.OnCompletion}, // 🏁
		{"\U0001f194", m.ID},           // 🆔
		{"\u26d4", m.DependsOn},        // ⛔
	}
	for _, f := range dateFields {
		if f.val != "" {
			sb.WriteString(" ")
			sb.WriteString(f.emoji)
			sb.WriteString(" ")
			sb.WriteString(f.val)
		}
	}
}

// metaFromParams extracts task metadata from CLI parameters.
func metaFromParams(params map[string]string) taskMeta {
	return taskMeta{
		Due:          params["due"],
		Scheduled:    params["scheduled"],
		Start:        params["start"],
		Created:      params["created"],
		Completion:   params["completion"],
		Cancelled:    params["cancelled"],
		Priority:     params["priority"],
		Repeat:       params["repeat"],
		OnCompletion: params["onCompletion"],
		ID:           params["id"],
		DependsOn:    params["dependsOn"],
	}
}

// mergeMeta updates existing meta fields from params.
// A value of "-" clears the field.
func mergeMeta(meta *taskMeta, params map[string]string) {
	mergeField := func(field *string, key string) {
		if v, ok := params[key]; ok {
			if v == "-" {
				*field = ""
			} else {
				*field = v
			}
		}
	}
	mergeField(&meta.Due, "due")
	mergeField(&meta.Scheduled, "scheduled")
	mergeField(&meta.Start, "start")
	mergeField(&meta.Created, "created")
	mergeField(&meta.Completion, "completion")
	mergeField(&meta.Cancelled, "cancelled")
	mergeField(&meta.Priority, "priority")
	mergeField(&meta.Repeat, "repeat")
	mergeField(&meta.OnCompletion, "onCompletion")
	mergeField(&meta.ID, "id")
	mergeField(&meta.DependsOn, "dependsOn")
}

// resolveTask finds a task in a file by ID, line number, or text match.
// Returns the task and its 0-based line index.
func resolveTask(lines []string, params map[string]string) (task, int, error) {
	tasks := parseTasks(strings.Join(lines, "\n"))

	// Priority 1: by Dataview ID
	if id := params["id"]; id != "" {
		for _, t := range tasks {
			if t.Meta.ID == id {
				return t, t.Line - 1, nil
			}
		}
		return task{}, 0, fmt.Errorf("task with id=%q not found", id)
	}

	// Priority 2: by line number
	if lineSpec := params["line"]; lineSpec != "" {
		lineNum, err := parseInt(lineSpec)
		if err != nil {
			return task{}, 0, fmt.Errorf("invalid line number: %s", lineSpec)
		}
		for _, t := range tasks {
			if t.Line == lineNum {
				return t, t.Line - 1, nil
			}
		}
		return task{}, 0, fmt.Errorf("no task at line %d", lineNum)
	}

	// Priority 3: by text match
	if match := params["match"]; match != "" {
		matchLower := strings.ToLower(match)
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Text), matchLower) {
				return t, t.Line - 1, nil
			}
		}
		return task{}, 0, fmt.Errorf("no task matching %q", match)
	}

	return task{}, 0, fmt.Errorf("task identification required: id=, line=, or match=")
}

// cmdTasksAdd adds a new task to a note.
// Supports positioning: heading= (with section="start"|"end"), line=, or end of file.
func cmdTasksAdd(vaultDir string, params map[string]string, flags map[string]bool) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("tasks:add requires file=\"<title>\"")
	}
	content := params["content"]
	if content == "" {
		content = readStdinIfPiped()
	}
	if content == "" {
		return fmt.Errorf("tasks:add requires content=\"<text>\" or stdin")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Build metadata from params
	meta := metaFromParams(params)

	// Auto-fill created date if not provided
	if meta.Created == "" {
		meta.Created = time.Now().Format("2006-01-02")
	}

	emoji := flags["--emoji"]
	taskLine := buildTaskLine("", false, content, meta, emoji)

	lines := strings.Split(string(data), "\n")

	// Determine insertion point
	insertIdx := len(lines) // default: end of file

	if heading := params["heading"]; heading != "" {
		bounds, found := findSection(lines, heading)
		if !found {
			return fmt.Errorf("heading %q not found", heading)
		}
		section := params["section"]
		if section == "start" {
			insertIdx = bounds.ContentStart
		} else {
			// "end" or default: end of section
			insertIdx = bounds.ContentEnd
		}
	} else if lineSpec := params["line"]; lineSpec != "" {
		lineNum, parseErr := parseInt(lineSpec)
		if parseErr != nil {
			return fmt.Errorf("invalid line number: %s", lineSpec)
		}
		insertIdx = lineNum - 1 // 1-based to 0-based
		if insertIdx < 0 {
			insertIdx = 0
		}
		if insertIdx > len(lines) {
			insertIdx = len(lines)
		}
	}

	// Insert the task line
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:insertIdx]...)
	result = append(result, taskLine)
	result = append(result, lines[insertIdx:]...)

	output := strings.Join(result, "\n")

	if timestampsEnabled(flags["timestamps"]) {
		output = ensureTimestamps(output, false, time.Now())
	}

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(vaultDir, path)
	fmt.Printf("added task in %s at line %d\n", relPath, insertIdx+1)
	return nil
}

// cmdTasksEdit modifies an existing task's text, metadata, or status.
func cmdTasksEdit(vaultDir string, params map[string]string, flags map[string]bool) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("tasks:edit requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	t, lineIdx, err := resolveTask(lines, params)
	if err != nil {
		return err
	}

	// Update text if content= provided
	newText := t.CleanText
	if content, ok := params["content"]; ok {
		newText = content
	}

	// Merge metadata
	newMeta := t.Meta
	mergeMeta(&newMeta, params)

	// Update done status if status= provided
	newDone := t.Done
	if status, ok := params["status"]; ok {
		switch status {
		case "done", "x":
			newDone = true
			if newMeta.Completion == "" {
				newMeta.Completion = time.Now().Format("2006-01-02")
			}
		case "pending", "todo":
			newDone = false
			newMeta.Completion = ""
		}
	}

	// Preserve original format unless --emoji is specified
	emoji := t.isEmoji
	if flags["--emoji"] {
		emoji = true
	}
	if flags["--dataview"] {
		emoji = false
	}

	newLine := buildTaskLine(t.indent, newDone, newText, newMeta, emoji)
	lines[lineIdx] = newLine

	output := strings.Join(lines, "\n")

	if timestampsEnabled(flags["timestamps"]) {
		output = ensureTimestamps(output, false, time.Now())
	}

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(vaultDir, path)
	fmt.Printf("edited task at %s:%d\n", relPath, lineIdx+1)
	return nil
}

// cmdTasksRemove removes a task line from a note.
func cmdTasksRemove(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("tasks:remove requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	_, lineIdx, err := resolveTask(lines, params)
	if err != nil {
		return err
	}

	result := append(lines[:lineIdx], lines[lineIdx+1:]...)
	output := strings.Join(result, "\n")

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(vaultDir, path)
	fmt.Printf("removed task from %s:%d\n", relPath, lineIdx+1)
	return nil
}

// cmdTasksDone marks a task as completed and sets the completion date.
func cmdTasksDone(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("tasks:done requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	t, lineIdx, err := resolveTask(lines, params)
	if err != nil {
		return err
	}

	if t.Done {
		relPath, _ := filepath.Rel(vaultDir, path)
		fmt.Printf("task already done at %s:%d\n", relPath, lineIdx+1)
		return nil
	}

	meta := t.Meta
	meta.Completion = time.Now().Format("2006-01-02")
	newLine := buildTaskLine(t.indent, true, t.CleanText, meta, t.isEmoji)
	lines[lineIdx] = newLine

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(vaultDir, path)
	fmt.Printf("done: %s:%d\n", relPath, lineIdx+1)
	return nil
}

// cmdTasksToggle toggles a task between done and pending.
func cmdTasksToggle(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("tasks:toggle requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	t, lineIdx, err := resolveTask(lines, params)
	if err != nil {
		return err
	}

	newDone := !t.Done
	meta := t.Meta
	if newDone {
		meta.Completion = time.Now().Format("2006-01-02")
	} else {
		meta.Completion = ""
	}

	newLine := buildTaskLine(t.indent, newDone, t.CleanText, meta, t.isEmoji)
	lines[lineIdx] = newLine

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(vaultDir, path)
	status := "pending"
	if newDone {
		status = "done"
	}
	fmt.Printf("toggled to %s: %s:%d\n", status, relPath, lineIdx+1)
	return nil
}
