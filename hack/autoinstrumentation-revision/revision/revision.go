// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package revision holds the canonical helpers for the operator-owned revision
// suffix on autoinstrumentation image tags (<sdk-version>-<revision>, e.g.
// 2.30.0-1). See autoinstrumentation/README.md for the tagging scheme.
package revision

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const autoinstrumentationDir = "autoinstrumentation"

// Repo resolves revision information for the autoinstrumentation image tags of a
// checkout rooted at Root, using Git to compare against a base commit.
type Repo struct {
	Root string
	Git  Git
}

// New returns a Repo rooted at root that shells out to git for history access.
func New(root string) Repo {
	return Repo{Root: root, Git: execGit{root: root}}
}

// Languages lists the languages that use the <sdk-version>-<revision> tag scheme,
// discovered from autoinstrumentation/<language>/revision.txt. Go (upstream
// image) and nginx (shares apache-httpd) have none.
func (r Repo) Languages() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(r.Root, autoinstrumentationDir, "*", "revision.txt"))
	if err != nil {
		return nil, err
	}
	langs := make([]string, 0, len(matches))
	for _, m := range matches {
		langs = append(langs, filepath.Base(filepath.Dir(m)))
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("no %s/*/revision.txt files found", autoinstrumentationDir)
	}
	slices.Sort(langs)
	return langs, nil
}

func sourceFile(lang string) string {
	switch lang {
	case "python":
		return filepath.Join(autoinstrumentationDir, "python", "requirements.txt")
	case "nodejs":
		return filepath.Join(autoinstrumentationDir, "nodejs", "package.json")
	default:
		return filepath.Join(autoinstrumentationDir, lang, "version.txt")
	}
}

func extractVersion(lang, source string) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}
	switch lang {
	case "python":
		line := source
		if i := strings.IndexByte(line, '\n'); i >= 0 {
			line = line[:i]
		}
		parts := strings.Split(line, "=")
		if len(parts) < 3 {
			return "", nil
		}
		return stripSpace(parts[2]), nil
	case "nodejs":
		var pkg struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.Unmarshal([]byte(source), &pkg); err != nil {
			return "", nil
		}
		return pkg.Dependencies["@opentelemetry/auto-instrumentations-node"], nil
	default:
		return stripSpace(source), nil
	}
}

// SDKVersion returns the upstream SDK version for a language from the working tree.
func (r Repo) SDKVersion(lang string) (string, error) {
	src := sourceFile(lang)
	content, err := os.ReadFile(filepath.Join(r.Root, src))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no SDK version source found at %q for %q; register it in sourceFile()/extractVersion() in hack/autoinstrumentation-revision", src, lang)
		}
		return "", err
	}
	return extractVersion(lang, string(content))
}

// Revision returns the current operator-owned revision for a language from the
// working tree.
func (r Repo) Revision(lang string) (int, error) {
	raw, err := r.readRevision(lang)
	if err != nil {
		return 0, err
	}
	rev, ok := parseRevision(raw)
	if !ok {
		return 0, fmt.Errorf("%s must contain a positive integer, got %q", revisionFile(lang), raw)
	}
	return rev, nil
}

func revisionFile(lang string) string {
	return filepath.Join(autoinstrumentationDir, lang, "revision.txt")
}

func (r Repo) readRevision(lang string) (string, error) {
	content, err := os.ReadFile(filepath.Join(r.Root, revisionFile(lang)))
	if err != nil {
		return "", err
	}
	return stripSpace(string(content)), nil
}

var positiveInteger = regexp.MustCompile(`^[1-9]\d*$`)

