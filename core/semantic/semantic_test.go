package semantic

import (
	"math"
	"testing"

	"github.com/ludo-technologies/polyscan/core/cfg"
	"github.com/ludo-technologies/polyscan/core/dfa"
)

func TestExtractCFGFeatures_Nil(t *testing.T) {
	f := ExtractCFGFeatures(nil)
	if f.BlockCount != 0 || f.EdgeCount != 0 {
		t.Errorf("nil CFG should produce zero features")
	}
}

func TestExtractCFGFeatures_Simple(t *testing.T) {
	c := cfg.NewCFG("test")
	block := c.CreateBlock("body")
	c.ConnectBlocks(c.Entry, block, cfg.EdgeNormal)
	c.ConnectBlocks(block, c.Exit, cfg.EdgeNormal)

	f := ExtractCFGFeatures(c)
	if f.BlockCount != 3 { // entry, body, exit
		t.Errorf("BlockCount = %d, want 3", f.BlockCount)
	}
	if f.EdgeCount != 2 {
		t.Errorf("EdgeCount = %d, want 2", f.EdgeCount)
	}
	if f.CyclomaticNumber != 1 { // 2 - 3 + 2 = 1
		t.Errorf("CyclomaticNumber = %d, want 1", f.CyclomaticNumber)
	}
}

func TestExtractCFGFeatures_WithBranching(t *testing.T) {
	c := cfg.NewCFG("branch")
	cond := c.CreateBlock("cond")
	thenB := c.CreateBlock("then")
	elseB := c.CreateBlock("else")
	merge := c.CreateBlock("merge")

	c.ConnectBlocks(c.Entry, cond, cfg.EdgeNormal)
	c.ConnectBlocks(cond, thenB, cfg.EdgeCondTrue)
	c.ConnectBlocks(cond, elseB, cfg.EdgeCondFalse)
	c.ConnectBlocks(thenB, merge, cfg.EdgeNormal)
	c.ConnectBlocks(elseB, merge, cfg.EdgeNormal)
	c.ConnectBlocks(merge, c.Exit, cfg.EdgeNormal)

	f := ExtractCFGFeatures(c)
	if f.BlockCount != 6 {
		t.Errorf("BlockCount = %d, want 6", f.BlockCount)
	}
	if f.EdgeCount != 6 {
		t.Errorf("EdgeCount = %d, want 6", f.EdgeCount)
	}
	if f.ConditionalCount != 2 { // true + false
		t.Errorf("ConditionalCount = %d, want 2", f.ConditionalCount)
	}
	if f.BranchingFactor == 0 {
		t.Error("BranchingFactor should be > 0 for branching CFG")
	}
}

func TestExtractCFGFeatures_WithLoop(t *testing.T) {
	c := cfg.NewCFG("loop")
	loopHead := c.CreateBlock("loop_head")
	body := c.CreateBlock("body")

	c.ConnectBlocks(c.Entry, loopHead, cfg.EdgeNormal)
	c.ConnectBlocks(loopHead, body, cfg.EdgeCondTrue)
	c.ConnectBlocks(loopHead, c.Exit, cfg.EdgeCondFalse)
	c.ConnectBlocks(body, loopHead, cfg.EdgeLoop)

	f := ExtractCFGFeatures(c)
	if f.LoopEdgeCount != 1 {
		t.Errorf("LoopEdgeCount = %d, want 1", f.LoopEdgeCount)
	}
}

func TestCompareCFGFeatures_Identical(t *testing.T) {
	f := &CFGFeatures{
		BlockCount:       5,
		EdgeCount:        6,
		CyclomaticNumber: 3,
		EdgeTypeCounts:   map[cfg.EdgeType]int{cfg.EdgeNormal: 4, cfg.EdgeCondTrue: 1, cfg.EdgeCondFalse: 1},
		BranchingFactor:  2.0,
		LoopEdgeCount:    0,
		ConditionalCount: 2,
	}
	sim := CompareCFGFeatures(f, f)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("identical features should have sim ~1.0, got %f", sim)
	}
}

func TestCompareCFGFeatures_Different(t *testing.T) {
	f1 := &CFGFeatures{
		BlockCount:       3,
		EdgeCount:        2,
		CyclomaticNumber: 1,
		EdgeTypeCounts:   map[cfg.EdgeType]int{cfg.EdgeNormal: 2},
		BranchingFactor:  0,
		LoopEdgeCount:    0,
		ConditionalCount: 0,
	}
	f2 := &CFGFeatures{
		BlockCount:       10,
		EdgeCount:        15,
		CyclomaticNumber: 7,
		EdgeTypeCounts:   map[cfg.EdgeType]int{cfg.EdgeNormal: 8, cfg.EdgeCondTrue: 3, cfg.EdgeCondFalse: 3, cfg.EdgeLoop: 1},
		BranchingFactor:  3.0,
		LoopEdgeCount:    1,
		ConditionalCount: 6,
	}
	sim := CompareCFGFeatures(f1, f2)
	if sim >= 0.8 {
		t.Errorf("very different features should have low similarity, got %f", sim)
	}
	if sim < 0 || sim > 1.0 {
		t.Errorf("similarity should be in [0,1], got %f", sim)
	}
}

