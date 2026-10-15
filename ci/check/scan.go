package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// These only point a reviewer at lines worth reading; none of them fails a
// submission, since each has innocent uses. A binary is the exception.
var (
	pipeToShellRe = regexp.MustCompile(`\b(curl|wget)\b[^|\n]*\|\s*(sudo\s+)?(ba|z)?sh\b`)
	privilegeRe   = regexp.MustCompile(`\b(sudo|pkexec|doas)\b`)
	remoteURLRe   = regexp.MustCompile(`https?://[^\s"'<>)]+`)
)

// skipExts are the binary files a plugin page may bundle: images and fonts.
var skipExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".ico": true, ".svg": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true,
}

// minifiedLine is longer than any line a person writes by hand.
const minifiedLine = 1000

func scanFiles(r *report, dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if skipExts[strings.ToLower(filepath.Ext(rel))] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		scanFile(r, filepath.ToSlash(rel), data)
		return nil
	})
}

func scanFile(r *report, rel string, data []byte) {
	if bytes.IndexByte(data[:min(len(data), 8000)], 0) != -1 {
		r.fail("%s: binary file, listed plugins run only scripts a reviewer can read (rule 5)", rel)
		return
	}
	ui := strings.HasPrefix(rel, "ui/")
	for i, line := range strings.Split(string(data), "\n") {
		n := i + 1
		if len(line) > minifiedLine && (strings.HasSuffix(rel, ".js") || strings.HasSuffix(rel, ".css") || strings.HasSuffix(rel, ".mjs")) {
			r.flag("%s:%d: looks minified, its source must be in the repository (rule 5)", rel, n)
			continue
		}
		if pipeToShellRe.MatchString(line) {
			r.flag("%s:%d: downloads and runs code (rule 6)", rel, n)
		}
		if privilegeRe.MatchString(line) {
			r.flag("%s:%d: asks for privilege (rule 8)", rel, n)
		}
		if ui && hasRemoteURL(line) {
			r.flag("%s:%d: loads something from another server (rule 7)", rel, n)
		}
	}
}

func hasRemoteURL(line string) bool {
	for _, u := range remoteURLRe.FindAllString(line, -1) {
		host := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
		if !strings.HasPrefix(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") {
			return true
		}
	}
	return false
}
