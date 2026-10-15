package registry

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FetchCommit checks out exactly commit, the way lerd installs it, so the
// checks read what users would get.
func FetchCommit(repo, commit string) (string, error) {
	dir, err := os.MkdirTemp("", "plugin-")
	if err != nil {
		return "", err
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"fetch", "-q", "--depth", "1", repo, commit},
		{"checkout", "-q", "FETCH_HEAD"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("fetching %s at %s: %s", repo, commit, strings.TrimSpace(string(out)))
		}
	}
	return dir, nil
}
