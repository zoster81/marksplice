package splice

import "testing"

func TestSectionLookupFailsClosedOnMismatchedIndexIdentity(t *testing.T) {
	t.Parallel()

	doc := &Document{
		sections: []Section{
			{HeadingID: "root", firstChildIndex: -1, nextSiblingIndex: -1},
			{HeadingID: "other", firstChildIndex: -1, nextSiblingIndex: -1},
		},
		sectionIndex: map[NodeID]int{"root": 1},
	}
	if section, ok := doc.SectionByHeadingID("root"); ok || section != (Section{}) {
		t.Fatalf("SectionByHeadingID(mismatched index) = %+v, %v; want zero, false", section, ok)
	}
	if children, ok := doc.SectionChildHeadingIDs("root"); ok || children != nil {
		t.Fatalf("SectionChildHeadingIDs(mismatched index) = %v, %v; want nil, false", children, ok)
	}
}

func TestSectionChildHeadingIDsFailsClosedOnInconsistentChildIndex(t *testing.T) {
	t.Parallel()

	doc := &Document{
		sections: []Section{
			{HeadingID: "root", firstChildIndex: 1, childCount: 1},
			{HeadingID: "child", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -1},
			{HeadingID: "other", firstChildIndex: -1, nextSiblingIndex: -1},
		},
		sectionIndex: map[NodeID]int{"root": 0, "child": 2, "other": 2},
	}
	if children, ok := doc.SectionChildHeadingIDs("root"); ok || children != nil {
		t.Fatalf("SectionChildHeadingIDs(inconsistent child index) = %v, %v; want nil, false", children, ok)
	}
}

func TestSectionChildHeadingIDsFailsClosedOnInvalidAdjacency(t *testing.T) {
	t.Parallel()

	valid := &Document{
		sections: []Section{
			{HeadingID: "root", firstChildIndex: 1, childCount: 1},
			{HeadingID: "child", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -1},
		},
		sectionIndex: map[NodeID]int{"root": 0, "child": 1},
	}
	children, ok := valid.SectionChildHeadingIDs("root")
	if !ok || len(children) != 1 || children[0] != "child" {
		t.Fatalf("SectionChildHeadingIDs(valid) = %v, %v; want [child], true", children, ok)
	}

	tests := []struct {
		name     string
		sections []Section
	}{
		{
			name: "out of range first child",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 2, childCount: 1},
			},
		},
		{
			name: "wrong parent relation",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 1, childCount: 1},
				{HeadingID: "child", ParentHeadingID: "other", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -1},
			},
		},
		{
			name: "cycle",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 1, childCount: 1},
				{HeadingID: "child", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: 1},
			},
		},
		{
			name: "count mismatch",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 1, childCount: 2},
				{HeadingID: "child", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -1},
			},
		},
		{
			name: "backward sibling",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 2, childCount: 2},
				{HeadingID: "first", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -1},
				{HeadingID: "second", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: 1},
			},
		},
		{
			name: "invalid negative sentinel",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: 1, childCount: 1},
				{HeadingID: "child", ParentHeadingID: "root", HasParent: true, firstChildIndex: -1, nextSiblingIndex: -2},
			},
		},
		{
			name: "invalid parent sentinel",
			sections: []Section{
				{HeadingID: "root", firstChildIndex: -2, childCount: 0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := &Document{sections: tt.sections, sectionIndex: map[NodeID]int{"root": 0}}
			if children, ok := doc.SectionChildHeadingIDs("root"); ok || children != nil {
				t.Fatalf("SectionChildHeadingIDs(corrupt) = %v, %v; want nil, false", children, ok)
			}
		})
	}
}
