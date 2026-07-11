package dfa

import (
	"fmt"
	"github.com/ludo-technologies/codescan-core/cfg"
)

// DefUseKind represents the kind of definition or use of a variable.
type DefUseKind int

const (
	DefKindAssign    DefUseKind = iota // Assignment (x = ...)
	DefKindAugAssign                    // Augmented assignment (x += ...)
	DefKindParam                        // Function parameter
	DefKindImport                       // Import statement
	DefKindFor                          // For-loop variable
	DefKindWith                         // With-statement variable
	DefKindExcept                       // Exception handler variable
	DefKindGlobal                       // Global declaration
	DefKindNonlocal                     // Nonlocal declaration
	DefKindDelete                       // Delete statement
	UseKindLoad                         // Simple name load
	UseKindAttribute                    // Attribute access
	UseKindCall                         // Function call
	UseKindSubscript                    // Subscript access
)

// String returns the string representation of a DefUseKind.
func (k DefUseKind) String() string {
	switch k {
	case DefKindAssign:
		return "assign"
	case DefKindAugAssign:
		return "aug_assign"
	case DefKindParam:
		return "param"
	case DefKindImport:
		return "import"
	case DefKindFor:
		return "for"
	case DefKindWith:
		return "with"
	case DefKindExcept:
		return "except"
	case DefKindGlobal:
		return "global"
	case DefKindNonlocal:
		return "nonlocal"
	case DefKindDelete:
		return "delete"
	case UseKindLoad:
		return "load"
	case UseKindAttribute:
		return "attribute"
	case UseKindCall:
		return "call"
	case UseKindSubscript:
		return "subscript"
	default:
		return fmt.Sprintf("unknown(%d)", int(k))
	}
}

// IsDef returns true if this kind represents a definition.
func (k DefUseKind) IsDef() bool {
	return k >= DefKindAssign && k <= DefKindDelete
}

// IsUse returns true if this kind represents a use.
func (k DefUseKind) IsUse() bool {
	return k >= UseKindLoad && k <= UseKindSubscript
}

// VarReference represents a reference to a variable (definition or use).
type VarReference struct {
	Name      string
	Kind      DefUseKind
	Block     *cfg.BasicBlock
	Statement any
	Position  int
}

// NewVarReference creates a new variable reference.
func NewVarReference(name string, kind DefUseKind, block *cfg.BasicBlock, stmt any, pos int) *VarReference {
	return &VarReference{
		Name:      name,
		Kind:      kind,
		Block:     block,
		Statement: stmt,
		Position:  pos,
	}
}

// DefUsePair represents a definition-use pair.
type DefUsePair struct {
	Def *VarReference
	Use *VarReference
}

// NewDefUsePair creates a new def-use pair.
func NewDefUsePair(def, use *VarReference) *DefUsePair {
	return &DefUsePair{Def: def, Use: use}
}

// IsCrossBlock returns true if the def and use are in different blocks.
func (p *DefUsePair) IsCrossBlock() bool {
	if p.Def == nil || p.Use == nil {
		return false
	}
	if p.Def.Block == nil || p.Use.Block == nil {
		return false
	}
	return p.Def.Block.ID != p.Use.Block.ID
}

// DefUseChain represents a chain of definitions and uses for a single variable.
type DefUseChain struct {
	Variable string
	Defs     []*VarReference
	Uses     []*VarReference
	Pairs    []*DefUsePair
}

// NewDefUseChain creates a new def-use chain for the given variable.
func NewDefUseChain(variable string) *DefUseChain {
	return &DefUseChain{
		Variable: variable,
		Defs:     []*VarReference{},
		Uses:     []*VarReference{},
		Pairs:    []*DefUsePair{},
	}
}

// AddDef adds a definition to the chain.
func (c *DefUseChain) AddDef(ref *VarReference) {
	if ref != nil {
		c.Defs = append(c.Defs, ref)
	}
}

// AddUse adds a use to the chain.
func (c *DefUseChain) AddUse(ref *VarReference) {
	if ref != nil {
		c.Uses = append(c.Uses, ref)
	}
}

// AddPair adds a def-use pair to the chain.
func (c *DefUseChain) AddPair(pair *DefUsePair) {
	if pair != nil {
		c.Pairs = append(c.Pairs, pair)
	}
}

