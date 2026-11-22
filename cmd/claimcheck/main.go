// Command claimcheck verifies the claims the video scripts make about this repo.
//
// Each docs/claims/*.json file lists the claims of one chapter and pins them to a git tag. A claim of kind "repo"
// carries checks that are run against the code at that tag; a claim of kind "reference" points at an outside source
// (a language spec) and is listed but not checked. See docs/claims/README.md for the file format.
//
// The report goes to stdout because it is the program's output, not a log.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// File is one docs/claims/*.json file.
type File struct {
	Chapter string  `json:"chapter"`
	Title   string  `json:"title"`
	Tag     string  `json:"tag"`
	Claims  []Claim `json:"claims"`
}

// Claim is one statement the script makes.
type Claim struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"` // "repo" or "reference"
	Claim   string   `json:"claim"`
	ClaimEN string   `json:"claim_en"`
	Sources []string `json:"sources,omitempty"`
	Checks  []Check  `json:"checks,omitempty"`
}

// Check is one way to verify a claim.
type Check struct {
	Type string `json:"type"` // "contains", "matches" or "run"

	// contains / matches: look at one file as it is at the tag.
	File    string `json:"file,omitempty"`
	Text    string `json:"text,omitempty"`
	Pattern string `json:"pattern,omitempty"`

	// run: execute a command in a throwaway checkout of the tag.
	Cmd            []string `json:"cmd,omitempty"`
	Exit           string   `json:"exit,omitempty"` // "zero" (default) or "nonzero"
	OutputContains string   `json:"output_contains,omitempty"`
	Edit           *Edit    `json:"edit,omitempty"`     // applied before the command, undone after
	Requires       []string `json:"requires,omitempty"` // "docker"
}

