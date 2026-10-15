package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// entry is one listing in plugins/<name>.yaml.
type entry struct {
	Repo        string   `yaml:"repo"`
	Commit      string   `yaml:"commit"`
	Author      string   `yaml:"author"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	// Images are paths inside the plugin repository at the approved commit,
	// the first one being the preview.
	Images []string `yaml:"images,omitempty"`
}

var (
	nameRe   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	repoRe   = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// readEntry parses an entry strictly, so a misspelt key fails instead of
// silently dropping what it meant to say.
func readEntry(path string) (string, entry, error) {
	name := strings.TrimSuffix(filepath.Base(path), ".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return name, entry{}, err
	}
	var e entry
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&e); err != nil {
		return name, entry{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return name, e, nil
}

func (e entry) problems(name string) []string {
	var out []string
	if !nameRe.MatchString(name) {
		out = append(out, fmt.Sprintf("entry name %q must be lowercase letters, digits and dashes", name))
	}
	if !repoRe.MatchString(e.Repo) {
		out = append(out, fmt.Sprintf("repo %q must be https://github.com/<owner>/<repo>", e.Repo))
	}
	if !commitRe.MatchString(e.Commit) {
		out = append(out, fmt.Sprintf("commit %q must be the full 40-character hash", e.Commit))
	}
	if strings.TrimSpace(e.Author) == "" {
		out = append(out, "author is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		out = append(out, "description is required")
	}
	return out
}
