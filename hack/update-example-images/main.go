// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Command update-example-images keeps the per-language image in the managed
// Instrumentation CR examples pinned to its canonical, pinnable reference. Image
// versions come from the revision package (the same source the revision tooling
// uses), so examples stay in lockstep with the published image tags.
//
// The tool only *replaces* an image that is already declared as the first child
// of a language block; it never inserts one. New examples must declare an image,
// which the required-image validation and tests enforce. Run via
// `make update-example-images`.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/hack/autoinstrumentation-revision/revision"
)

const (
	operatorImageRepo = "ghcr.io/open-telemetry/opentelemetry-operator"
	goImageRepo       = "ghcr.io/open-telemetry/opentelemetry-go-instrumentation"
)

// dirOverride maps a CR language key to the autoinstrumentation image directory
// the revision package tracks, for keys whose directory differs from the key.
// apacheHttpd and nginx both share the apache-httpd image.
var dirOverride = map[string]string{
	"apacheHttpd": "apache-httpd",
	"nginx":       "apache-httpd",
}

// excludedDocs are docs/auto-instrumentation files that intentionally show
// custom/placeholder images and must not be rewritten.
var excludedDocs = map[string]bool{
	"custom-images.md": true,
}

var imageRe = regexp.MustCompile(`^[ \t]+image:[ \t]`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "update-example-images:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	keys, err := languageKeys()
	if err != nil {
		return err
	}
	headerRe := headerRegex(keys)

	refs, err := resolveRefs(root, keys)
	if err != nil {
		return err
	}

	files, err := managedFiles(root)
	if err != nil {
		return err
	}

	changed := 0
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		content, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		updated := pinImages(string(content), headerRe, refs)
		if updated == string(content) {
			continue
		}
		if err := os.WriteFile(abs, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		fmt.Println("updated", rel)
		changed++
	}

	if changed == 0 {
		fmt.Println("All Instrumentation example images are up to date.")
	}
	return nil
}

// pinImages replaces the value of every language block's first-child image with
// refs[langKey]. A block whose first child is not an image is left untouched -
// the tool never inserts a missing image. Everything else is preserved verbatim,
// including comments, ordering, and trailing-newline state.
func pinImages(content string, headerRe *regexp.Regexp, refs map[string]string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		m := headerRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		childIndent := len(m[1]) + 2
		next := i + 1
		if next < len(lines) && leadingSpaces(lines[next]) == childIndent && imageRe.MatchString(lines[next]) {
			lines[next] = strings.Repeat(" ", childIndent) + "image: " + refs[m[2]]
		}
	}
	return strings.Join(lines, "\n")
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

// languageKeys returns the Instrumentation CR spec field names that carry an
// image, derived from InstrumentationSpec so new languages are picked up
// automatically: a language field is a struct-typed field with an Image field.
func languageKeys() ([]string, error) {
	var keys []string
	for f := range reflect.TypeFor[v1alpha1.InstrumentationSpec]().Fields() {
		if f.Type.Kind() != reflect.Struct {
			continue
		}
		if _, ok := f.Type.FieldByName("Image"); !ok {
			continue
		}
		if name := jsonName(f.Tag.Get("json")); name != "" {
			keys = append(keys, name)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no language fields with an Image found on InstrumentationSpec")
	}
	slices.Sort(keys)
	return keys, nil
}

func jsonName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	return name
}

func headerRegex(keys []string) *regexp.Regexp {
	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = regexp.QuoteMeta(k)
	}
	return regexp.MustCompile(`^([ \t]*)(` + strings.Join(quoted, "|") + `):[ \t]*$`)
}

// resolveRefs builds the canonical image reference for each language key, reusing
// the revision package as the single source of truth for SDK versions and
// operator-owned revisions. go uses the upstream image with no revision.
func resolveRefs(root string, keys []string) (map[string]string, error) {
	repo := revision.New(root)
	refs := make(map[string]string, len(keys))
	for _, key := range keys {
		if key == "go" {
			v, err := goVersion(root)
			if err != nil {
				return nil, err
			}
			refs[key] = fmt.Sprintf("%s/autoinstrumentation-go:%s", goImageRepo, v)
			continue
		}
		dir := key
		if d, ok := dirOverride[key]; ok {
			dir = d
		}
		sdk, err := repo.SDKVersion(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve SDK version for %q: %w", key, err)
		}
		rev, err := repo.Revision(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve revision for %q: %w", key, err)
		}
		refs[key] = fmt.Sprintf("%s/autoinstrumentation-%s:%s-%d", operatorImageRepo, dir, sdk, rev)
	}
	return refs, nil
}

// goVersion reads the upstream go instrumentation version from versions.txt. Go
// references the upstream image directly and has no operator-owned revision.
func goVersion(root string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, "versions.txt"))
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(content), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "autoinstrumentation-go="); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("could not read autoinstrumentation-go version from versions.txt")
}

// managedFiles lists the Instrumentation CR examples this tool pins (repo-relative):
// the e2e installs, the CR sample, and the auto-instrumentation docs. e2e-upgrade
// is excluded on purpose - those CRs pin old images to exercise upgrade-blocking;
// custom-images.md is excluded because its placeholder images are illustrative.
func managedFiles(root string) ([]string, error) {
	set := map[string]struct{}{}
	add := func(path string) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		set[filepath.ToSlash(rel)] = struct{}{}
		return nil
	}

	testRoots := []string{"tests/e2e-instrumentation", "tests/e2e-multi-instrumentation", "tests/e2e"}
	for _, r := range testRoots {
		err := filepath.WalkDir(filepath.Join(root, r), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), "install-instrumentation.yaml") {
				return add(path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	err := filepath.WalkDir(filepath.Join(root, "docs/auto-instrumentation"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") && !excludedDocs[d.Name()] {
			return add(path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, f := range []string{
		"tests/e2e-openshift/must-gather/install-instrumentation.yaml",
		"config/samples/instrumentation_v1alpha1_instrumentation.yaml",
	} {
		set[f] = struct{}{}
	}

	files := make([]string, 0, len(set))
	for f := range set {
		files = append(files, f)
	}
	slices.Sort(files)
	return files, nil
}

// repoRoot walks up from the working directory to the module root (the directory
// containing go.mod), so the tool works regardless of where it is invoked.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (go.mod) from %s", dir)
		}
		dir = parent
	}
}
