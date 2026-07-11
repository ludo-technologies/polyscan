package dfa

import (
	"errors"

	"github.com/ludo-technologies/polyscan/core/cfg"
)

// RefExtractor is the language-specific interface for extracting variable references.
// Language adapters (e.g. pyscn, jscan) implement this to extract definitions and uses
// from their AST nodes stored in BasicBlock.Statements.
type RefExtractor interface {
	ExtractDefinitions(stmt any, block *cfg.BasicBlock, pos int) []*VarReference
	ExtractUses(stmt any, block *cfg.BasicBlock, pos int) []*VarReference
}

// DFABuilder builds DFA information from a CFG using a language-specific RefExtractor.
type DFABuilder struct {
	extractor RefExtractor
}

// NewDFABuilder creates a new DFABuilder with the given reference extractor.
func NewDFABuilder(extractor RefExtractor) *DFABuilder {
	return &DFABuilder{extractor: extractor}
}

// Build performs data flow analysis on the given CFG.
// It collects definitions, collects uses, then links def-use pairs.
func (b *DFABuilder) Build(c *cfg.CFG) (*DFAInfo, error) {
	if c == nil {
		return NewDFAInfo(nil), nil
	}
	if b.extractor == nil {
		return nil, errors.New("dfa: RefExtractor is nil")
	}

	info := NewDFAInfo(c)

	// Phase 1: Collect definitions from all blocks
	b.collectDefinitions(c, info)

	// Phase 2: Collect uses from all blocks
	b.collectUses(c, info)

	// Phase 3: Link definitions to uses (reaching definitions)
	b.linkDefUse(c, info)

	return info, nil
}

// collectDefinitions extracts all variable definitions from the CFG.
func (b *DFABuilder) collectDefinitions(c *cfg.CFG, info *DFAInfo) {
	for _, block := range c.Blocks {
		for pos, stmt := range block.Statements {
			defs := b.extractor.ExtractDefinitions(stmt, block, pos)
			for _, def := range defs {
				info.AddDef(def)
			}
		}
	}
}

// collectUses extracts all variable uses from the CFG.
func (b *DFABuilder) collectUses(c *cfg.CFG, info *DFAInfo) {
	for _, block := range c.Blocks {
		for pos, stmt := range block.Statements {
			uses := b.extractor.ExtractUses(stmt, block, pos)
			for _, use := range uses {
				info.AddUse(use)
			}
		}
	}
}

// linkDefUse links definitions to uses by finding reaching definitions.
// Uses an approximate reaching definition algorithm:
// 1. Look for definitions in the same block before the use
// 2. If none found, BFS through predecessor blocks
func (b *DFABuilder) linkDefUse(_ *cfg.CFG, info *DFAInfo) {
	for varName, chain := range info.Chains {
		for _, use := range chain.Uses {
			def := b.findReachingDef(varName, use, info)
			if def != nil {
				pair := NewDefUsePair(def, use)
				chain.AddPair(pair)
			}
		}
	}
}

// findReachingDef finds the reaching definition for a use.
func (b *DFABuilder) findReachingDef(varName string, use *VarReference, info *DFAInfo) *VarReference {
	if use.Block == nil {
		return nil
	}

	// First: look in the same block before the use position
	def := b.findDefInBlockBefore(varName, use.Block, use.Position, info)
	if def != nil {
		return def
	}

	// Second: BFS through predecessor blocks
	return b.findDefInPredecessors(varName, use.Block, info)
}

// findDefInBlockBefore finds the latest definition of varName in block before the given position.
func (b *DFABuilder) findDefInBlockBefore(varName string, block *cfg.BasicBlock, beforePos int, info *DFAInfo) *VarReference {
	defs := info.BlockDefs[block.ID]
	var latest *VarReference
	for _, def := range defs {
		if def.Name == varName && def.Position < beforePos {
			if latest == nil || def.Position > latest.Position {
				latest = def
			}
		}
	}
	return latest
}

// findDefInPredecessors searches predecessor blocks for the most recent definition using BFS.
func (b *DFABuilder) findDefInPredecessors(varName string, block *cfg.BasicBlock, info *DFAInfo) *VarReference {
	visited := make(map[string]bool)
	visited[block.ID] = true

	queue := make([]*cfg.BasicBlock, 0)
	for _, edge := range block.Predecessors {
		if edge.From != nil && !visited[edge.From.ID] {
			queue = append(queue, edge.From)
			visited[edge.From.ID] = true
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Look for the latest def in this block (any position)
		defs := info.BlockDefs[current.ID]
		var latest *VarReference
		for _, def := range defs {
			if def.Name == varName {
				if latest == nil || def.Position > latest.Position {
					latest = def
				}
			}
		}
		if latest != nil {
			return latest
		}

		// Continue searching predecessors
		for _, edge := range current.Predecessors {
			if edge.From != nil && !visited[edge.From.ID] {
				queue = append(queue, edge.From)
				visited[edge.From.ID] = true
			}
		}
	}

	return nil
}
