package analysis

import (
	"path/filepath"

	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
)

// scopedKey identifies one type the way the coupling analysis keys types:
// language, location and name, where the location is the directory for a
// language whose types span one, else the file.
func scopedKey(language *engine.Language, location, name string) string {
	return language.Name + "\x00" + location + "\x00" + name
}

// locationOf returns the location a type of the language belongs to: the
// directory when the language spreads a type over one, otherwise the file.
func locationOf(language *engine.Language, display string) string {
	if language.TypeSpansDirectory {
		return filepath.Dir(display)
	}
	return display
}

// declIndex tracks the declarations of each bare type name across the
// tree. T is the declaration payload the analysis keeps.
type declIndex[T any] struct {
	byKey  map[string]T
	byBare map[string][]T
}

func newDeclIndex[T any]() *declIndex[T] {
	return &declIndex[T]{byKey: map[string]T{}, byBare: map[string][]T{}}
}

// add records one declaration; a repeat under the same key is ignored, so
// same-named declarations in different files each keep an entry under the
// bare name.
func (ix *declIndex[T]) add(language *engine.Language, location, name string, decl T) {
	key := scopedKey(language, location, name)
	if _, ok := ix.byKey[key]; ok {
		return
	}
	ix.byKey[key] = decl
	bare := language.Name + "\x00" + bareName(language, name)
	ix.byBare[bare] = append(ix.byBare[bare], decl)
}

// unique returns the declaration when the bare type name has exactly one
// across the tree.
func (ix *declIndex[T]) unique(language *engine.Language, name string) (T, bool) {
	candidates := ix.byBare[language.Name+"\x00"+bareName(language, name)]
	if len(candidates) == 1 {
		return candidates[0], true
	}
	var zero T
	return zero, false
}
