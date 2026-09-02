// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package revision

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name   string
		lang   string
		source string
		want   string
	}{
		{name: "java version.txt", lang: "java", source: "2.30.0\n", want: "2.30.0"},
		{name: "default trims whitespace", lang: "dotnet", source: "  1.16.0  \n", want: "1.16.0"},
		{name: "python distro pin", lang: "python", source: "opentelemetry-distro==0.65b0\nopentelemetry-exporter-otlp==1.2.3\n", want: "0.65b0"},
		{name: "nodejs package.json", lang: "nodejs", source: `{"dependencies":{"@opentelemetry/auto-instrumentations-node":"0.78.0"}}`, want: "0.78.0"},
		{name: "empty source", lang: "java", source: "  \n", want: ""},
		{name: "nodejs invalid json", lang: "nodejs", source: "not json", want: ""},
		{name: "nodejs missing dep", lang: "nodejs", source: `{"dependencies":{}}`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractVersion(tt.lang, tt.source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("extractVersion(%q, %q) = %q, want %q", tt.lang, tt.source, got, tt.want)
			}
		})
	}
}

func TestParseRevision(t *testing.T) {
	tests := []struct {
		raw  string
		want int
		ok   bool
	}{
		{"1", 1, true},
		{"42", 42, true},
		{"0", 0, false},
		{"-1", 0, false},
		{"01", 0, false},
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseRevision(tt.raw)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseRevision(%q) = (%d, %v), want (%d, %v)", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

// fakeGit serves file contents at a single base commit and a fixed set of files
// changed between that base and the working tree.
type fakeGit struct {
	base    map[string]string
	changed []string
}

func (fakeGit) MergeBase(string) (string, error) { return "BASE", nil }

func (f fakeGit) Show(_, path string) string { return f.base[path] }

func (f fakeGit) DiffNames(_, dir, exclude string) ([]string, error) {
	var out []string
	for _, c := range f.changed {
		if strings.HasPrefix(c, dir+"/") && c != exclude {
			out = append(out, c)
		}
	}
	return out, nil
}

// writeLang lays out working-tree files for a language under root.
func writeLang(t *testing.T, root, lang string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, autoinstrumentationDir, lang)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name         string
		lang         string // defaults to "java"
		work         map[string]string
		base         map[string]string
		changed      []string
		wantProblems int
	}{
		{
			name:         "unchanged is valid",
			work:         map[string]string{"version.txt": "2.30.0", "revision.txt": "1"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			wantProblems: 0,
		},
		{
			name:         "sdk bump without reset",
			work:         map[string]string{"version.txt": "2.31.0", "revision.txt": "5"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "5"},
			wantProblems: 1,
		},
		{
			name:         "sdk bump with reset to 1",
			work:         map[string]string{"version.txt": "2.31.0", "revision.txt": "1"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "5"},
			wantProblems: 0,
		},
		{
			name:         "content change without increment",
			work:         map[string]string{"version.txt": "2.30.0", "revision.txt": "1"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			changed:      []string{"autoinstrumentation/java/Dockerfile"},
			wantProblems: 1,
		},
		{
			name:         "content change with increment",
			work:         map[string]string{"version.txt": "2.30.0", "revision.txt": "2"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			changed:      []string{"autoinstrumentation/java/Dockerfile"},
			wantProblems: 0,
		},
		{
			name:         "revision file change alone is not content",
			work:         map[string]string{"version.txt": "2.30.0", "revision.txt": "1"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			changed:      []string{"autoinstrumentation/java/revision.txt"},
			wantProblems: 0,
		},
		{
			name:         "invalid revision",
			work:         map[string]string{"version.txt": "2.30.0", "revision.txt": "0"},
			base:         map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			wantProblems: 1,
		},
		{
			name:         "new language has no base to compare",
			work:         map[string]string{"version.txt": "1.0.0", "revision.txt": "1"},
			base:         map[string]string{},
			changed:      []string{"autoinstrumentation/java/Dockerfile"},
			wantProblems: 0,
		},
		{
			name:         "python sdk bump without reset",
			lang:         "python",
			work:         map[string]string{"requirements.txt": "opentelemetry-distro==0.66b0\n", "revision.txt": "3"},
			base:         map[string]string{"autoinstrumentation/python/requirements.txt": "opentelemetry-distro==0.65b0\n", "autoinstrumentation/python/revision.txt": "3"},
			wantProblems: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := tt.lang
			if lang == "" {
				lang = "java"
			}
			root := t.TempDir()
			writeLang(t, root, lang, tt.work)
			repo := Repo{Root: root, Git: fakeGit{base: tt.base, changed: tt.changed}}

			problems, err := repo.Check("BASE")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(problems) != tt.wantProblems {
				t.Errorf("Check problems = %d, want %d\nproblems: %+v", len(problems), tt.wantProblems, problems)
			}
		})
	}
}

func TestCheckMissingSource(t *testing.T) {
	root := t.TempDir()
	writeLang(t, root, "java", map[string]string{"revision.txt": "1"}) // no version.txt
	repo := Repo{Root: root, Git: fakeGit{base: map[string]string{}}}

	problems, err := repo.Check("BASE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Errorf("Check problems = %d, want 1\nproblems: %+v", len(problems), problems)
	}
}

func TestCheckProblemDetail(t *testing.T) {
	root := t.TempDir()
	writeLang(t, root, "java", map[string]string{"version.txt": "2.30.0", "revision.txt": "1"})
	git := fakeGit{
		base:    map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
		changed: []string{"autoinstrumentation/java/Dockerfile"},
	}
	repo := Repo{Root: root, Git: git}

	problems, err := repo.Check("BASE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("Check problems = %d, want 1", len(problems))
	}
	p := problems[0]
	if p.Language != "java" {
		t.Errorf("Language = %q, want java", p.Language)
	}
	if p.File != filepath.Join(autoinstrumentationDir, "java", "revision.txt") {
		t.Errorf("File = %q", p.File)
	}
	if !slices.Equal(p.Changed, []string{"autoinstrumentation/java/Dockerfile"}) {
		t.Errorf("Changed = %v", p.Changed)
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		work    map[string]string
		base    map[string]string
		changed []string
		want    string
	}{
		{
			name:    "increment on content change",
			work:    map[string]string{"version.txt": "2.30.0", "revision.txt": "1"},
			base:    map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "1"},
			changed: []string{"autoinstrumentation/java/Dockerfile"},
			want:    "2\n",
		},
		{
			name: "reset on sdk bump",
			work: map[string]string{"version.txt": "2.31.0", "revision.txt": "5"},
			base: map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "5"},
			want: "1\n",
		},
		{
			name: "no change leaves file untouched",
			work: map[string]string{"version.txt": "2.30.0", "revision.txt": "3"},
			base: map[string]string{"autoinstrumentation/java/version.txt": "2.30.0", "autoinstrumentation/java/revision.txt": "3"},
			want: "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeLang(t, root, "java", tt.work)
			repo := Repo{Root: root, Git: fakeGit{base: tt.base, changed: tt.changed}}

			if _, err := repo.Apply("BASE"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(root, autoinstrumentationDir, "java", "revision.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("revision.txt = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveBaseSHA(t *testing.T) {
	if got, _ := ResolveBaseSHA(fakeGit{}, "deadbeef", ""); got != "deadbeef" {
		t.Errorf("explicit BASE_SHA = %q, want deadbeef", got)
	}
	got, err := ResolveBaseSHA(fakeGit{}, "", "some-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "BASE" {
		t.Errorf("merge-base = %q, want BASE", got)
	}
}
