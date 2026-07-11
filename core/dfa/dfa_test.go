package dfa

import (
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/core/cfg"
)

// ---------------------------------------------------------------------------
// DefUseKind tests
// ---------------------------------------------------------------------------

func TestDefUseKind_String(t *testing.T) {
	tests := []struct {
		kind     DefUseKind
		expected string
	}{
		{DefKindAssign, "assign"},
		{DefKindAugAssign, "aug_assign"},
		{DefKindParam, "param"},
		{DefKindImport, "import"},
		{DefKindFor, "for"},
		{DefKindWith, "with"},
		{DefKindExcept, "except"},
		{DefKindGlobal, "global"},
		{DefKindNonlocal, "nonlocal"},
		{DefKindDelete, "delete"},
		{UseKindLoad, "load"},
		{UseKindAttribute, "attribute"},
		{UseKindCall, "call"},
		{UseKindSubscript, "subscript"},
		{DefUseKind(99), "unknown(99)"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.expected {
			t.Errorf("DefUseKind(%d).String() = %q, want %q", tt.kind, got, tt.expected)
		}
	}
}

func TestDefUseKind_IsDef(t *testing.T) {
	defKinds := []DefUseKind{
		DefKindAssign, DefKindAugAssign, DefKindParam, DefKindImport,
		DefKindFor, DefKindWith, DefKindExcept, DefKindGlobal,
		DefKindNonlocal, DefKindDelete,
	}
	for _, k := range defKinds {
		if !k.IsDef() {
			t.Errorf("%s.IsDef() = false, want true", k)
		}
		if k.IsUse() {
			t.Errorf("%s.IsUse() = true, want false", k)
		}
	}

	useKinds := []DefUseKind{UseKindLoad, UseKindAttribute, UseKindCall, UseKindSubscript}
	for _, k := range useKinds {
		if k.IsDef() {
			t.Errorf("%s.IsDef() = true, want false", k)
		}
		if !k.IsUse() {
			t.Errorf("%s.IsUse() = false, want true", k)
		}
	}

	// Unknown kind is neither def nor use
	unknown := DefUseKind(99)
	if unknown.IsDef() {
		t.Error("unknown.IsDef() = true, want false")
	}
	if unknown.IsUse() {
		t.Error("unknown.IsUse() = true, want false")
	}
}

// ---------------------------------------------------------------------------
// VarReference tests
// ---------------------------------------------------------------------------

func TestNewVarReference(t *testing.T) {
	block := cfg.NewBasicBlock("bb0")
	ref := NewVarReference("x", DefKindAssign, block, "x = 1", 0)

	if ref.Name != "x" {
		t.Errorf("Name = %q, want %q", ref.Name, "x")
	}
	if ref.Kind != DefKindAssign {
		t.Errorf("Kind = %v, want %v", ref.Kind, DefKindAssign)
	}
	if ref.Block != block {
		t.Error("Block mismatch")
	}
	if ref.Statement != "x = 1" {
		t.Errorf("Statement = %v, want %q", ref.Statement, "x = 1")
	}
	if ref.Position != 0 {
		t.Errorf("Position = %d, want 0", ref.Position)
	}
}

// ---------------------------------------------------------------------------
// DefUsePair tests
// ---------------------------------------------------------------------------

func TestDefUsePair_IsCrossBlock_SameBlock(t *testing.T) {
	block := cfg.NewBasicBlock("bb0")
	def := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block, "print(x)", 1)
	pair := NewDefUsePair(def, use)

	if pair.IsCrossBlock() {
		t.Error("Same block pair should not be cross-block")
	}
}

func TestDefUsePair_IsCrossBlock_DifferentBlock(t *testing.T) {
	block1 := cfg.NewBasicBlock("bb0")
	block2 := cfg.NewBasicBlock("bb1")
	def := NewVarReference("x", DefKindAssign, block1, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block2, "print(x)", 0)
	pair := NewDefUsePair(def, use)

	if !pair.IsCrossBlock() {
		t.Error("Different block pair should be cross-block")
	}
}

func TestDefUsePair_IsCrossBlock_NilDef(t *testing.T) {
	block := cfg.NewBasicBlock("bb0")
	use := NewVarReference("x", UseKindLoad, block, "print(x)", 0)
	pair := NewDefUsePair(nil, use)

	if pair.IsCrossBlock() {
		t.Error("Nil def pair should not be cross-block")
	}
}

