// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Command update-example-images pins each per-language image in the managed
// Instrumentation CR examples to the reference from
// hack/autoinstrumentation-revision.sh. Run via `make update-example-images`.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const revisionScript = "hack/autoinstrumentation-revision.sh"

// langKeys are the Instrumentation CR spec field names that carry an image.
var langKeys = []string{"java", "nodejs", "python", "dotnet", "go", "apacheHttpd", "nginx"}

var (
	headerRe = regexp.MustCompile(`^([ \t]*)(java|nodejs|python|dotnet|go|apacheHttpd|nginx):[ \t]*$`)
	imageRe  = regexp.MustCompile(`^[ \t]+image:[ \t]`)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "update-example-images:", err)
		os.Exit(1)
	}
}

func run() error {
	refs, err := resolveRefs()
	if err != nil {
		return err
	}

	files, err := managedFiles()
	if err != nil {
		return err
	}

	changed := 0
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		// Nothing to pin in an SDK-only example that declares no language block.
		if !headerRe.Match(content) {
			continue
		}
		updated := pinImages(string(content), refs)
		if updated == string(content) {
			continue
		}
		if err := os.WriteFile(f, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", f, err)
		}
		fmt.Println("updated", f)
		changed++
	}

	if changed == 0 {
		fmt.Println("All Instrumentation example images are up to date.")
	}
	return nil
}

// pinImages rewrites content so that every per-language block has its `image`
// set to refs[langKey] as the first child of the block. Any pre-existing
// direct-child image line in the block is replaced. Everything else is
// preserved verbatim, including comments, ordering, and trailing-newline state.
func pinImages(content string, refs map[string]string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines)+len(langKeys))

	inBlock := false
	blockIndent, childIndent := 0, 0

	for _, line := range lines {
		if inBlock {
			if strings.TrimSpace(line) == "" {
				out = append(out, line)
				continue
			}
			indent := leadingSpaces(line)
			if indent > blockIndent {
				// Drop an existing direct-child image so we can rewrite it.
				if indent == childIndent && imageRe.MatchString(line) {
					continue
				}
				out = append(out, line)
				continue
			}
			inBlock = false
		}

		if m := headerRe.FindStringSubmatch(line); m != nil {
			blockIndent = len(m[1])
			childIndent = blockIndent + 2
			out = append(out, line)
			out = append(out, strings.Repeat(" ", childIndent)+"image: "+refs[m[2]])
			inBlock = true
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

// resolveRefs asks the canonical revision script for each language's image so
// there is a single source of truth for the pinned versions.
func resolveRefs() (map[string]string, error) {
	refs := make(map[string]string, len(langKeys))
	for _, key := range langKeys {
		out, err := exec.Command(revisionScript, "image", key).Output()
		if err != nil {
			return nil, fmt.Errorf("resolve image for %q via %s: %w", key, revisionScript, err)
		}
		ref := strings.TrimSpace(string(out))
		if ref == "" {
			return nil, fmt.Errorf("empty image reference for %q from %s", key, revisionScript)
		}
		refs[key] = ref
	}
	return refs, nil
}

// managedFiles lists the Instrumentation CR examples this tool pins: the e2e
// installs, the CR sample, and the copy-paste getting-started docs. e2e-upgrade
// is excluded on purpose - those CRs pin old images to exercise upgrade-blocking.
func managedFiles() ([]string, error) {
	set := map[string]struct{}{}

	roots := []string{"tests/e2e-instrumentation", "tests/e2e-multi-instrumentation", "tests/e2e"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), "install-instrumentation.yaml") {
				set[filepath.ToSlash(path)] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	for _, f := range []string{
		"tests/e2e-openshift/must-gather/install-instrumentation.yaml",
		"config/samples/instrumentation_v1alpha1_instrumentation.yaml",
		"docs/auto-instrumentation/README.md",
		"docs/auto-instrumentation/languages/apache-httpd.md",
		"docs/auto-instrumentation/languages/nginx.md",
	} {
		set[f] = struct{}{}
	}

	files := make([]string, 0, len(set))
	for f := range set {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}
