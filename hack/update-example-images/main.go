// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Command update-example-images keeps the per-language image in the managed
// Instrumentation CR examples pinned to the latest revision tag.
package main

import (
	"errors"
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

var dirOverride = map[string]string{
	"apacheHttpd": "apache-httpd",
	"nginx":       "apache-httpd",
}

var scanRoots = []string{"tests", "docs", "config"}

var managedExts = []string{".yaml", ".yml", ".md"}

var excludedPaths = map[string]bool{
	"tests/e2e-upgrade": true, // pins old images to exercise upgrade-blocking
	"docs/rfcs":         true, // illustrative versions in design docs
	"docs/auto-instrumentation/custom-images.md": true, // deliberate placeholder images
}

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

	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer rootFS.Close()

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
		content, err := rootFS.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		updated := pinImages(string(content), headerRe, refs)
		if updated == string(content) {
			continue
		}
		if err := rootFS.WriteFile(rel, []byte(updated), 0o600); err != nil {
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

// pinImages returns content with each language block's first-child image line
// rewritten to refs[langKey]. A block whose first child is not an image is left
// untouched: the tool replaces existing images but never inserts a missing one.
func pinImages(content string, headerRe *regexp.Regexp, refs map[string]string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		m := headerRe.FindStringSubmatch(line)
		if m == nil || i+1 >= len(lines) {
			continue
		}
		// The block's first child is an image line one indent level (two spaces)
		// deeper than the header; leave anything else untouched.
		prefix := m[1] + "  image: "
		if strings.HasPrefix(lines[i+1], prefix) {
			lines[i+1] = prefix + refs[m[2]]
		}
	}
	return strings.Join(lines, "\n")
}

// languageKeys returns the Instrumentation CR spec field names that carry an
// image, derived by reflection from InstrumentationSpec so new languages are
// picked up automatically. A language field is a struct-typed field with an
// Image field.
func languageKeys() ([]string, error) {
	var keys []string
	for f := range reflect.TypeFor[v1alpha1.InstrumentationSpec]().Fields() {
		if f.Type.Kind() != reflect.Struct {
			continue
		}
		if _, ok := f.Type.FieldByName("Image"); !ok {
			continue
		}
		// The json tag (e.g. "java,omitempty") gives the CR spec field name.
		if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("no language fields with an Image found on InstrumentationSpec")
	}
	slices.Sort(keys)
	return keys, nil
}

// headerRegex builds a regexp matching an indented `<language>:` block header
// for any of keys, capturing the leading indentation and the matched key.
func headerRegex(keys []string) *regexp.Regexp {
	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = regexp.QuoteMeta(k)
	}
	return regexp.MustCompile(`^([ \t]*)(` + strings.Join(quoted, "|") + `):[ \t]*$`)
}

// resolveRefs returns the canonical image reference for each language key,
// sourcing SDK versions and revisions from the revision package. go uses the
// upstream image and has no operator-owned revision.
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

// goVersion returns the upstream go instrumentation version from versions.txt.
// Go references the upstream image directly and has no operator-owned revision.
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
	return "", errors.New("could not read autoinstrumentation-go version from versions.txt")
}

// managedFiles returns the repo-relative example files to pin: every YAML or
// Markdown file under scanRoots that is not under excludedPaths.
func managedFiles(root string) ([]string, error) {
	var files []string
	for _, r := range scanRoots {
		err := filepath.WalkDir(filepath.Join(root, r), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if excludedPaths[rel] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() && hasManagedExt(d.Name()) {
				files = append(files, rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.Sort(files)
	return files, nil
}

// hasManagedExt reports whether name has an extension that may hold an
// Instrumentation CR.
func hasManagedExt(name string) bool {
	for _, ext := range managedExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// repoRoot returns the module root by walking up from the working directory to
// the directory containing go.mod, so the tool works regardless of where it is
// invoked.
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