func TestDefUsePair_IsCrossBlock_NilUse(t *testing.T) {
	block := cfg.NewBasicBlock("bb0")
	def := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	pair := NewDefUsePair(def, nil)

	if pair.IsCrossBlock() {
		t.Error("Nil use pair should not be cross-block")
	}
}

func TestDefUsePair_IsCrossBlock_NilBlock(t *testing.T) {
	def := NewVarReference("x", DefKindAssign, nil, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, nil, "print(x)", 0)
	pair := NewDefUsePair(def, use)

	if pair.IsCrossBlock() {
		t.Error("Nil block pair should not be cross-block")
	}
}

// ---------------------------------------------------------------------------
// DefUseChain tests
// ---------------------------------------------------------------------------

func TestDefUseChain_AddDef(t *testing.T) {
	chain := NewDefUseChain("x")
	block := cfg.NewBasicBlock("bb0")
	ref := NewVarReference("x", DefKindAssign, block, "x = 1", 0)

	chain.AddDef(ref)
	chain.AddDef(nil) // should be ignored

	if len(chain.Defs) != 1 {
		t.Errorf("Expected 1 def, got %d", len(chain.Defs))
	}
	if chain.Defs[0] != ref {
		t.Error("Def mismatch")
	}
}

func TestDefUseChain_AddUse(t *testing.T) {
	chain := NewDefUseChain("x")
	block := cfg.NewBasicBlock("bb0")
	ref := NewVarReference("x", UseKindLoad, block, "print(x)", 0)

	chain.AddUse(ref)
	chain.AddUse(nil) // should be ignored

	if len(chain.Uses) != 1 {
		t.Errorf("Expected 1 use, got %d", len(chain.Uses))
	}
	if chain.Uses[0] != ref {
		t.Error("Use mismatch")
	}
}

func TestDefUseChain_AddPair(t *testing.T) {
	chain := NewDefUseChain("x")
	block := cfg.NewBasicBlock("bb0")
	def := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block, "print(x)", 1)
	pair := NewDefUsePair(def, use)

	chain.AddPair(pair)
	chain.AddPair(nil) // should be ignored

	if len(chain.Pairs) != 1 {
		t.Errorf("Expected 1 pair, got %d", len(chain.Pairs))
	}
	if chain.Pairs[0] != pair {
		t.Error("Pair mismatch")
	}
}

func TestNewDefUseChain(t *testing.T) {
	chain := NewDefUseChain("y")
	if chain.Variable != "y" {
		t.Errorf("Variable = %q, want %q", chain.Variable, "y")
	}
	if len(chain.Defs) != 0 {
		t.Error("Expected empty defs")
	}
	if len(chain.Uses) != 0 {
		t.Error("Expected empty uses")
	}
	if len(chain.Pairs) != 0 {
		t.Error("Expected empty pairs")
	}
}

// ---------------------------------------------------------------------------
// DFAInfo tests
// ---------------------------------------------------------------------------

func TestDFAInfo_AddDef(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block := cfg.NewBasicBlock("bb0")
	ref := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	info.AddDef(ref)
	info.AddDef(nil) // should be ignored

	if info.TotalDefs() != 1 {
		t.Errorf("TotalDefs = %d, want 1", info.TotalDefs())
	}
	if len(info.BlockDefs["bb0"]) != 1 {
		t.Errorf("BlockDefs[bb0] = %d, want 1", len(info.BlockDefs["bb0"]))
	}
}

func TestDFAInfo_AddUse(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block := cfg.NewBasicBlock("bb0")
	ref := NewVarReference("x", UseKindLoad, block, "print(x)", 0)
	info.AddUse(ref)
	info.AddUse(nil) // should be ignored

	if info.TotalUses() != 1 {
		t.Errorf("TotalUses = %d, want 1", info.TotalUses())
	}
	if len(info.BlockUses["bb0"]) != 1 {
		t.Errorf("BlockUses[bb0] = %d, want 1", len(info.BlockUses["bb0"]))
	}
}

func TestDFAInfo_AddDef_NilBlock(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	ref := NewVarReference("x", DefKindAssign, nil, "x = 1", 0)
	info.AddDef(ref)

	if info.TotalDefs() != 1 {
		t.Errorf("TotalDefs = %d, want 1", info.TotalDefs())
	}
	// Should not have any block defs since block is nil
	if len(info.BlockDefs) != 0 {
		t.Errorf("Expected no BlockDefs entries, got %d", len(info.BlockDefs))
	}
}

