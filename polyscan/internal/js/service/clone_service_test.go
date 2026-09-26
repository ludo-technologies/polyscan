package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

func TestCloneServiceDetectClones_AllFilesFailReturnsError(t *testing.T) {
	svc := NewCloneServiceWithDefaults()
	req := domain.DefaultCloneRequest()
	req.Paths = []string{"missing.js"}

	resp, err := svc.DetectClones(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when all clone inputs fail")
	}
	if resp == nil {
		t.Fatal("expected response even when analysis fails")
	}
	if resp.Success {
		t.Fatal("expected Success=false when all files fail")
	}
	if !strings.Contains(resp.Error, "missing.js") {
		t.Fatalf("expected response error to mention failing file, got: %q", resp.Error)
	}
}

func TestCloneStatisticsIncludesGroupOnlyMembers(t *testing.T) {
	svc := NewCloneServiceWithDefaults()
	a := &domain.Clone{Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	b := &domain.Clone{Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	c := &domain.Clone{Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}
	pairs := []*domain.ClonePair{{Clone1: a, Clone2: b, Similarity: 0.95, Type: domain.Type2Clone}}
	groups := []*domain.CloneGroup{{Clones: []*domain.Clone{a, b, c}, Type: domain.Type2Clone}}

	stats := svc.buildStatistics(pairs, groups, 3, 30)
	if stats.TotalClones != 3 {
		t.Fatalf("expected all pair or group members to be counted, got %d", stats.TotalClones)
	}

	clones := svc.extractUniqueClones(pairs, groups)
	if len(clones) != 3 {
		t.Fatalf("expected group-only member in clone output, got %d clones", len(clones))
	}
}

func TestCloneServiceDetectClones_PartialFailureReturnsResponseAndError(t *testing.T) {
	svc := NewCloneServiceWithDefaults()
	req := domain.DefaultCloneRequest()

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.js")
	content := []byte(`function alpha(value) {
  if (value > 10) {
    return value + 1;
  }
  return value - 1;
}
`)
	if err := os.WriteFile(validFile, content, 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	req.Paths = []string{validFile, filepath.Join(tempDir, "missing.js")}

	resp, err := svc.DetectClones(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when one input file fails")
	}
	if resp == nil {
		t.Fatal("expected response for partial failure")
	}
	if resp.Success {
		t.Fatal("expected Success=false when any input file fails")
	}
	if resp.Statistics == nil || resp.Statistics.FilesAnalyzed != 1 {
		t.Fatalf("expected one successfully analyzed file, got %+v", resp.Statistics)
	}
	if !strings.Contains(resp.Error, "missing.js") {
		t.Fatalf("expected response error to mention failing file, got: %q", resp.Error)
	}
}

// A group the detector formed through a Type-3 pair must not survive the
// default type filter as one family: it splits along the retained pairs.
func TestFilterCloneGroupsSplitsAlongRetainedPairs(t *testing.T) {
	clone := func(id int) *domain.Clone {
		return &domain.Clone{ID: id, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: id * 10, EndLine: id*10 + 5}}
	}
	a, b, c := clone(0), clone(1), clone(2)
	pairs := []*domain.ClonePair{
		{Clone1: a, Clone2: b, Similarity: 0.96, Type: domain.Type1Clone},
		{Clone1: b, Clone2: c, Similarity: 0.82, Type: domain.Type3Clone},
	}
	groups := []*domain.CloneGroup{{ID: 0, Clones: []*domain.Clone{a, b, c}, Type: domain.Type1Clone, Similarity: 0.89, Size: 3}}
	req := domain.DefaultCloneRequest()

	pairs = filterClonePairs(pairs, req)
	groups = filterCloneGroups(splitCloneGroupsByPairs(groups, pairs), req)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	group := groups[0]
	if group.Size != 2 || group.Clones[0] != a || group.Clones[1] != b {
		t.Errorf("group members = %v, want A and B only", group.Clones)
	}
	if group.Type != domain.Type1Clone || group.Similarity != 0.96 {
		t.Errorf("group = %v %.2f, want Type-1 0.96 from the retained pair", group.Type, group.Similarity)
	}
}
