package clone

import (
	"fmt"

	"github.com/ludo-technologies/codescan-core/apted"
	"github.com/ludo-technologies/codescan-core/domain"
)

// CodeFragment represents a code fragment for clone detection.
// It implements GroupableItem so it can be used with the grouping framework.
type CodeFragment struct {
	ID        int
	FilePath  string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
	Content   string
	ASTNode   *apted.TreeNode
	NodeCount int
	LineCount int
}

// ItemID returns the fragment's unique ID for GroupableItem.
func (f *CodeFragment) ItemID() int {
	return f.ID
}

// ItemKey returns the sorting key for GroupableItem.
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
