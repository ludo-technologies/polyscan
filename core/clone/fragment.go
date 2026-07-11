package clone

import (
	"fmt"

	"github.com/ludo-technologies/polyscan/core/apted"
	"github.com/ludo-technologies/polyscan/core/domain"
)

// CodeFragment represents a code fragment for clone detection.
// It implements GroupableItem so it can be used with the grouping framework.
type CodeFragment struct {
	ID         int
	FilePath   string
	StartLine  int
	EndLine    int
	StartCol   int
	EndCol     int
	Content    string
	Hash       string // Hex hash of Type-1 normalized content; "" when no source content
	ASTNode    *apted.TreeNode
	NodeCount  int
	LineCount  int
	Complexity int      // Cyclomatic complexity (if applicable)
	Features   []string // Detector-populated clone feature cache for this fragment's tree
}

// ItemID returns the fragment's unique ID for GroupableItem.
func (f *CodeFragment) ItemID() int {
	return f.ID
}

// ItemLocation returns the fragment's source location for GroupableItem.
func (f *CodeFragment) ItemLocation() ItemLocation {
	return ItemLocation{
		FilePath:  f.FilePath,
		StartLine: f.StartLine,
		EndLine:   f.EndLine,
		StartCol:  f.StartCol,
		EndCol:    f.EndCol,
	}
}

// ItemKey returns a stable location-based key for the fragment.
func (f *CodeFragment) ItemKey() string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", f.FilePath, f.StartLine, f.EndLine, f.StartCol, f.EndCol)
}

// ClonePair represents a detected clone pair between two code fragments.
type ClonePair struct {
	Fragment1    *CodeFragment
	Fragment2    *CodeFragment
	Similarity   float64
	CloneType    domain.CloneType
	Confidence   float64
	AnalyzerName string
}

// CloneGroup represents a group of code fragments that are clones of each other.
type CloneGroup struct {
	ID            int
	Fragments     []*CodeFragment
	CloneType     domain.CloneType
	AvgSimilarity float64
}

// CloneStatistics holds aggregate statistics about clone detection results.
type CloneStatistics struct {
	TotalFragments int
	TotalPairs     int
	TypeCounts     map[domain.CloneType]int
	AvgSimilarity  float64
}
