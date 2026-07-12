package dfa

import (
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/core/cfg"
)

// patternRefExtractor extends testRefExtractor semantics with match-pattern
// definitions: "case x:" defines x as DefKindPattern and uses it at the same
// statement position; "use x" is a plain load.
type patternRefExtractor struct{}

func (e *patternRefExtractor) ExtractDefinitions(stmt any, block *cfg.BasicBlock, pos int) []*VarReference {
	s, ok := stmt.(string)
	if !ok {
		return nil
	}
	if name, found := strings.CutPrefix(s, "case "); found {
		name = strings.TrimSuffix(name, ":")
		return []*VarReference{NewVarReference(name, DefKindPattern, block, stmt, pos)}
	}
	parts := strings.SplitN(s, " = ", 2)
	if len(parts) == 2 {
		return []*VarReference{NewVarReference(strings.TrimSpace(parts[0]), DefKindAssign, block, stmt, pos)}
	}
	return nil
}

func (e *patternRefExtractor) ExtractUses(stmt any, block *cfg.BasicBlock, pos int) []*VarReference {
	s, ok := stmt.(string)
	if !ok {
		return nil
	}
	if name, found := strings.CutPrefix(s, "use "); found {
		return []*VarReference{NewVarReference(name, UseKindLoad, block, stmt, pos)}
	}
	if name, found := strings.CutPrefix(s, "case "); found {
		// The pattern statement also uses the captured name at the same position.
		name = strings.TrimSuffix(name, ":")
		return []*VarReference{NewVarReference(name, UseKindLoad, block, stmt, pos)}
	}
	return nil
}

// ExtractParameterDefs seeds comma-separated names from the FunctionNode
// string as parameter defs.
func (e *patternRefExtractor) ExtractParameterDefs(functionNode any, entry *cfg.BasicBlock) []*VarReference {
	s, ok := functionNode.(string)
	if !ok {
		return nil
	}
	var defs []*VarReference
	for _, name := range strings.Split(s, ",") {
		if name = strings.TrimSpace(name); name != "" {
			defs = append(defs, NewVarReference(name, DefKindParam, entry, functionNode, -1))
		}
	}
	return defs
}

func TestDefKindPatternIsDef(t *testing.T) {
	if !DefKindPattern.IsDef() {
		t.Error("DefKindPattern must be classified as a definition")
	}
	if DefKindPattern.IsUse() {
		t.Error("DefKindPattern must not be classified as a use")
	}
	if DefKindPattern.String() != "pattern" {
		t.Errorf("String() = %q, want pattern", DefKindPattern.String())
	}
}

func TestPatternDefReachesSamePositionUse(t *testing.T) {
	c := cfg.NewCFG("match")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("case captured:")
	c.ConnectBlocks(c.Entry, b1, cfg.EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, cfg.EdgeNormal)

	info, err := NewDFABuilder(&patternRefExtractor{}).Build(c)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	chain := info.Chains["captured"]
	if chain == nil {
		t.Fatal("expected chain for pattern-captured variable")
	}
	if len(chain.Pairs) != 1 {
		t.Fatalf("expected same-position pattern def to reach the use, got %d pairs", len(chain.Pairs))
	}
	if chain.Pairs[0].Def.Kind != DefKindPattern {
		t.Errorf("expected pattern def in pair, got %v", chain.Pairs[0].Def.Kind)
	}
}

func TestAssignDefDoesNotReachSamePositionUse(t *testing.T) {
	// A regular assignment at the same position must NOT reach the use
	// (the use happens before/while the def executes).
	def := NewVarReference("x", DefKindAssign, nil, nil, 3)
	if defReachesUseAtPosition(def, 3) {
		t.Error("assign def at same position must not reach the use")
	}
	if !defReachesUseAtPosition(def, 4) {
		t.Error("assign def at earlier position must reach the use")
	}
}

func TestParamExtractorSeedsEntryDefs(t *testing.T) {
	c := cfg.NewCFG("fn")
	c.FunctionNode = "arg1, arg2"
	b1 := c.CreateBlock("b1")
	b1.AddStatement("use arg1")
	c.ConnectBlocks(c.Entry, b1, cfg.EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, cfg.EdgeNormal)

	info, err := NewDFABuilder(&patternRefExtractor{}).Build(c)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	chain := info.Chains["arg1"]
	if chain == nil || len(chain.Defs) != 1 {
		t.Fatal("expected parameter definition seeded for arg1")
	}
	if chain.Defs[0].Kind != DefKindParam {
		t.Errorf("expected DefKindParam, got %v", chain.Defs[0].Kind)
	}
	if len(chain.Pairs) != 1 {
		t.Fatalf("expected parameter def to reach body use, got %d pairs", len(chain.Pairs))
	}
}

func TestBuildWithoutParamExtractorStillWorks(t *testing.T) {
	c := cfg.NewCFG("fn")
	c.FunctionNode = "arg1"
	b1 := c.CreateBlock("b1")
	b1.AddStatement("x = 1")
	c.ConnectBlocks(c.Entry, b1, cfg.EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, cfg.EdgeNormal)

	info, err := NewDFABuilder(&testRefExtractor{}).Build(c)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if info.Chains["arg1"] != nil {
		t.Error("extractor without ParamExtractor must not seed parameter defs")
	}
}
