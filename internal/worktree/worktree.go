package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Entry represents a git worktree as reported by porcelain output.
type Entry struct {
	Path       string
	Head       string
	Branch     string
	Bare       bool
	Detached   bool
	Locked     bool
	LockReason string
}

// HeadShort returns the short SHA representation (7 characters) if available.
func (e Entry) HeadShort() string {
	if len(e.Head) >= 7 {
		return e.Head[:7]
	}
	return e.Head
}

// Basename returns the last element of the worktree path.
func (e Entry) Basename() string {
	return filepath.Base(e.Path)
}

var blankLineRegex = regexp.MustCompile(`\n{2,}`)

// ParsePorcelainList parses `git worktree list --porcelain` output.
func ParsePorcelainList(out string) ([]Entry, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	blocks := blankLineRegex.Split(out, -1)
	entries := make([]Entry, 0, len(blocks))

	for _, block := range blocks {
		var entry Entry
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, " ", 2)
			key := parts[0]
			var value string
			if len(parts) == 2 {
				value = strings.TrimSpace(parts[1])
			}

			switch key {
			case "worktree":
				entry.Path = value
			case "HEAD":
				entry.Head = value
			case "branch":
				entry.Branch = value
			case "bare":
				entry.Bare = true
			case "detached":
				entry.Detached = true
			case "locked":
				entry.Locked = true
				entry.LockReason = strings.TrimPrefix(value, "reason: ")
			default:
				// ignore unknown keys per spec
			}
		}

		if entry.Path == "" {
			return nil, errors.New("porcelain worktree entry missing path")
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

var sanitizeHunk = regexp.MustCompile(`[\\/\s]+`)

// SanitizeBranch converts a branch ref into a filesystem-friendly name.
func SanitizeBranch(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "refs/heads/")
	if name == "" {
		return "wt"
	}

	name = sanitizeHunk.ReplaceAllString(name, "-")

	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		repl := r
		switch {
		case unicode.IsSpace(r):
			repl = '-'
		case r == '/' || r == '\\':
			repl = '-'
		case r == '-':
			repl = '-'
		case unicode.IsControl(r):
			repl = '-'
		}

		if repl == '-' {
			if lastHyphen {
				continue
			}
			lastHyphen = true
			b.WriteRune('-')
			continue
		}

		lastHyphen = false
		b.WriteRune(repl)
	}

	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		sanitized = "wt"
	}

	if len(sanitized) > 64 {
		sanitized = sanitized[:64]
		// Trim trailing hyphen again if slicing exposed it.
		sanitized = strings.TrimRight(sanitized, "-")
		if sanitized == "" {
			sanitized = "wt"
		}
	}

	return sanitized
}

// DefaultWorktreePath returns the desired path for a new worktree branch.
func DefaultWorktreePath(repoRoot, branch string, exists func(string) bool) (string, error) {
	if repoRoot == "" {
		return "", errors.New("repo root required")
	}

	sanitized := SanitizeBranch(branch)

	parent := filepath.Dir(repoRoot)
	base := fmt.Sprintf(".%s-koz", filepath.Base(repoRoot))
	root := filepath.Join(parent, base)

	candidate := filepath.Join(root, sanitized)

	if exists == nil {
		exists = func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		}
	}

	if !exists(candidate) {
		return candidate, nil
	}

	for i := 2; i < 1000; i++ {
		alt := filepath.Join(root, fmt.Sprintf("%s-%d", sanitized, i))
		if !exists(alt) {
			return alt, nil
		}
	}

	return "", fmt.Errorf("unable to compute unique worktree path for %s", branch)
}
