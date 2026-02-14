package apted

// CostModel defines the interface for calculating edit operation costs.
// Language-specific cost models (e.g. PythonCostModel, JavaScriptCostModel)
// should implement this interface in their respective projects.
type CostModel interface {
	Insert(node *TreeNode) float64
	Delete(node *TreeNode) float64
	Rename(node1, node2 *TreeNode) float64
}

// DefaultCostModel implements a uniform cost model where all operations cost 1.0.
type DefaultCostModel struct{}

func NewDefaultCostModel() *DefaultCostModel {
	return &DefaultCostModel{}
}

func (c *DefaultCostModel) Insert(node *TreeNode) float64 { return 1.0 }
func (c *DefaultCostModel) Delete(node *TreeNode) float64 { return 1.0 }

func (c *DefaultCostModel) Rename(node1, node2 *TreeNode) float64 {
	if node1 == nil || node2 == nil {
		return 1.0
	}
	if node1.Label == node2.Label {
		return 0.0
	}
	return 1.0
}

// WeightedCostModel allows custom weights for different operation types.
type WeightedCostModel struct {
	InsertWeight  float64
	DeleteWeight  float64
	RenameWeight  float64
	BaseCostModel CostModel
}

func NewWeightedCostModel(insertWeight, deleteWeight, renameWeight float64, baseCostModel CostModel) *WeightedCostModel {
	return &WeightedCostModel{
		InsertWeight:  insertWeight,
		DeleteWeight:  deleteWeight,
		RenameWeight:  renameWeight,
		BaseCostModel: baseCostModel,
	}
}

func (c *WeightedCostModel) Insert(node *TreeNode) float64 {
	return c.InsertWeight * c.BaseCostModel.Insert(node)
}

func (c *WeightedCostModel) Delete(node *TreeNode) float64 {
	return c.DeleteWeight * c.BaseCostModel.Delete(node)
}

func (c *WeightedCostModel) Rename(node1, node2 *TreeNode) float64 {
	return c.RenameWeight * c.BaseCostModel.Rename(node1, node2)
}
