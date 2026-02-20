package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// discoverTemplateFolder determines the template folder for a vault.
// Discovery order:
//  1. .obsidian/templates.json -- has a "folder" key
//  2. Default "templates/" directory exists in vault root
//  3. Error: no template folder configured or found
func discoverTemplateFolder(vaultDir string) (string, error) {
	// 1. Try .obsidian/templates.json
	configPath := filepath.Join(vaultDir, ".obsidian", "templates.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var raw map[string]any
		if jsonErr := json.Unmarshal(data, &raw); jsonErr == nil {
			if folder, ok := raw["folder"].(string); ok && folder != "" {
				return folder, nil
			}
		}
	}

	// 2. Fall back to default templates/ directory if it exists
	defaultDir := filepath.Join(vaultDir, "templates")
	if info, err := os.Stat(defaultDir); err == nil && info.IsDir() {
		return "templates", nil
	}

	return "", fmt.Errorf("no template folder configured or found")
}

// templateVarPattern matches {{varname}} and {{varname:format}} patterns.
var templateVarPattern = regexp.MustCompile(`\{\{(date|time|title)(?::([^}]+))?\}\}`)

// substituteTemplateVars replaces known template variables in content.
// Known variables: {{title}}, {{date}}, {{time}}, {{date:FORMAT}}, {{time:FORMAT}}.
// Unknown variables (e.g., {{foo}}) are left as-is.
func substituteTemplateVars(content string, title string, now time.Time) string {
	return templateVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := templateVarPattern.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		varName := sub[1]
		varFormat := sub[2]

		switch varName {
		case "title":
			return title
		case "date":
			if varFormat != "" {
				goFmt := momentToGoFormat(varFormat)
				return now.Format(goFmt)
			}
			return now.Format("2006-01-02")
		case "time":
			if varFormat != "" {
				goFmt := momentToGoFormat(varFormat)
				return now.Format(goFmt)
			}
			return now.Format("15:04")
		default:
			return match
		}
	})
}

// cmdTemplates lists available template files in the configured template folder.
func cmdTemplates(vaultDir string, params map[string]string, format string) error {
	folder, err := discoverTemplateFolder(vaultDir)
	if err != nil {
		return err
	}

	tmplDir := filepath.Join(vaultDir, folder)

	var templates []string
	err = filepath.WalkDir(tmplDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(tmplDir, path)
		templates = append(templates, relPath)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(templates)
	formatList(templates, format)
	return nil
}

// cmdTemplatesApply reads a template file, substitutes variables, and creates
// a new note at the specified path.
func cmdTemplatesApply(vaultDir string, params map[string]string) error {
	templateName := params["template"]
	noteName := params["name"]
	notePath := params["path"]

	if templateName == "" {
		return fmt.Errorf("templates:apply requires template=\"<name>\"")
	}
	if noteName == "" || notePath == "" {
		return fmt.Errorf("templates:apply requires name=\"<title>\" path=\"<path>\"")
	}

	folder, err := discoverTemplateFolder(vaultDir)
	if err != nil {
		return err
	}

	// Resolve template file
	tmplPath := filepath.Join(vaultDir, folder, templateName)
	if !strings.HasSuffix(tmplPath, ".md") {
		tmplPath += ".md"
	}

	tmplData, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("template %q not found in %s", templateName, folder)
	}

	// Check target doesn't already exist
	fullPath := filepath.Join(vaultDir, notePath)
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("note already exists: %s", notePath)
	}

	// Substitute variables
	content := substituteTemplateVars(string(tmplData), noteName, time.Now())

	// Ensure parent directories exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("created: %s (from template %q)\n", notePath, templateName)
	return nil
}
