// Package registry reads the entries in plugins/ and fetches the commits they
// approve, for the submission check and the index builder alike.
package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one listing in plugins/<name>.yaml.
type Entry struct {
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
	nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// RepoPattern is a var so tests can list a local repository.
	RepoPattern = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitRe    = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// ReadEntry parses an entry strictly, so a misspelt key fails instead of
// silently dropping what it meant to say.
func ReadEntry(path string) (string, Entry, error) {
	name := strings.TrimSuffix(filepath.Base(path), ".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return name, Entry{}, err
	}
	var e Entry
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&e); err != nil {
		return name, Entry{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return name, e, nil
}

// Problems lists what is wrong with the entry itself, before anything is fetched.
func (e Entry) Problems(name string) []string {
	var out []string
	if !nameRe.MatchString(name) {
		out = append(out, fmt.Sprintf("entry name %q must be lowercase letters, digits and dashes", name))
	}
	if !RepoPattern.MatchString(e.Repo) {
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