func TestDFAInfo_GetChain(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	chain := info.GetChain("x")
	if chain == nil {
		t.Fatal("Expected non-nil chain")
	}
	if chain.Variable != "x" {
		t.Errorf("Variable = %q, want %q", chain.Variable, "x")
	}

	// Getting the same variable returns the same chain
	chain2 := info.GetChain("x")
	if chain2 != chain {
		t.Error("GetChain should return the same chain for the same variable")
	}
}

func TestDFAInfo_UniqueVariables(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block := cfg.NewBasicBlock("bb0")
	info.AddDef(NewVarReference("x", DefKindAssign, block, "x = 1", 0))
	info.AddDef(NewVarReference("y", DefKindAssign, block, "y = 2", 1))
	info.AddUse(NewVarReference("x", UseKindLoad, block, "print(x)", 2))

	if info.UniqueVariables() != 2 {
		t.Errorf("UniqueVariables = %d, want 2", info.UniqueVariables())
	}
}

func TestDFAInfo_TotalPairs(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	if info.TotalPairs() != 0 {
		t.Errorf("Empty TotalPairs = %d, want 0", info.TotalPairs())
	}

	block := cfg.NewBasicBlock("bb0")
	def := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block, "print(x)", 1)
	chain := info.GetChain("x")
	chain.AddDef(def)
	chain.AddUse(use)
	chain.AddPair(NewDefUsePair(def, use))

	if info.TotalPairs() != 1 {
		t.Errorf("TotalPairs = %d, want 1", info.TotalPairs())
	}
}

func TestNewDFAInfo(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	if info.CFG != c {
		t.Error("CFG mismatch")
	}
	if info.TotalDefs() != 0 {
		t.Errorf("TotalDefs = %d, want 0", info.TotalDefs())
	}
	if info.TotalUses() != 0 {
		t.Errorf("TotalUses = %d, want 0", info.TotalUses())
	}
	if info.UniqueVariables() != 0 {
		t.Errorf("UniqueVariables = %d, want 0", info.UniqueVariables())
	}
}

// ---------------------------------------------------------------------------
// DFAFeatures tests
// ---------------------------------------------------------------------------

func TestExtractDFAFeatures_Nil(t *testing.T) {
	features := ExtractDFAFeatures(nil)
	if features.PairCount != 0 {
		t.Errorf("PairCount = %d, want 0", features.PairCount)
	}
	if features.AvgChainLength != 0 {
		t.Errorf("AvgChainLength = %f, want 0", features.AvgChainLength)
	}
	if features.CrossBlockRatio != 0 {
		t.Errorf("CrossBlockRatio = %f, want 0", features.CrossBlockRatio)
	}
}

func TestExtractDFAFeatures_Empty(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)
	features := ExtractDFAFeatures(info)

	if features.PairCount != 0 {
		t.Errorf("PairCount = %d, want 0", features.PairCount)
	}
	if features.AvgChainLength != 0 {
		t.Errorf("AvgChainLength = %f, want 0", features.AvgChainLength)
	}
}

func TestExtractDFAFeatures_IntraBlock(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block := cfg.NewBasicBlock("bb0")
	def := NewVarReference("x", DefKindAssign, block, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block, "print(x)", 1)
	info.AddDef(def)
	info.AddUse(use)

	chain := info.GetChain("x")
	chain.AddPair(NewDefUsePair(def, use))

	features := ExtractDFAFeatures(info)
	if features.PairCount != 1 {
		t.Errorf("PairCount = %d, want 1", features.PairCount)
	}
	if features.CrossBlockRatio != 0 {
		t.Errorf("CrossBlockRatio = %f, want 0 (intra-block pair)", features.CrossBlockRatio)
	}
	// chain length: 1 def + 1 use = 2, 1 chain => avg = 2.0
	if features.AvgChainLength != 2.0 {
		t.Errorf("AvgChainLength = %f, want 2.0", features.AvgChainLength)
	}
	if features.DefKindDist[DefKindAssign] != 1 {
		t.Errorf("DefKindDist[assign] = %d, want 1", features.DefKindDist[DefKindAssign])
	}
	if features.UseKindDist[UseKindLoad] != 1 {
		t.Errorf("UseKindDist[load] = %d, want 1", features.UseKindDist[UseKindLoad])
	}
}

