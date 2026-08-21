package splice

import "testing"

func TestListParentAnchorAfterPatchesUsesSourceOwnedByte(t *testing.T) {
	t.Parallel()

	item := Node{ListHasParent: true, ListParentAnchor: 10}
	tests := []struct {
		name    string
		patches []patchTransform
		want    int
		ok      bool
	}{
		{
			name:    "insertion exactly before parent shifts anchor right",
			patches: []patchTransform{{Range: Range{Start: 10, End: 10}, ReplacementLength: 5}},
			want:    15,
			ok:      true,
		},
		{
			name:    "replacement later in parent line keeps anchor",
			patches: []patchTransform{{Range: Range{Start: 14, End: 18}, ReplacementLength: 8}},
			want:    10,
			ok:      true,
		},
		{
			name:    "deletion before parent shifts anchor left",
			patches: []patchTransform{{Range: Range{Start: 2, End: 6}}},
			want:    6,
			ok:      true,
		},
		{
			name:    "patch consuming parent anchor is invalid",
			patches: []patchTransform{{Range: Range{Start: 10, End: 11}}},
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := listParentAnchorAfterPatches(item, tt.patches)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("listParentAnchorAfterPatches() = (%d,%v), want (%d,%v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
