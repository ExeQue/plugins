package main

import (
	"github.com/lerd-env/plugins/ci/registry"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// gitRepo commits files into a fresh repository and returns its file:// URL
// and the commit.
func gitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	dir := write(t, t.TempDir(), files)
	run := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	run("add", ".")
	run("commit", "-q", "-m", "plugin")
	return "file://" + dir, run("rev-parse", "HEAD")
}

func pluginFiles(name, service string) map[string]string {
	return map[string]string{
		"plugin.yaml":                   "api_version: 1\nname: " + name + "\nlerd: \">=1.0\"\n",
		"README.md":                     "Runs ghcr.io/acme/" + service + " published by Acme.\n",
		"LICENSE":                       "MIT\n",
		"services/" + service + ".yaml": "name: " + service + "\nimage: ghcr.io/acme/" + service + ":1\n",
	}
}

// Drives the real lerd binary, so it runs only when LERD_BIN names one.
func TestCheckEntry_InstallsAlongsideTheListedPlugins(t *testing.T) {
	lerd := os.Getenv("LERD_BIN")
	if lerd == "" {
		t.Skip("set LERD_BIN to a lerd binary")
	}
	orig := publicCheck
	publicCheck = func(string) error { return nil }
	t.Cleanup(func() { publicCheck = orig })

	plugins := t.TempDir()
	listedRepo, listedCommit := gitRepo(t, pluginFiles("listed", "queuebox"))
	write(t, plugins, map[string]string{"listed.yaml": "repo: " + listedRepo + "\ncommit: " + listedCommit + "\nauthor: a\ndescription: d\n"})

	entryFor := func(name, service string) string {
		repo, commit := gitRepo(t, pluginFiles(name, service))
		write(t, plugins, map[string]string{name + ".yaml": "repo: " + repo + "\ncommit: " + commit + "\nauthor: a\ndescription: d\n"})
		return filepath.Join(plugins, name+".yaml")
	}
	// The repo check wants github.com; these entries are local, so only the
	// steps after it are judged.
	origRepo := registry.RepoPattern
	registry.RepoPattern = regexpAny
	t.Cleanup(func() { registry.RepoPattern = origRepo })

	if _, r := checkEntry(entryFor("fresh", "jobbox"), plugins, lerd); len(r.Fail) != 0 || len(r.Flag) != 0 {
		t.Errorf("a plugin with names of its own passes, got %+v", r)
	}
	_, r := checkEntry(entryFor("copycat", "queuebox"), plugins, lerd)
	if !has(r.Fail, "queuebox") {
		t.Errorf("a service another listed plugin ships must block, got %+v", r)
	}
}

var regexpAny = regexp.MustCompile(`.`)