// DFAInfo holds the complete data flow analysis results for a CFG.
type DFAInfo struct {
	CFG       *cfg.CFG
	Chains    map[string]*DefUseChain
	BlockDefs map[string][]*VarReference // block ID -> definitions
	BlockUses map[string][]*VarReference // block ID -> uses
}

// NewDFAInfo creates a new DFAInfo for the given CFG.
func NewDFAInfo(c *cfg.CFG) *DFAInfo {
	return &DFAInfo{
		CFG:       c,
		Chains:    make(map[string]*DefUseChain),
		BlockDefs: make(map[string][]*VarReference),
		BlockUses: make(map[string][]*VarReference),
	}
}

// GetChain returns the def-use chain for the given variable, creating it if needed.
func (d *DFAInfo) GetChain(variable string) *DefUseChain {
	chain, ok := d.Chains[variable]
	if !ok {
		chain = NewDefUseChain(variable)
		d.Chains[variable] = chain
	}
	return chain
}

// AddDef records a definition in the DFA info.
func (d *DFAInfo) AddDef(ref *VarReference) {
	if ref == nil {
		return
	}
	chain := d.GetChain(ref.Name)
	chain.AddDef(ref)
	if ref.Block != nil {
		d.BlockDefs[ref.Block.ID] = append(d.BlockDefs[ref.Block.ID], ref)
	}
}

// AddUse records a use in the DFA info.
func (d *DFAInfo) AddUse(ref *VarReference) {
	if ref == nil {
		return
	}
	chain := d.GetChain(ref.Name)
	chain.AddUse(ref)
	if ref.Block != nil {
		d.BlockUses[ref.Block.ID] = append(d.BlockUses[ref.Block.ID], ref)
	}
}

// TotalDefs returns the total number of definitions across all chains.
func (d *DFAInfo) TotalDefs() int {
	total := 0
	for _, chain := range d.Chains {
		total += len(chain.Defs)
	}
	return total
}

// TotalUses returns the total number of uses across all chains.
func (d *DFAInfo) TotalUses() int {
	total := 0
	for _, chain := range d.Chains {
		total += len(chain.Uses)
	}
	return total
}

// TotalPairs returns the total number of def-use pairs across all chains.
func (d *DFAInfo) TotalPairs() int {
	total := 0
	for _, chain := range d.Chains {
		total += len(chain.Pairs)
	}
	return total
}

// UniqueVariables returns the number of unique variables.
func (d *DFAInfo) UniqueVariables() int {
	return len(d.Chains)
}

// DFAFeatures holds extracted DFA feature values for similarity comparison.
type DFAFeatures struct {
	PairCount       int
	AvgChainLength  float64
	CrossBlockRatio float64
	DefKindDist     map[DefUseKind]int
	UseKindDist     map[DefUseKind]int
}

// NewDFAFeatures creates an empty DFAFeatures.
func NewDFAFeatures() *DFAFeatures {
	return &DFAFeatures{
		DefKindDist: make(map[DefUseKind]int),
		UseKindDist: make(map[DefUseKind]int),
	}
}

// ExtractDFAFeatures extracts DFA features from DFAInfo for similarity comparison.
func ExtractDFAFeatures(info *DFAInfo) *DFAFeatures {
	if info == nil {
		return NewDFAFeatures()
	}

	features := NewDFAFeatures()
	features.PairCount = info.TotalPairs()

	// Average chain length
	totalChainLen := 0
	chainCount := 0
	for _, chain := range info.Chains {
		totalChainLen += len(chain.Defs) + len(chain.Uses)
		chainCount++
	}
	if chainCount > 0 {
		features.AvgChainLength = float64(totalChainLen) / float64(chainCount)
	}

	// Cross-block ratio
	crossBlock := 0
	for _, chain := range info.Chains {
		for _, pair := range chain.Pairs {
			if pair.IsCrossBlock() {
				crossBlock++
			}
		}
	}
	if features.PairCount > 0 {
		features.CrossBlockRatio = float64(crossBlock) / float64(features.PairCount)
	}

	// Def/Use kind distributions
	for _, chain := range info.Chains {
		for _, def := range chain.Defs {
			features.DefKindDist[def.Kind]++
		}
		for _, use := range chain.Uses {
			features.UseKindDist[use.Kind]++
		}
	}

	return features
}
