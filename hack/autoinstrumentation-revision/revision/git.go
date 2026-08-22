// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package revision

import (
	"context"
	"os/exec"
	"strings"
)

// Git provides the history access needed to compare the working tree against a
// base commit.
type Git interface {
	// MergeBase returns the merge-base of HEAD and ref.
	MergeBase(ref string) (string, error)
	// Show returns the contents of path at sha, or an empty string if the path
	// does not exist there.
	Show(sha, path string) string
	// DiffNames lists the files changed under dir between sha and the working
	// tree, excluding excludePath.
	DiffNames(sha, dir, excludePath string) ([]string, error)
}

// execGit is the default Git implementation, shelling out to the git binary
// rooted at root.
type execGit struct {
	root string
}

func (g execGit) run(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = g.root
	out, err := cmd.Output()
	return string(out), err
}

// MergeBase returns the merge-base of HEAD and ref via `git merge-base`.
func (g execGit) MergeBase(ref string) (string, error) {
	out, err := g.run("merge-base", "HEAD", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Show returns the contents of path at sha via `git show`, or an empty string if
// the path does not exist there.
func (g execGit) Show(sha, path string) string {
	out, err := g.run("show", sha+":"+path)
	if err != nil {
		return ""
	}
	return out
}

// DiffNames lists the files changed under dir between sha and the working tree
// via `git diff`, excluding excludePath.
func (g execGit) DiffNames(sha, dir, excludePath string) ([]string, error) {
	out, err := g.run("diff", "--name-only", sha, "--", dir, ":(exclude)"+excludePath)
	if err != nil {
		return nil, err
	}
	var names []string
	for line := range strings.SplitSeq(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}