func parseRevision(raw string) (int, bool) {
	if !positiveInteger.MatchString(raw) {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

func stripSpace(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// ResolveBaseSHA returns baseSHA if set, otherwise the merge-base of HEAD with
// the first resolvable ref among targetBranch, origin/main, and main.
func ResolveBaseSHA(git Git, baseSHA, targetBranch string) (string, error) {
	if baseSHA != "" {
		return baseSHA, nil
	}
	for _, ref := range []string{targetBranch, "origin/main", "main"} {
		if ref == "" {
			continue
		}
		if sha, err := git.MergeBase(ref); err == nil && sha != "" {
			return sha, nil
		}
	}
	return "", errors.New("could not determine a base commit; set BASE_SHA (or TARGET_BRANCH) explicitly")
}

type bumpKind int

const (
	bumpNone bumpKind = iota
	bumpReset
	bumpIncrement
)

type languageState struct {
	lang           string
	revFile        string
	srcFile        string
	srcExists      bool
	headRevRaw     string
	headRev        int
	headRevValid   bool
	headSDK        string
	baseSDK        string
	baseRev        string
	contentChanged []string
}

func (r Repo) gather(baseSHA, lang string) (languageState, error) {
	ls := languageState{
		lang:    lang,
		revFile: revisionFile(lang),
		srcFile: sourceFile(lang),
	}

	if _, err := os.Stat(filepath.Join(r.Root, ls.srcFile)); err != nil {
		return ls, nil
	}
	ls.srcExists = true

	raw, err := r.readRevision(lang)
	if err != nil {
		return ls, err
	}
	ls.headRevRaw = raw
	ls.headRev, ls.headRevValid = parseRevision(raw)

	src, err := os.ReadFile(filepath.Join(r.Root, ls.srcFile))
	if err != nil {
		return ls, err
	}
	if ls.headSDK, err = extractVersion(lang, string(src)); err != nil {
		return ls, err
	}
	if ls.baseSDK, err = extractVersion(lang, r.Git.Show(baseSHA, ls.srcFile)); err != nil {
		return ls, err
	}
	ls.baseRev = stripSpace(r.Git.Show(baseSHA, ls.revFile))
	if ls.contentChanged, err = r.Git.DiffNames(baseSHA, filepath.Join(autoinstrumentationDir, lang), ls.revFile); err != nil {
		return ls, err
	}
	return ls, nil
}

func (s languageState) desiredRevision() (int, bumpKind) {
	switch {
	case s.baseSDK != "" && s.headSDK != s.baseSDK:
		return 1, bumpReset
	case len(s.contentChanged) > 0 && s.baseRev != "":
		if base, ok := parseRevision(s.baseRev); ok && s.headRev <= base {
			return base + 1, bumpIncrement
		}
	}
	return s.headRev, bumpNone
}

// Problem describes a single revision rule violation found by Check.
type Problem struct {
	// Language is the autoinstrumentation language the problem concerns.
	Language string
	// File is the repo-relative revision.txt the problem concerns.
	File string
	// Message explains what is wrong and how to fix it.
	Message string
	// Changed lists the image content files that require the revision to be
	// incremented, when that is the violation; it is nil otherwise.
	Changed []string
}

// Check validates the revision bumps for a pull request against baseSHA and
// returns a Problem for each rule violation. The error is non-nil only on an
// unexpected IO or git failure, not for validation problems.
func (r Repo) Check(baseSHA string) ([]Problem, error) {
	langs, err := r.Languages()
	if err != nil {
		return nil, err
	}

	var problems []Problem
	for _, lang := range langs {
		ls, err := r.gather(baseSHA, lang)
		if err != nil {
			return nil, err
		}

		switch {
		case !ls.srcExists:
			problems = append(problems, Problem{
				Language: lang,
				File:     ls.revFile,
				Message:  fmt.Sprintf("no SDK version source found at %q for %q; register it in sourceFile()/extractVersion() in hack/autoinstrumentation-revision", ls.srcFile, lang),
			})
			continue
		case !ls.headRevValid:
			problems = append(problems, Problem{
				Language: lang,
				File:     ls.revFile,
				Message:  fmt.Sprintf("%s must contain a positive integer, got %q", ls.revFile, ls.headRevRaw),
			})
			continue
		}

		desired, kind := ls.desiredRevision()
		if ls.headRev == desired {
			continue
		}
		switch kind {
		case bumpReset:
			problems = append(problems, Problem{
				Language: lang,
				File:     ls.revFile,
				Message:  fmt.Sprintf("%s SDK version changed (%s -> %s); reset %s to 1", lang, ls.baseSDK, ls.headSDK, ls.revFile),
			})
		case bumpIncrement:
			problems = append(problems, Problem{
				Language: lang,
				File:     ls.revFile,
				Message:  fmt.Sprintf("%s image content changed with no SDK version bump; increment %s (%s -> %d)", lang, ls.revFile, ls.baseRev, desired),
				Changed:  ls.contentChanged,
			})
		case bumpNone:
		}
	}

	return problems, nil
}

// Change records a revision file rewritten by Apply.
type Change struct {
	Language string
	From     int
	To       int
}

// Apply writes the correct revisions for a pull request against baseSHA (reset on
// SDK bump, increment on image content change) and returns the changes it made.
func (r Repo) Apply(baseSHA string) ([]Change, error) {
	langs, err := r.Languages()
	if err != nil {
		return nil, err
	}

	var changes []Change
	for _, lang := range langs {
		ls, err := r.gather(baseSHA, lang)
		if err != nil {
			return nil, err
		}
		if !ls.srcExists || !ls.headRevValid {
			continue
		}

		desired, _ := ls.desiredRevision()
		if desired == ls.headRev {
			continue
		}
		if err := os.WriteFile(filepath.Join(r.Root, ls.revFile), []byte(strconv.Itoa(desired)+"\n"), 0o600); err != nil {
			return nil, err
		}
		changes = append(changes, Change{Language: lang, From: ls.headRev, To: desired})
	}

	return changes, nil
}
