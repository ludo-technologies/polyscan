package apted

import "testing"

func TestDefaultCostModel_Insert(t *testing.T) {
	cm := NewDefaultCostModel()
	node := NewTreeNode(0, "Test")
	if cost := cm.Insert(node); cost != 1.0 {
		t.Errorf("Insert cost = %f, want 1.0", cost)
	}
	if cost := cm.Insert(nil); cost != 1.0 {
		t.Errorf("Insert nil cost = %f, want 1.0", cost)
	}
}

func TestDefaultCostModel_Delete(t *testing.T) {
	cm := NewDefaultCostModel()
	node := NewTreeNode(0, "Test")
	if cost := cm.Delete(node); cost != 1.0 {
		t.Errorf("Delete cost = %f, want 1.0", cost)
	}
}

func TestDefaultCostModel_Rename(t *testing.T) {
	cm := NewDefaultCostModel()

	tests := []struct {
		name     string
		n1, n2   *TreeNode
		expected float64
	}{
		{"identical labels", NewTreeNode(0, "A"), NewTreeNode(1, "A"), 0.0},
		{"different labels", NewTreeNode(0, "A"), NewTreeNode(1, "B"), 1.0},
		{"nil node1", nil, NewTreeNode(1, "A"), 1.0},
		{"nil node2", NewTreeNode(0, "A"), nil, 1.0},
		{"both nil", nil, nil, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if cost := cm.Rename(tt.n1, tt.n2); cost != tt.expected {
				t.Errorf("Rename cost = %f, want %f", cost, tt.expected)
			}
		})
	}
}

func TestWeightedCostModel(t *testing.T) {
	base := NewDefaultCostModel()
	wm := NewWeightedCostModel(2.0, 3.0, 0.5, base)

	node := NewTreeNode(0, "A")

	if cost := wm.Insert(node); cost != 2.0 {
		t.Errorf("Weighted insert = %f, want 2.0", cost)
	}
	if cost := wm.Delete(node); cost != 3.0 {
		t.Errorf("Weighted delete = %f, want 3.0", cost)
	}

	n1 := NewTreeNode(0, "A")
	n2 := NewTreeNode(1, "B")
	// base rename = 1.0, weight = 0.5 -> 0.5
	if cost := wm.Rename(n1, n2); cost != 0.5 {
		t.Errorf("Weighted rename = %f, want 0.5", cost)
	}

	// Same labels: base rename = 0.0, weight * 0.0 = 0.0
	n3 := NewTreeNode(2, "A")
	if cost := wm.Rename(n1, n3); cost != 0.0 {
		t.Errorf("Weighted rename same labels = %f, want 0.0", cost)
	}
}

func TestCostModelInterface(t *testing.T) {
	// Verify both models implement the CostModel interface
	var _ CostModel = NewDefaultCostModel()
	var _ CostModel = NewWeightedCostModel(1, 1, 1, NewDefaultCostModel())
}
