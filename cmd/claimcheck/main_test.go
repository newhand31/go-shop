package main

import (
	"strings"
	"testing"
)

func validFile() File {
	return File{
		Chapter: "ch05",
		Tag:     "ep05",
		Claims: []Claim{
			{ID: "C5-1", Kind: kindRepo, Claim: "x", Checks: []Check{{Type: "contains", File: "a.go", Text: "x"}}},
			{ID: "X5-1", Kind: kindReference, Claim: "y", Sources: []string{"https://go.dev/ref/spec"}},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*File)
		want   string // substring of the one expected problem; "" = no problems
	}{
		{"valid", func(*File) {}, ""},
		{"no tag", func(f *File) { f.Tag = "" }, "tag is empty"},
		{"bad id", func(f *File) { f.Claims[0].ID = "five" }, "does not look like"},
		{"duplicate id", func(f *File) { f.Claims[1].ID = "C5-1" }, "used twice"},
		{"repo claim without checks", func(f *File) { f.Claims[0].Checks = nil }, "needs at least one check"},
		{"reference claim without source", func(f *File) { f.Claims[1].Sources = nil }, "needs at least one source"},
		{"reference claim with checks", func(f *File) { f.Claims[1].Checks = f.Claims[0].Checks }, "cannot have checks"},
		{"unknown kind", func(f *File) { f.Claims[0].Kind = "opinion" }, "is not repo or reference"},
		{"unknown check type", func(f *File) { f.Claims[0].Checks[0].Type = "vibes" }, "is not contains, matches or run"},
		{"bad pattern", func(f *File) { f.Claims[0].Checks[0] = Check{Type: "matches", File: "a.go", Pattern: "("} }, "does not compile"},
		{"run without cmd", func(f *File) { f.Claims[0].Checks[0] = Check{Type: "run"} }, "run needs cmd"},
		{"bad exit", func(f *File) { f.Claims[0].Checks[0] = Check{Type: "run", Cmd: []string{"go"}, Exit: "maybe"} }, "is not zero or nonzero"},
		{"unknown requirement", func(f *File) {
			f.Claims[0].Checks[0] = Check{Type: "run", Cmd: []string{"go"}, Requires: []string{"moon"}}
		}, "is not known"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := validFile()
			tt.mutate(&f)
			got := validate(f)
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no problems, got %v", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("want one problem containing %q, got %v", tt.want, got)
			}
		})
	}
}

func TestCheckText(t *testing.T) {
	const content = "type CartStore interface {\r\n\tItems()\r\n}\r\n"
	tests := []struct {
		name    string
		check   Check
		wantErr bool
	}{
		{"contains hit", Check{Type: "contains", File: "a.go", Text: "type CartStore interface {"}, false},
		{"contains spans a CRLF line break", Check{Type: "contains", File: "a.go", Text: "interface {\n\tItems()"}, false},
		{"contains miss", Check{Type: "contains", File: "a.go", Text: "implements"}, true},
		{"matches hit", Check{Type: "matches", File: "a.go", Pattern: `(?m)^type \w+ interface`}, false},
		{"matches miss", Check{Type: "matches", File: "a.go", Pattern: `(?m)^func `}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkText(tt.check, content)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckRun(t *testing.T) {
	cmd := []string{"go", "build"}
	tests := []struct {
		name    string
		check   Check
		exit    int
		output  string
		wantErr bool
	}{
		{"zero wanted, zero got", Check{Cmd: cmd}, 0, "", false},
		{"zero wanted, failure got", Check{Cmd: cmd}, 1, "boom", true},
		{"failure wanted, failure got", Check{Cmd: cmd, Exit: "nonzero"}, 1, "boom", false},
		{"failure wanted, zero got", Check{Cmd: cmd, Exit: "nonzero"}, 0, "", true},
		{"output present", Check{Cmd: cmd, Exit: "nonzero", OutputContains: "missing method"}, 1, "x: missing method Y", false},
		{"output absent", Check{Cmd: cmd, Exit: "nonzero", OutputContains: "missing method"}, 1, "syntax error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRun(tt.check, tt.exit, tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubstitute(t *testing.T) {
	tests := []struct {
		name string
		cmd  []string
		want []string
	}{
		{"whole argument", []string{"git", "grep", "-q", "x", "{root}"}, []string{"git", "grep", "-q", "x", "abc123"}},
		{"inside an argument", []string{"git", "cat-file", "-e", "{root}:a.go"}, []string{"git", "cat-file", "-e", "abc123:a.go"}},
		{"absent", []string{"go", "build", "./..."}, []string{"go", "build", "./..."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]string(nil), tt.cmd...)
			got := substitute(tt.cmd, rootPlaceholder, "abc123")
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if strings.Join(tt.cmd, " ") != strings.Join(in, " ") {
				t.Fatalf("input was changed to %v", tt.cmd)
			}
		})
	}
}
