package clone

import (
	"testing"
)

func TestCodeFragmentItemID(t *testing.T) {
	f := &CodeFragment{ID: 42}
	if f.ItemID() != 42 {
		t.Errorf("ItemID() = %d, want 42", f.ItemID())
	}
}

func TestCodeFragmentItemKey(t *testing.T) {
	f := &CodeFragment{
		ID:        1,
		FilePath:  "main.go",
		StartLine: 10,
		EndLine:   20,
		StartCol:  0,
		EndCol:    80,
	}
	want := "main.go|10|20|0|80"
	if got := f.ItemKey(); got != want {
		t.Errorf("ItemKey() = %q, want %q", got, want)
	}
}

func TestCodeFragmentImplementsGroupableItem(t *testing.T) {
	var _ GroupableItem = &CodeFragment{}
}
