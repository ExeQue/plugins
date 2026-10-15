package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodCommit = "0123456789abcdef0123456789abcdef01234567"

func write(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func has(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func TestReadEntry(t *testing.T) {
	dir := write(t, t.TempDir(), map[string]string{
		"queue-board.yaml": "repo: https://github.com/acme/queue-board\ncommit: " + goodCommit + "\nauthor: acme\ndescription: Queued jobs\n",
		"typo.yaml":        "repo: https://github.com/acme/x\ncommit: " + goodCommit + "\nauthor: acme\ndescription: x\ncomit: x\n",
	})
	name, e, err := ReadEntry(filepath.Join(dir, "queue-board.yaml"))
	if err != nil || name != "queue-board" || e.Commit != goodCommit {
		t.Fatalf("got %q %+v %v", name, e, err)
	}
	if _, _, err := ReadEntry(filepath.Join(dir, "typo.yaml")); err == nil || !strings.Contains(err.Error(), "comit") {
		t.Errorf("a misspelt key must be named, got %v", err)
	}
}

func TestEntryProblems(t *testing.T) {
	good := Entry{Repo: "https://github.com/acme/qb", Commit: goodCommit, Author: "acme", Description: "d"}
	if p := good.Problems("qb"); len(p) != 0 {
		t.Fatalf("unexpected %v", p)
	}
	cases := map[string]struct {
		name string
		e    Entry
	}{
		"Qb":          {"Qb", good},
		"github.com":  {"qb", Entry{Repo: "https://gitlab.com/acme/qb", Commit: goodCommit, Author: "a", Description: "d"}},
		"40":          {"qb", Entry{Repo: good.Repo, Commit: "main", Author: "a", Description: "d"}},
		"author":      {"qb", Entry{Repo: good.Repo, Commit: goodCommit, Description: "d"}},
		"description": {"qb", Entry{Repo: good.Repo, Commit: goodCommit, Author: "a"}},
	}
	for want, c := range cases {
		if p := c.e.Problems(c.name); !has(p, want) {
			t.Errorf("%+v: problems %v, want one naming %q", c.e, p, want)
		}
	}
}