// Edit replaces one piece of text in one file, to show what breaks without it.
type Edit struct {
	File string `json:"file"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

const (
	kindRepo      = "repo"
	kindReference = "reference"
	runTimeout    = 10 * time.Minute
)

var idPattern = regexp.MustCompile(`^[A-Z]+[0-9]+-[0-9]+$`)

// validate reports what is wrong with the shape of a claims file, before anything is run.
func validate(f File) []string {
	var problems []string
	if f.Chapter == "" {
		problems = append(problems, "chapter is empty")
	}
	if f.Tag == "" {
		problems = append(problems, "tag is empty")
	}
	seen := map[string]bool{}
	for _, c := range f.Claims {
		switch {
		case !idPattern.MatchString(c.ID):
			problems = append(problems, fmt.Sprintf("claim id %q does not look like C5-1", c.ID))
		case seen[c.ID]:
			problems = append(problems, fmt.Sprintf("claim id %s is used twice", c.ID))
		}
		seen[c.ID] = true
		if c.Claim == "" {
			problems = append(problems, fmt.Sprintf("%s: claim text is empty", c.ID))
		}
		switch c.Kind {
		case kindRepo:
			if len(c.Checks) == 0 {
				problems = append(problems, fmt.Sprintf("%s: a repo claim needs at least one check", c.ID))
			}
		case kindReference:
			if len(c.Sources) == 0 {
				problems = append(problems, fmt.Sprintf("%s: a reference claim needs at least one source", c.ID))
			}
			if len(c.Checks) > 0 {
				problems = append(problems, fmt.Sprintf("%s: a reference claim cannot have checks", c.ID))
			}
		default:
			problems = append(problems, fmt.Sprintf("%s: kind %q is not repo or reference", c.ID, c.Kind))
		}
		for i, k := range c.Checks {
			if msg := validateCheck(k); msg != "" {
				problems = append(problems, fmt.Sprintf("%s check %d: %s", c.ID, i+1, msg))
			}
		}
	}
	return problems
}

func validateCheck(k Check) string {
	switch k.Type {
	case "contains":
		if k.File == "" || k.Text == "" {
			return "contains needs file and text"
		}
	case "matches":
		if k.File == "" || k.Pattern == "" {
			return "matches needs file and pattern"
		}
		if _, err := regexp.Compile(k.Pattern); err != nil {
			return "pattern does not compile: " + err.Error()
		}
	case "run":
		if len(k.Cmd) == 0 {
			return "run needs cmd"
		}
		if k.Exit != "" && k.Exit != "zero" && k.Exit != "nonzero" {
			return fmt.Sprintf("exit %q is not zero or nonzero", k.Exit)
		}
		if k.Edit != nil && (k.Edit.File == "" || k.Edit.Old == "") {
			return "edit needs file and old"
		}
		for _, r := range k.Requires {
			if r != "docker" {
				return fmt.Sprintf("requires %q is not known", r)
			}
		}
	default:
		return fmt.Sprintf("type %q is not contains, matches or run", k.Type)
	}
	return ""
}

// checkText runs a contains or matches check against file content. Line endings are normalised first, so a
// checkout with CRLF gives the same answer as one with LF.
func checkText(k Check, content string) error {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	switch k.Type {
	case "contains":
		if !strings.Contains(content, k.Text) {
			return fmt.Errorf("%s does not contain %q", k.File, k.Text)
		}
	case "matches":
		if !regexp.MustCompile(k.Pattern).MatchString(content) {
			return fmt.Errorf("%s does not match %q", k.File, k.Pattern)
		}
	}
	return nil
}

// checkRun compares a finished command with what the check expects.
func checkRun(k Check, exitCode int, output string) error {
	wantZero := k.Exit != "nonzero"
	if wantZero && exitCode != 0 {
		return fmt.Errorf("%s: exit %d, want 0\n%s", strings.Join(k.Cmd, " "), exitCode, tail(output, 15))
	}
	if !wantZero && exitCode == 0 {
		return fmt.Errorf("%s: exit 0, want a failure", strings.Join(k.Cmd, " "))
	}
	if k.OutputContains != "" && !strings.Contains(output, k.OutputContains) {
		return fmt.Errorf("%s: output does not contain %q\n%s", strings.Join(k.Cmd, " "), k.OutputContains, tail(output, 15))
	}
	return nil
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

var errSkipped = errors.New("skipped")

// checkout is a throwaway worktree of one tag. It is created on first use.
type checkout struct {
	repo, tag, dir string
}

func (c *checkout) path(ctx context.Context) (string, error) {
	if c.dir != "" {
		return c.dir, nil
	}
	dir, err := os.MkdirTemp("", "claimcheck-")
	if err != nil {
		return "", fmt.Errorf("make temp dir: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", c.repo, "worktree", "add", "--detach", dir, c.tag).CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("check out %s: %w\n%s", c.tag, err, out)
	}
	c.dir = dir
	return dir, nil
}

func (c *checkout) close() {
	if c.dir == "" {
		return
	}
	// Best effort: a leftover directory is only disk space, and "git worktree prune" forgets it.
	_ = exec.Command("git", "-C", c.repo, "worktree", "remove", "--force", c.dir).Run()
	_ = os.RemoveAll(c.dir)
}

type runner struct {
	repo      string
	hasDocker bool
}

func (r runner) check(ctx context.Context, co *checkout, k Check) error {
	if k.Type != "run" {
		out, err := exec.CommandContext(ctx, "git", "-C", r.repo, "show", co.tag+":"+k.File).Output()
		if err != nil {
			return fmt.Errorf("read %s at %s: %w", k.File, co.tag, err)
		}
		return checkText(k, string(out))
	}
	for _, need := range k.Requires {
		if need == "docker" && !r.hasDocker {
			return fmt.Errorf("%w: needs Docker, which is not running", errSkipped)
		}
	}
	dir, err := co.path(ctx)
	if err != nil {
		return err
	}
	if k.Edit != nil {
		undo, err := applyEdit(dir, *k.Edit)
		if err != nil {
			return err
		}
		defer undo()
	}
	if k.Cmd, err = r.expand(ctx, co.tag, k.Cmd); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, k.Cmd[0], k.Cmd[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	exitCode := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		exitCode = exitErr.ExitCode()
	case err != nil:
		return fmt.Errorf("start %s: %w", strings.Join(k.Cmd, " "), err)
	}
	return checkRun(k, exitCode, string(out))
}

// rootPlaceholder in a run command stands for the first commit in the history of the chapter's tag. A claim
// about the starting point uses it instead of a commit id, which would go stale if history were ever rewritten.
const rootPlaceholder = "{root}"

// expand replaces rootPlaceholder in the command. It asks git only when the placeholder is present.
func (r runner) expand(ctx context.Context, tag string, cmd []string) ([]string, error) {
	if !slices.ContainsFunc(cmd, func(a string) bool { return strings.Contains(a, rootPlaceholder) }) {
		return cmd, nil
	}
	out, err := exec.CommandContext(ctx, "git", "-C", r.repo, "rev-list", "--max-parents=0", tag).Output()
	if err != nil {
		return nil, fmt.Errorf("find the first commit of %s: %w", tag, err)
	}
	roots := strings.Fields(string(out))
	if len(roots) != 1 {
		return nil, fmt.Errorf("%s has %d root commits, want exactly 1", tag, len(roots))
	}
	return substitute(cmd, rootPlaceholder, roots[0]), nil
}

// substitute returns a copy of cmd with every occurrence of old replaced by new.
func substitute(cmd []string, old, new string) []string {
	out := make([]string, len(cmd))
	for i, a := range cmd {
		out[i] = strings.ReplaceAll(a, old, new)
	}
	return out
}

// applyEdit changes one file and returns the function that puts it back.
func applyEdit(dir string, e Edit) (func(), error) {
	path := filepath.Join(dir, filepath.FromSlash(e.File))
	before, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("edit %s: %w", e.File, err)
	}
	if !strings.Contains(string(before), e.Old) {
		return nil, fmt.Errorf("edit %s: %q is not in the file", e.File, e.Old)
	}
	after := strings.Replace(string(before), e.Old, e.New, 1)
	if err := os.WriteFile(path, []byte(after), 0o644); err != nil {
		return nil, fmt.Errorf("edit %s: %w", e.File, err)
	}
	return func() { _ = os.WriteFile(path, before, 0o644) }, nil
}

func load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read claims: %w", err)
	}
	var f File
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return f, nil
}

type tally struct{ pass, fail, skip, ref int }

func run(ctx context.Context, paths []string, r runner) (tally, error) {
	var t tally
	for _, path := range paths {
		f, err := load(path)
		if err != nil {
			return t, err
		}
		if problems := validate(f); len(problems) > 0 {
			return t, fmt.Errorf("%s is not valid:\n  %s", path, strings.Join(problems, "\n  "))
		}
		fmt.Printf("%s  %s  (tag %s)\n", f.Chapter, f.Title, f.Tag)
		co := &checkout{repo: r.repo, tag: f.Tag}
		for _, c := range f.Claims {
			if c.Kind == kindReference {
				t.ref++
				fmt.Printf("  REF   %s  %s  [not machine-checked: %s]\n", c.ID, c.Claim, strings.Join(c.Sources, ", "))
				continue
			}
			status, detail := "PASS", ""
			for _, k := range c.Checks {
				err := r.check(ctx, co, k)
				if errors.Is(err, errSkipped) {
					status, detail = "SKIP", err.Error()
					continue
				}
				if err != nil {
					status, detail = "FAIL", err.Error()
					break
				}
			}
			switch status {
			case "PASS":
				t.pass++
			case "SKIP":
				t.skip++
			default:
				t.fail++
			}
			fmt.Printf("  %s  %s  %s\n", status, c.ID, c.Claim)
			if detail != "" {
				fmt.Printf("        %s\n", strings.ReplaceAll(detail, "\n", "\n        "))
			}
		}
		co.close()
	}
	return t, nil
}

func main() {
	strict := flag.Bool("strict", false, "treat a skipped claim as a failure")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: claimcheck [-strict] [docs/claims/ch05.json ...]   (no file = every docs/claims/*.json)")
	}
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		found, err := filepath.Glob(filepath.Join("docs", "claims", "*.json"))
		if err != nil || len(found) == 0 {
			fmt.Fprintln(os.Stderr, "claimcheck: no docs/claims/*.json here; run it from the repo root")
			os.Exit(2)
		}
		sort.Strings(found)
		paths = found
	}
	repo, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "claimcheck:", err)
		os.Exit(2)
	}

	ctx := context.Background()
	r := runner{repo: repo, hasDocker: exec.CommandContext(ctx, "docker", "info").Run() == nil}
	t, err := run(ctx, paths, r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "claimcheck:", err)
		os.Exit(2)
	}
	fmt.Printf("\n%d passed, %d failed, %d skipped, %d reference (not machine-checked)\n", t.pass, t.fail, t.skip, t.ref)
	if t.fail > 0 || (*strict && t.skip > 0) {
		os.Exit(1)
	}
}