func TestExtractDFAFeatures_CrossBlock(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block1 := cfg.NewBasicBlock("bb0")
	block2 := cfg.NewBasicBlock("bb1")
	def := NewVarReference("x", DefKindAssign, block1, "x = 1", 0)
	use := NewVarReference("x", UseKindLoad, block2, "print(x)", 0)
	info.AddDef(def)
	info.AddUse(use)

	chain := info.GetChain("x")
	chain.AddPair(NewDefUsePair(def, use))

	features := ExtractDFAFeatures(info)
	if features.CrossBlockRatio != 1.0 {
		t.Errorf("CrossBlockRatio = %f, want 1.0", features.CrossBlockRatio)
	}
}

func TestExtractDFAFeatures_MultipleVariables(t *testing.T) {
	c := cfg.NewCFG("test")
	info := NewDFAInfo(c)

	block := cfg.NewBasicBlock("bb0")
	info.AddDef(NewVarReference("x", DefKindAssign, block, "x = 1", 0))
	info.AddDef(NewVarReference("y", DefKindParam, block, "y param", 1))
	info.AddUse(NewVarReference("x", UseKindCall, block, "x()", 2))
	info.AddUse(NewVarReference("y", UseKindAttribute, block, "y.attr", 3))

	features := ExtractDFAFeatures(info)
	// 2 variables, each with 1 def + 1 use = 2 refs => avg = 2.0
	if features.AvgChainLength != 2.0 {
		t.Errorf("AvgChainLength = %f, want 2.0", features.AvgChainLength)
	}
	if features.DefKindDist[DefKindAssign] != 1 {
		t.Errorf("DefKindDist[assign] = %d, want 1", features.DefKindDist[DefKindAssign])
	}
	if features.DefKindDist[DefKindParam] != 1 {
		t.Errorf("DefKindDist[param] = %d, want 1", features.DefKindDist[DefKindParam])
	}
	if features.UseKindDist[UseKindCall] != 1 {
		t.Errorf("UseKindDist[call] = %d, want 1", features.UseKindDist[UseKindCall])
	}
	if features.UseKindDist[UseKindAttribute] != 1 {
		t.Errorf("UseKindDist[attribute] = %d, want 1", features.UseKindDist[UseKindAttribute])
	}
}

func TestNewDFAFeatures(t *testing.T) {
	f := NewDFAFeatures()
	if f.PairCount != 0 {
		t.Errorf("PairCount = %d, want 0", f.PairCount)
	}
	if f.DefKindDist == nil {
		t.Error("DefKindDist should not be nil")
	}
	if f.UseKindDist == nil {
		t.Error("UseKindDist should not be nil")
	}
}

// ---------------------------------------------------------------------------
// DFABuilder tests
// ---------------------------------------------------------------------------

// testRefExtractor is a simple RefExtractor for testing.
// It recognizes string statements:
//   - "x = 1" => definition of "x" (DefKindAssign)
//   - "print(x)" => use of "x" (UseKindCall)
type testRefExtractor struct{}

func (e *testRefExtractor) ExtractDefinitions(stmt any, block *cfg.BasicBlock, pos int) []*VarReference {
	s, ok := stmt.(string)
	if !ok {
		return nil
	}
	// Pattern: "varname = value"
	parts := strings.SplitN(s, " = ", 2)
	if len(parts) == 2 {
		varName := strings.TrimSpace(parts[0])
		return []*VarReference{NewVarReference(varName, DefKindAssign, block, stmt, pos)}
	}
	return nil
}

func (e *testRefExtractor) ExtractUses(stmt any, block *cfg.BasicBlock, pos int) []*VarReference {
	s, ok := stmt.(string)
	if !ok {
		return nil
	}
	// Pattern: "print(varname)"
	if strings.HasPrefix(s, "print(") && strings.HasSuffix(s, ")") {
		varName := s[6 : len(s)-1]
		return []*VarReference{NewVarReference(varName, UseKindCall, block, stmt, pos)}
	}
	return nil
}

