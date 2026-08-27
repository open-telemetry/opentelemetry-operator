// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Command autoinstrumentation-revision manages the operator-owned revision
// suffix on autoinstrumentation image tags (<sdk-version>-<revision>). See
// autoinstrumentation/README.md for the tagging scheme.
//
// Usage:
//
//	go run ./hack/autoinstrumentation-revision languages
//	go run ./hack/autoinstrumentation-revision sdk-version <language>
//	go run ./hack/autoinstrumentation-revision revision <language>
//	go run ./hack/autoinstrumentation-revision check
//	go run ./hack/autoinstrumentation-revision apply
//
// check and apply diff against BASE_SHA, or the merge-base with the target
// branch (origin/main, then main, or TARGET_BRANCH) when BASE_SHA is unset.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-telemetry/opentelemetry-operator/hack/autoinstrumentation-revision/revision"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}
	repo := revision.New(root)

	switch args[0] {
	case "languages":
		langs, err := repo.Languages()
		if err != nil {
			return err
		}
		for _, lang := range langs {
			fmt.Println(lang)
		}
		return nil

	case "sdk-version":
		lang, err := requireLanguage(args, "sdk-version")
		if err != nil {
			return err
		}
		v, err := repo.SDKVersion(lang)
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil

	case "revision":
		lang, err := requireLanguage(args, "revision")
		if err != nil {
			return err
		}
		rev, err := repo.Revision(lang)
		if err != nil {
			return err
		}
		fmt.Println(rev)
		return nil

	case "check":
		baseSHA, err := revision.ResolveBaseSHA(repo.Git, os.Getenv("BASE_SHA"), os.Getenv("TARGET_BRANCH"))
		if err != nil {
			return err
		}
		problems, err := repo.Check(baseSHA)
		if err != nil {
			return err
		}
		// GitHub Actions renders "::error file=...::" lines as inline annotations.
		for _, p := range problems {
			fmt.Printf("::error file=%s::%s\n", p.File, p.Message)
			for _, f := range p.Changed {
				fmt.Printf("  changed: %s\n", f)
			}
		}
		if len(problems) > 0 {
			fmt.Printf("Found %d autoinstrumentation revision problem(s). See https://github.com/open-telemetry/opentelemetry-operator/blob/main/autoinstrumentation/README.md#image-tagging\n", len(problems))
			os.Exit(1)
		}
		fmt.Println("All autoinstrumentation revisions are valid.")
		return nil

	case "apply":
		baseSHA, err := revision.ResolveBaseSHA(repo.Git, os.Getenv("BASE_SHA"), os.Getenv("TARGET_BRANCH"))
		if err != nil {
			return err
		}
		changes, err := repo.Apply(baseSHA)
		if err != nil {
			return err
		}
		for _, c := range changes {
			fmt.Printf("bumped %s revision: %d -> %d\n", c.Language, c.From, c.To)
		}
		if len(changes) == 0 {
			fmt.Println("No revision changes needed.")
		}
		return nil

	default:
		return usageError()
	}
}

func requireLanguage(args []string, cmd string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: %s <language>", cmd)
	}
	return args[1], nil
}

func usageError() error {
	return errors.New("usage: autoinstrumentation-revision {languages|sdk-version <language>|revision <language>|check|apply}")
}

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
