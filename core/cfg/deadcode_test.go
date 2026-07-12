package cfg

import (
	"sort"
	"testing"
)

func TestDeadCodeNone(t *testing.T) {
	c := NewCFG("no_dead")
	b1 := c.CreateBlock("b1")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(result.Findings))
	}
	if result.DeadBlocks != 0 {
		t.Fatalf("expected 0 dead blocks, got %d", result.DeadBlocks)
	}
}

func TestDeadCodeUnreachable(t *testing.T) {
	c := NewCFG("unreachable")
	b1 := c.CreateBlock("b1")
	orphan := c.CreateBlock("orphan")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	if result.DeadBlocks != 1 {
		t.Fatalf("expected 1 dead block, got %d", result.DeadBlocks)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	f := result.Findings[0]
	if f.BlockID != orphan.ID {
		t.Fatalf("expected finding for block %s, got %s", orphan.ID, f.BlockID)
	}
	if f.Severity != SeverityCritical {
		t.Fatalf("expected SeverityCritical, got %d", f.Severity)
	}
	if f.Reason != "unreachable" {
		t.Fatalf("expected reason 'unreachable', got %s", f.Reason)
	}
}

func TestDeadCodeAfterReturn(t *testing.T) {
	// Block has code after a return statement (intra-block dead code).
	// The block itself is reachable; the dead code is within it.
	c := NewCFG("after_return")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("x = 1")
	b1.AddStatement("return")
	b1.AddStatement("y = 2") // dead code after return
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	// No successor — return terminates flow.

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	found := false
	for _, f := range result.Findings {
		if f.BlockID == b1.ID && f.Reason == "after_return" {
			found = true
			if f.Severity != SeverityWarning {
				t.Fatalf("expected SeverityWarning for after_return, got %d", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected after_return finding")
	}
}

func TestDeadCodeAfterBreak(t *testing.T) {
	c := NewCFG("after_break")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("break")
	b1.AddStatement("x = 1") // dead code after break
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	found := false
	for _, f := range result.Findings {
		if f.BlockID == b1.ID && f.Reason == "after_break" {
			found = true
			if f.Severity != SeverityInfo {
				t.Fatalf("expected SeverityInfo for after_break, got %d", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected after_break finding")
	}
}

func TestDeadCodeAfterThrow(t *testing.T) {
	c := NewCFG("after_throw")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("throw")
	b1.AddStatement("cleanup") // dead code after throw
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	found := false
	for _, f := range result.Findings {
		if f.BlockID == b1.ID && f.Reason == "after_throw" {
			found = true
			if f.Severity != SeverityWarning {
				t.Fatalf("expected SeverityWarning for after_throw, got %d", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected after_throw finding")
	}
}

func TestDeadCodeAfterContinue(t *testing.T) {
	c := NewCFG("after_continue")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("continue")
	b1.AddStatement("x = 1") // dead code after continue
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	found := false
	for _, f := range result.Findings {
		if f.BlockID == b1.ID && f.Reason == "after_continue" {
			found = true
			if f.Severity != SeverityInfo {
				t.Fatalf("expected SeverityInfo for after_continue, got %d", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected after_continue finding")
	}
}

func TestDeadCodeTerminatorAtEnd(t *testing.T) {
	// Return is the last statement — no intra-block dead code.
	c := NewCFG("terminator_last")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("x = 1")
	b1.AddStatement("return")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	for _, f := range result.Findings {
		if f.BlockID == b1.ID {
			t.Fatalf("unexpected finding for block with return at end: %s", f.Reason)
		}
	}
}

func TestDeadCodeNilClassifier(t *testing.T) {
	c := NewCFG("nil_classifier")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("return")
	b1.AddStatement("dead code")
	orphan := c.CreateBlock("orphan")
	_ = orphan
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	// Without classifier, only structurally disconnected blocks are detected.
	// b1's successor (exit) is reachable because nil classifier follows all edges.
	result := DetectDeadCode(c, DeadCodeConfig{})

	if result.DeadBlocks != 1 {
		t.Fatalf("expected 1 dead block (orphan only), got %d", result.DeadBlocks)
	}
	if result.Findings[0].Reason != "unreachable" {
		t.Fatalf("expected 'unreachable' reason, got %s", result.Findings[0].Reason)
	}
}

func TestDeadCodeNilCFG(t *testing.T) {
	result := DetectDeadCode(nil, DeadCodeConfig{})
	if result.TotalBlocks != 0 {
		t.Fatalf("expected 0 total blocks, got %d", result.TotalBlocks)
	}
}

func TestDeadCodeTotalBlocks(t *testing.T) {
	c := NewCFG("total")
	b1 := c.CreateBlock("b1")
	b2 := c.CreateBlock("b2")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, b2, EdgeNormal)
	c.ConnectBlocks(b2, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	// entry + exit + b1 + b2 = 4
	if result.TotalBlocks != 4 {
		t.Fatalf("expected 4 total blocks, got %d", result.TotalBlocks)
	}
}

func TestDeadCodeMixed(t *testing.T) {
	// b1 has return with dead code after it (intra-block), orphan is disconnected.
	c := NewCFG("mixed")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("return")
	b1.AddStatement("dead")
	orphan := c.CreateBlock("orphan")
	_ = orphan
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	// Sort findings by reason for deterministic checking.
	sort.Slice(result.Findings, func(i, j int) bool {
		return result.Findings[i].Reason < result.Findings[j].Reason
	})

	// Expect: after_return (b1 intra-block) + unreachable (orphan) + unreachable (exit)
	if result.DeadBlocks != 3 {
		t.Fatalf("expected 3 dead blocks, got %d", result.DeadBlocks)
	}

	reasons := map[string]int{}
	for _, f := range result.Findings {
		reasons[f.Reason]++
	}
	if reasons["after_return"] != 1 {
		t.Fatalf("expected 1 after_return finding, got %d", reasons["after_return"])
	}
	if reasons["unreachable"] != 2 {
		t.Fatalf("expected 2 unreachable findings (orphan + exit), got %d", reasons["unreachable"])
	}
}

// TestDeadCodeSuccessorOfReturnBlock is the regression test for P1:
// entry -> block(return) -> next should mark "next" as unreachable.
func TestDeadCodeSuccessorOfReturnBlock(t *testing.T) {
	c := NewCFG("p1_regression")
	retBlock := c.CreateBlock("return_block")
	retBlock.AddStatement("return")
	nextBlock := c.CreateBlock("next_block")
	nextBlock.AddStatement("should_be_dead")

	c.ConnectBlocks(c.Entry, retBlock, EdgeNormal)
	c.ConnectBlocks(retBlock, nextBlock, EdgeNormal)
	c.ConnectBlocks(nextBlock, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	// nextBlock should be unreachable (only path is through return_block which terminates).
	foundNext := false
	for _, f := range result.Findings {
		if f.BlockID == nextBlock.ID {
			foundNext = true
			if f.Reason != "unreachable" {
				t.Fatalf("expected 'unreachable' for next_block, got %s", f.Reason)
			}
			if f.Severity != SeverityCritical {
				t.Fatalf("expected SeverityCritical for unreachable block, got %d", f.Severity)
			}
		}
	}
	if !foundNext {
		t.Fatal("next_block after return should be reported as unreachable")
	}
}

// TestDeadCodeAlternatePathAroundReturn verifies that a block reachable via
// an alternative path is NOT falsely reported as dead.
func TestDeadCodeAlternatePathAroundReturn(t *testing.T) {
	// entry --true-->  retBlock(return) --> join --> exit
	// entry --false--> normBlock ----------> join
	c := NewCFG("alternate_path")
	retBlock := c.CreateBlock("return_block")
	retBlock.AddStatement("return")
	normBlock := c.CreateBlock("normal_block")
	join := c.CreateBlock("join")

	c.ConnectBlocks(c.Entry, retBlock, EdgeCondTrue)
	c.ConnectBlocks(c.Entry, normBlock, EdgeCondFalse)
	c.ConnectBlocks(retBlock, join, EdgeNormal)
	c.ConnectBlocks(normBlock, join, EdgeNormal)
	c.ConnectBlocks(join, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	// join is reachable via normBlock, so it should NOT be reported.
	for _, f := range result.Findings {
		if f.BlockID == join.ID {
			t.Fatalf("join block should not be dead (reachable via alternate path), got %s", f.Reason)
		}
	}
}

// testNoOpClassifier extends testClassifier with no-op detection (";").
type testNoOpClassifier struct {
	testClassifier
}

func (tc *testNoOpClassifier) IsNoOp(stmt any) bool {
	s, ok := stmt.(string)
	return ok && s == ";"
}

func TestDeadCodeSkipsNoOpOnlyBlocks(t *testing.T) {
	c := NewCFG("noop")
	b1 := c.CreateBlock("b1")
	orphan := c.CreateBlock("orphan")
	orphan.AddStatement(";")
	orphan.AddStatement(";")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testNoOpClassifier{}})

	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for no-op-only unreachable block, got %d", len(result.Findings))
	}
}

func TestDeadCodeReportsMixedNoOpBlocks(t *testing.T) {
	c := NewCFG("mixed")
	b1 := c.CreateBlock("b1")
	orphan := c.CreateBlock("orphan")
	orphan.AddStatement(";")
	orphan.AddStatement("call()")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testNoOpClassifier{}})

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding for block with real statements, got %d", len(result.Findings))
	}
}

func TestDeadCodeFindingsAreDeterministicallyOrdered(t *testing.T) {
	c := NewCFG("order")
	b1 := c.CreateBlock("b1")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)
	// Multiple orphan blocks; findings must come back in sorted block-ID order.
	for _, id := range []string{"orphan_c", "orphan_a", "orphan_b"} {
		c.CreateBlock(id)
	}

	result := DetectDeadCode(c, DeadCodeConfig{Classifier: &testClassifier{}})

	if len(result.Findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(result.Findings))
	}
	for i := 1; i < len(result.Findings); i++ {
		if result.Findings[i-1].BlockID > result.Findings[i].BlockID {
			t.Fatalf("findings not sorted by block ID: %s > %s", result.Findings[i-1].BlockID, result.Findings[i].BlockID)
		}
	}
}

// --- Line-level merge pass ---

func lf(start, end int, reason string, sev DeadCodeSeverity) *LineFinding {
	return &LineFinding{StartLine: start, EndLine: end, Reason: reason, Severity: sev, Code: "x"}
}

func TestMergeContiguousFindings(t *testing.T) {
	t.Run("merges overlapping same-reason findings", func(t *testing.T) {
		in := []*LineFinding{
			lf(4, 6, "after_throw", SeverityWarning),
			lf(6, 6, "after_throw", SeverityCritical),
		}
		out := MergeContiguousFindings(in)
		if len(out) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(out))
		}
		if out[0].StartLine != 4 || out[0].EndLine != 6 {
			t.Errorf("expected merged range 4-6, got %d-%d", out[0].StartLine, out[0].EndLine)
		}
		if out[0].Severity != SeverityCritical {
			t.Errorf("expected highest severity to be kept, got %d", out[0].Severity)
		}
	})

	t.Run("merges adjacent same-reason findings", func(t *testing.T) {
		in := []*LineFinding{
			lf(4, 6, "after_throw", SeverityCritical),
			lf(7, 8, "after_throw", SeverityCritical),
		}
		out := MergeContiguousFindings(in)
		if len(out) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(out))
		}
		if out[0].EndLine != 8 {
			t.Errorf("expected merged end line 8, got %d", out[0].EndLine)
		}
	})

	t.Run("does not merge across a gap", func(t *testing.T) {
		in := []*LineFinding{
			lf(4, 4, "after_throw", SeverityCritical),
			lf(6, 6, "after_throw", SeverityCritical),
		}
		out := MergeContiguousFindings(in)
		if len(out) != 2 {
			t.Errorf("expected 2 findings, got %d", len(out))
		}
	})

	t.Run("does not merge different reasons", func(t *testing.T) {
		in := []*LineFinding{
			lf(4, 6, "after_throw", SeverityCritical),
			lf(6, 6, "unreachable", SeverityWarning),
		}
		out := MergeContiguousFindings(in)
		if len(out) != 2 {
			t.Errorf("expected 2 findings, got %d", len(out))
		}
	})
}

func TestSortLineFindings(t *testing.T) {
	in := []*LineFinding{
		lf(5, 9, "unreachable", SeverityCritical),
		lf(2, 8, "unreachable", SeverityCritical),
		lf(2, 3, "unreachable", SeverityCritical),
	}
	SortLineFindings(in)
	if in[0].EndLine != 3 || in[1].EndLine != 8 || in[2].StartLine != 5 {
		t.Errorf("expected sort by StartLine then EndLine, got %+v", in)
	}
}

func TestMergeCodeLines(t *testing.T) {
	if got := mergeCodeLines("", "b"); got != "b" {
		t.Errorf("empty a: got %q", got)
	}
	if got := mergeCodeLines("a", ""); got != "a" {
		t.Errorf("empty b: got %q", got)
	}
	// Duplicated boundary line is emitted once.
	if got := mergeCodeLines("if (x) {\n  dead()", "  dead()\n}"); got != "if (x) {\n  dead()\n}" {
		t.Errorf("boundary dedup failed: got %q", got)
	}
	if got := mergeCodeLines("a", "b"); got != "a\nb" {
		t.Errorf("simple join: got %q", got)
	}
}