func TestDFABuilder_Build_NilExtractor(t *testing.T) {
	builder := NewDFABuilder(nil)

	// nil CFG should still succeed (short-circuit before using extractor)
	info, err := builder.Build(nil)
	if err != nil {
		t.Fatalf("Build(nil CFG) with nil extractor should succeed, got: %v", err)
	}
	if info == nil {
		t.Fatal("Expected non-nil DFAInfo for nil CFG")
	}

	// non-nil CFG should return an error, not panic
	c := cfg.NewCFG("test")
	_, err = builder.Build(c)
	if err == nil {
		t.Fatal("Build with nil extractor and non-nil CFG should return an error")
	}
}

func TestDFABuilder_Build_NilCFG(t *testing.T) {
	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(nil)
	if err != nil {
		t.Fatalf("Build(nil) error: %v", err)
	}
	if info == nil {
		t.Fatal("Expected non-nil DFAInfo")
	}
	if info.TotalDefs() != 0 {
		t.Errorf("TotalDefs = %d, want 0", info.TotalDefs())
	}
}

func TestDFABuilder_Build_EmptyCFG(t *testing.T) {
	c := cfg.NewCFG("empty")
	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if info.TotalDefs() != 0 {
		t.Errorf("TotalDefs = %d, want 0", info.TotalDefs())
	}
	if info.TotalUses() != 0 {
		t.Errorf("TotalUses = %d, want 0", info.TotalUses())
	}
}

func TestDFABuilder_Build_SameBlockDefUse(t *testing.T) {
	// CFG: entry -> block1 -> exit
	// block1: "x = 1", "print(x)"
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block1.AddStatement("x = 1")
	block1.AddStatement("print(x)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalDefs() != 1 {
		t.Errorf("TotalDefs = %d, want 1", info.TotalDefs())
	}
	if info.TotalUses() != 1 {
		t.Errorf("TotalUses = %d, want 1", info.TotalUses())
	}
	if info.UniqueVariables() != 1 {
		t.Errorf("UniqueVariables = %d, want 1", info.UniqueVariables())
	}
	if info.TotalPairs() != 1 {
		t.Errorf("TotalPairs = %d, want 1", info.TotalPairs())
	}

	chain := info.GetChain("x")
	if len(chain.Pairs) != 1 {
		t.Fatalf("Expected 1 pair for x, got %d", len(chain.Pairs))
	}
	pair := chain.Pairs[0]
	if pair.IsCrossBlock() {
		t.Error("Same-block pair should not be cross-block")
	}
	if pair.Def.Position != 0 {
		t.Errorf("Def position = %d, want 0", pair.Def.Position)
	}
	if pair.Use.Position != 1 {
		t.Errorf("Use position = %d, want 1", pair.Use.Position)
	}
}