func TestCompareCFGFeatures_Nil(t *testing.T) {
	f := &CFGFeatures{EdgeTypeCounts: make(map[cfg.EdgeType]int)}
	if sim := CompareCFGFeatures(nil, f); sim != 0.0 {
		t.Errorf("nil should return 0, got %f", sim)
	}
	if sim := CompareCFGFeatures(f, nil); sim != 0.0 {
		t.Errorf("nil should return 0, got %f", sim)
	}
}

func TestCompareDFAFeatures_Identical(t *testing.T) {
	f := &dfa.DFAFeatures{
		PairCount:       5,
		AvgChainLength:  3.0,
		CrossBlockRatio: 0.4,
		DefKindDist:     map[dfa.DefUseKind]int{dfa.DefKindAssign: 3, dfa.DefKindParam: 2},
		UseKindDist:     map[dfa.DefUseKind]int{dfa.UseKindLoad: 5},
	}
	sim := CompareDFAFeatures(f, f)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("identical DFA features should have sim ~1.0, got %f", sim)
	}
}

func TestCompareDFAFeatures_Nil(t *testing.T) {
	f := dfa.NewDFAFeatures()
	if sim := CompareDFAFeatures(nil, f); sim != 0.0 {
		t.Errorf("nil should return 0, got %f", sim)
	}
}

func TestComputeSimilarity_CFGOnly(t *testing.T) {
	c := cfg.NewCFG("test")
	block := c.CreateBlock("body")
	c.ConnectBlocks(c.Entry, block, cfg.EdgeNormal)
	c.ConnectBlocks(block, c.Exit, cfg.EdgeNormal)

	config := DefaultConfig()
	config.EnableDFA = false

	sim := ComputeSimilarity(c, c, nil, nil, config)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("same CFG should have sim ~1.0, got %f", sim)
	}
}

func TestComputeSimilarity_WithDFA(t *testing.T) {
	c := cfg.NewCFG("test")
	block := c.CreateBlock("body")
	c.ConnectBlocks(c.Entry, block, cfg.EdgeNormal)
	c.ConnectBlocks(block, c.Exit, cfg.EdgeNormal)

	dfaInfo := dfa.NewDFAInfo(c)
	ref := dfa.NewVarReference("x", dfa.DefKindAssign, block, nil, 0)
	dfaInfo.AddDef(ref)

	config := DefaultConfig()
	sim := ComputeSimilarity(c, c, dfaInfo, dfaInfo, config)
	if sim < 0.5 || sim > 1.0 {
		t.Errorf("same CFG+DFA should have high similarity, got %f", sim)
	}
}

func TestComputeSimilarity_NilDFA(t *testing.T) {
	c := cfg.NewCFG("test")
	config := DefaultConfig()

	// With DFA enabled but nil DFA info, should fall back to CFG only
	sim := ComputeSimilarity(c, c, nil, nil, config)
	if sim < 0 || sim > 1.0 {
		t.Errorf("similarity should be in [0,1], got %f", sim)
	}
}

func TestComputeSimilarity_NilCFG(t *testing.T) {
	config := DefaultConfig()

	// Both nil: absent data should not look like a perfect match
	if sim := ComputeSimilarity(nil, nil, nil, nil, config); sim != 0.0 {
		t.Errorf("both nil CFGs should return 0.0, got %f", sim)
	}

	c := cfg.NewCFG("test")

	// One nil: still absent, should return 0
	if sim := ComputeSimilarity(nil, c, nil, nil, config); sim != 0.0 {
		t.Errorf("first nil CFG should return 0.0, got %f", sim)
	}
	if sim := ComputeSimilarity(c, nil, nil, nil, config); sim != 0.0 {
		t.Errorf("second nil CFG should return 0.0, got %f", sim)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if !config.EnableDFA {
		t.Error("EnableDFA should be true by default")
	}
	if config.CFGFeatureWeight != 0.60 {
		t.Errorf("CFGFeatureWeight = %f, want 0.60", config.CFGFeatureWeight)
	}
	if config.DFAFeatureWeight != 0.40 {
		t.Errorf("DFAFeatureWeight = %f, want 0.40", config.DFAFeatureWeight)
	}
}

func TestComputeCountSimilarity(t *testing.T) {
	tests := []struct {
		a, b int
		want float64
	}{
		{0, 0, 1.0},
		{5, 5, 1.0},
		{10, 0, 0.0},
		{10, 5, 0.5},
	}
	for _, tt := range tests {
		got := computeCountSimilarity(tt.a, tt.b)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("computeCountSimilarity(%d, %d) = %f, want %f", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCompareEdgeDistributions_Empty(t *testing.T) {
	sim := compareEdgeDistributions(map[cfg.EdgeType]int{}, map[cfg.EdgeType]int{})
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("empty distributions should have sim 1.0, got %f", sim)
	}
}

func TestCompareEdgeDistributions_Identical(t *testing.T) {
	d := map[cfg.EdgeType]int{cfg.EdgeNormal: 3, cfg.EdgeCondTrue: 1}
	sim := compareEdgeDistributions(d, d)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("identical distributions should have sim 1.0, got %f", sim)
	}
}

func TestCompareEdgeDistributions_Orthogonal(t *testing.T) {
	d1 := map[cfg.EdgeType]int{cfg.EdgeNormal: 5}
	d2 := map[cfg.EdgeType]int{cfg.EdgeLoop: 5}
	sim := compareEdgeDistributions(d1, d2)
	if sim != 0.0 {
		t.Errorf("orthogonal distributions should have sim 0.0, got %f", sim)
	}
}
