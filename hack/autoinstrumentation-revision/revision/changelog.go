// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package revision

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func languageName(lang string) string {
	switch lang {
	case "dotnet":
		return ".NET"
	case "nodejs":
		return "Node.js"
	case "apache-httpd":
		return "Apache HTTPD / Nginx"
	case "php":
		return "PHP"
	default:
		return strings.ToUpper(lang[:1]) + lang[1:]
	}
}

// upstreamReleaseURL links to the upstream SDK release for languages whose
// upstream uses predictable GitHub release tags, or "" when none is known.
func upstreamReleaseURL(lang, sdkVersion string) string {
	if sdkVersion == "" {
		return ""
	}
	var repo string
	switch lang {
	case "java":
		repo = "open-telemetry/opentelemetry-java-instrumentation"
	case "dotnet":
		repo = "open-telemetry/opentelemetry-dotnet-instrumentation"
	case "python":
		repo = "open-telemetry/opentelemetry-python-contrib"
	default:
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/releases/tag/v%s", repo, sdkVersion)
}

// changelogChange returns the published tag, changelog note, and upstream release
// URL for a language's change against the base commit, or ok=false when there is
// nothing to log. The release URL is set only for SDK bumps. The tag uses the
// desired revision, so it is correct whether or not Apply has run.
func (s languageState) changelogChange() (tag, note, releaseURL string, ok bool) {
	if s.headSDK == "" {
		return "", "", "", false
	}
	desired, _ := s.desiredRevision()
	tag = s.headSDK + "-" + strconv.Itoa(desired)
	name := languageName(s.lang)
	switch {
	case s.baseSDK == "":
		note = fmt.Sprintf("Add %s auto-instrumentation %s.", name, s.headSDK)
		releaseURL = upstreamReleaseURL(s.lang, s.headSDK)
	case s.headSDK != s.baseSDK:
		note = fmt.Sprintf("Update %s auto-instrumentation from %s to %s.", name, s.baseSDK, s.headSDK)
		releaseURL = upstreamReleaseURL(s.lang, s.headSDK)
	case len(s.contentChanged) > 0:
		note = fmt.Sprintf("Rebuild %s auto-instrumentation image (base image or dependency update).", name)
	default:
		return "", "", "", false
	}
	return tag, note, releaseURL, true
}

func renderChangelogEntry(tag, note, prRef, releaseURL string) string {
	line := "- " + note
	if releaseURL != "" {
		line += " See [release notes](" + releaseURL + ")."
	}
	if prRef != "" {
		line += " (#" + prRef + ")"
	}
	return fmt.Sprintf("## %s\n\n%s\n", tag, line)
}

func hasChangelogTag(content, tag string) bool {
	return regexp.MustCompile("(?m)^## " + regexp.QuoteMeta(tag) + "$").MatchString(content)
}

// prependChangelogEntry inserts entry above the first existing "## " heading,
// keeping newest entries on top.
func prependChangelogEntry(content, entry string) string {
	entry = strings.TrimRight(entry, "\n") + "\n\n"
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "## ") {
			head := strings.TrimRight(strings.Join(lines[:i], "\n"), "\n")
			return head + "\n\n" + entry + strings.Join(lines[i:], "\n")
		}
	}
	return strings.TrimRight(content, "\n") + "\n\n" + entry
}

// ChangelogEntry records a per-language CHANGELOG.md entry written by ApplyChangelog.
type ChangelogEntry struct {
	Language string
	Tag      string
}

// ApplyChangelog appends a CHANGELOG.md entry for every language whose published
// tag changed against baseSHA and does not already have one. prRef, when set, is
// appended as a "(#<prRef>)" reference. It is idempotent.
func (r Repo) ApplyChangelog(baseSHA, prRef string) ([]ChangelogEntry, error) {
	langs, err := r.Languages()
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var entries []ChangelogEntry
	for _, lang := range langs {
		ls, err := r.gather(baseSHA, lang)
		if err != nil {
			return nil, err
		}
		if !ls.srcExists || !ls.headRevValid {
			continue
		}
		tag, note, releaseURL, ok := ls.changelogChange()
		if !ok {
			continue
		}
		rel := changelogFile(lang)
		content, err := root.ReadFile(rel)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		if hasChangelogTag(string(content), tag) {
			continue
		}
		entry := renderChangelogEntry(tag, note, prRef, releaseURL)
		if err := root.WriteFile(rel, []byte(prependChangelogEntry(string(content), entry)), 0o600); err != nil {
			return nil, err
		}
		entries = append(entries, ChangelogEntry{Language: lang, Tag: tag})
	}
	return entries, nil
}

// CheckChangelog returns a Problem for every language whose published tag changed
// against baseSHA but has no matching CHANGELOG.md entry.
func (r Repo) CheckChangelog(baseSHA string) ([]Problem, error) {
	langs, err := r.Languages()
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var problems []Problem
	for _, lang := range langs {
		ls, err := r.gather(baseSHA, lang)
		if err != nil {
			return nil, err
		}
		if !ls.srcExists || !ls.headRevValid {
			continue
		}
		tag, _, _, ok := ls.changelogChange()
		if !ok {
			continue
		}
		rel := changelogFile(lang)
		content, err := root.ReadFile(rel)
		if err != nil {
			if os.IsNotExist(err) {
				problems = append(problems, Problem{Language: lang, File: rel, Message: fmt.Sprintf("%s image tag changed to %s but %s is missing; run `make bump-autoinstrumentation-revision`", lang, tag, rel)})
				continue
			}
			return nil, err
		}
		if !hasChangelogTag(string(content), tag) {
			problems = append(problems, Problem{Language: lang, File: rel, Message: fmt.Sprintf("%s image tag changed to %s but %s has no matching entry; run `make bump-autoinstrumentation-revision`", lang, tag, rel)})
		}
	}
	return problems, nil
}