func TestDFABuilder_Build_CrossBlockDefUse(t *testing.T) {
	// CFG: entry -> block1 -> block2 -> exit
	// block1: "x = 1"
	// block2: "print(x)"
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block2 := c.CreateBlock("block2")
	block1.AddStatement("x = 1")
	block2.AddStatement("print(x)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, block2, cfg.EdgeNormal)
	c.ConnectBlocks(block2, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalDefs() != 1 {
		t.Errorf("TotalDefs = %d, want 1", info.TotalDefs())
	}
	if info.TotalUses() != 1 {
		t.Errorf("TotalUses = %d, want 1", info.TotalUses())
	}
	if info.TotalPairs() != 1 {
		t.Errorf("TotalPairs = %d, want 1", info.TotalPairs())
	}

	chain := info.GetChain("x")
	if len(chain.Pairs) != 1 {
		t.Fatalf("Expected 1 pair for x, got %d", len(chain.Pairs))
	}
	pair := chain.Pairs[0]
	if !pair.IsCrossBlock() {
		t.Error("Cross-block pair should be cross-block")
	}
}

func TestDFABuilder_Build_MultipleVariables(t *testing.T) {
	// CFG: entry -> block1 -> block2 -> exit
	// block1: "x = 1", "y = 2"
	// block2: "print(x)", "print(y)"
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block2 := c.CreateBlock("block2")
	block1.AddStatement("x = 1")
	block1.AddStatement("y = 2")
	block2.AddStatement("print(x)")
	block2.AddStatement("print(y)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, block2, cfg.EdgeNormal)
	c.ConnectBlocks(block2, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalDefs() != 2 {
		t.Errorf("TotalDefs = %d, want 2", info.TotalDefs())
	}
	if info.TotalUses() != 2 {
		t.Errorf("TotalUses = %d, want 2", info.TotalUses())
	}
	if info.UniqueVariables() != 2 {
		t.Errorf("UniqueVariables = %d, want 2", info.UniqueVariables())
	}
	if info.TotalPairs() != 2 {
		t.Errorf("TotalPairs = %d, want 2", info.TotalPairs())
	}
}

func TestDFABuilder_Build_RedefInBlock(t *testing.T) {
	// block1: "x = 1", "x = 2", "print(x)"
	// The use should link to the second (later) definition.
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block1.AddStatement("x = 1")
	block1.AddStatement("x = 2")
	block1.AddStatement("print(x)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalDefs() != 2 {
		t.Errorf("TotalDefs = %d, want 2", info.TotalDefs())
	}

	chain := info.GetChain("x")
	if len(chain.Pairs) != 1 {
		t.Fatalf("Expected 1 pair, got %d", len(chain.Pairs))
	}
	// Should link to the second definition (position 1)
	if chain.Pairs[0].Def.Position != 1 {
		t.Errorf("Reaching def position = %d, want 1 (latest def before use)", chain.Pairs[0].Def.Position)
	}
}

func TestDFABuilder_Build_NoDefForUse(t *testing.T) {
	// block1: "print(x)" — no definition of x exists
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block1.AddStatement("print(x)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalUses() != 1 {
		t.Errorf("TotalUses = %d, want 1", info.TotalUses())
	}
	if info.TotalPairs() != 0 {
		t.Errorf("TotalPairs = %d, want 0 (no reaching def)", info.TotalPairs())
	}
}

func TestDFABuilder_Build_PredecessorChain(t *testing.T) {
	// entry -> block1 -> block2 -> block3 -> exit
	// block1: "x = 1"
	// block2: (empty)
	// block3: "print(x)"
	// The BFS should traverse through block2 to find x's def in block1.
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block2 := c.CreateBlock("block2")
	block3 := c.CreateBlock("block3")
	block1.AddStatement("x = 1")
	block3.AddStatement("print(x)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, block2, cfg.EdgeNormal)
	c.ConnectBlocks(block2, block3, cfg.EdgeNormal)
	c.ConnectBlocks(block3, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if info.TotalPairs() != 1 {
		t.Errorf("TotalPairs = %d, want 1", info.TotalPairs())
	}

	chain := info.GetChain("x")
	if len(chain.Pairs) != 1 {
		t.Fatalf("Expected 1 pair, got %d", len(chain.Pairs))
	}
	if !chain.Pairs[0].IsCrossBlock() {
		t.Error("Expected cross-block pair")
	}
}

func TestDFABuilder_Build_FeatureExtraction(t *testing.T) {
	// Integration test: build DFA then extract features
	c := cfg.NewCFG("test")
	block1 := c.CreateBlock("block1")
	block2 := c.CreateBlock("block2")
	block1.AddStatement("x = 1")
	block1.AddStatement("y = 2")
	block2.AddStatement("print(x)")
	block2.AddStatement("print(y)")
	c.ConnectBlocks(c.Entry, block1, cfg.EdgeNormal)
	c.ConnectBlocks(block1, block2, cfg.EdgeNormal)
	c.ConnectBlocks(block2, c.Exit, cfg.EdgeNormal)

	builder := NewDFABuilder(&testRefExtractor{})
	info, err := builder.Build(c)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	features := ExtractDFAFeatures(info)
	if features.PairCount != 2 {
		t.Errorf("PairCount = %d, want 2", features.PairCount)
	}
	if features.CrossBlockRatio != 1.0 {
		t.Errorf("CrossBlockRatio = %f, want 1.0 (all cross-block)", features.CrossBlockRatio)
	}
	// 2 variables, each with 1 def + 1 use = 2 => avg = 2.0
	if features.AvgChainLength != 2.0 {
		t.Errorf("AvgChainLength = %f, want 2.0", features.AvgChainLength)
	}
	if features.DefKindDist[DefKindAssign] != 2 {
		t.Errorf("DefKindDist[assign] = %d, want 2", features.DefKindDist[DefKindAssign])
	}
	if features.UseKindDist[UseKindCall] != 2 {
		t.Errorf("UseKindDist[call] = %d, want 2", features.UseKindDist[UseKindCall])
	}
}
