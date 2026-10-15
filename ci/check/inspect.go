package main

import (
	"fmt"
	"github.com/lerd-env/plugins/ci/registry"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// report holds what blocks a submission and what a reviewer should look at.
type report struct {
	Fail []string
	Flag []string
}

func (r *report) fail(format string, a ...any) { r.Fail = append(r.Fail, fmt.Sprintf(format, a...)) }
func (r *report) flag(format string, a ...any) { r.Flag = append(r.Flag, fmt.Sprintf(format, a...)) }

// inspect checks a plugin checked out at its approved commit in dir.
func inspect(dir, name string, e registry.Entry) report {
	var r report
	if !anyFile(dir, "LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING") {
		r.fail("no LICENSE file (rule 2)")
	}
	readme, ok := readFirst(dir, "README.md", "README", "readme.md")
	if !ok {
		r.fail("no README (rule 3)")
	}
	checkManifest(&r, dir, name)
	checkServices(&r, dir, readme)
	checkImages(&r, dir, e.Images)
	scanFiles(&r, dir)
	return r
}

// lowerBoundRe accepts only a lower bound, so a new lerd never strands the plugin.
var lowerBoundRe = regexp.MustCompile(`^>=\s*\d+\.\d+(\.\d+)?$`)

func checkManifest(r *report, dir, name string) {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		r.fail("no plugin.yaml")
		return
	}
	var m struct {
		Name string `yaml:"name"`
		Lerd string `yaml:"lerd"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		r.fail("plugin.yaml: %v", err)
		return
	}
	if m.Name != name {
		r.fail("plugin.yaml name %q must match the entry name %q", m.Name, name)
	}
	switch {
	case m.Lerd == "":
		r.fail("plugin.yaml must declare lerd, the oldest release it runs on (rule 4)")
	case !lowerBoundRe.MatchString(strings.TrimSpace(m.Lerd)):
		r.fail("plugin.yaml lerd %q must be a lower bound only, like \">=1.37\" (rule 4)", m.Lerd)
	}
}

func checkServices(r *report, dir, readme string) {
	paths, _ := filepath.Glob(filepath.Join(dir, "services", "*.yaml"))
	for _, path := range paths {
		rel := "services/" + filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			r.fail("%s: %v", rel, err)
			continue
		}
		var svc struct {
			Image    string           `yaml:"image"`
			Versions []serviceVersion `yaml:"versions"`
		}
		if err := yaml.Unmarshal(data, &svc); err != nil {
			r.fail("%s: %v", rel, err)
			continue
		}
		seen := map[string]bool{"": true}
		for _, img := range append([]string{svc.Image}, versionImages(svc.Versions)...) {
			if !seen[img] {
				seen[img] = true
				checkImage(r, rel, img, readme)
			}
		}
	}
}

type serviceVersion struct {
	Image string `yaml:"image"`
}

func versionImages(vs []serviceVersion) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Image)
	}
	return out
}

func checkImage(r *report, rel, image, readme string) {
	repo, tag := splitImage(image)
	switch {
	case tag == "":
		r.fail("%s: image %s has no tag, pin a version (rule 17)", rel, image)
	case tag == "latest":
		r.fail("%s: image %s uses latest, pin a version (rule 17)", rel, image)
	}
	if !strings.Contains(readme, repo) {
		r.fail("%s: the README must name the image %s and who publishes it (rule 16)", rel, repo)
	}
}

// splitImage separates an image reference into its repository and its tag or
// digest; a port in the registry host is not a tag.
func splitImage(image string) (repo, tag string) {
	if i := strings.Index(image, "@"); i != -1 {
		return image[:i], image[i+1:]
	}
	slash := strings.LastIndex(image, "/")
	if i := strings.LastIndex(image, ":"); i > slash {
		return image[:i], image[i+1:]
	}
	return image, ""
}

var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}

func checkImages(r *report, dir string, images []string) {
	for _, img := range images {
		if !imageExts[strings.ToLower(filepath.Ext(img))] {
			r.fail("image %s must be a png, jpg or webp", img)
			continue
		}
		if !insideFile(dir, img) {
			r.fail("image %s is not in the repository at this commit", img)
		}
	}
}

func insideFile(dir, rel string) bool {
	abs := filepath.Join(dir, rel)
	if !strings.HasPrefix(abs, filepath.Clean(dir)+string(os.PathSeparator)) {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && info.Mode().IsRegular()
}

func anyFile(dir string, names ...string) bool {
	_, ok := readFirst(dir, names...)
	return ok
}

func readFirst(dir string, names ...string) (string, bool) {
	for _, n := range names {
		if data, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
			return string(data), true
		}
	}
	return "", false
}
