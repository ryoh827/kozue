package cli

import (
	"testing"

	"github.com/ryoh827/kozue/internal/worktree"
)

func TestHazardsFor(t *testing.T) {
	entry := worktree.Entry{Locked: true, LockReason: "freeze"}
	dirty := 3
	aheadVal := 2

	hazards := hazardsFor(entry, dirty, &aheadVal)
	if len(hazards) != 3 {
		t.Fatalf("expected 3 hazards, got %d", len(hazards))
	}
}
