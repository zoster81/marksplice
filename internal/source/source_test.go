package source

import (
	"bytes"
	"errors"
	"testing"
)

func TestNewChangeSetRejectsInvalidRange(t *testing.T) {
	t.Parallel()

	source := []byte("abc")
	_, err := NewChangeSet(source, []Patch{{
		Range:       Range{Start: 2, End: 4},
		Replacement: []byte("x"),
	}})
	if !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("NewChangeSet() error = %v, want ErrInvalidRange", err)
	}
}

func TestNewChangeSetRejectsOverlappingPatches(t *testing.T) {
	t.Parallel()

	source := []byte("abcdef")
	_, err := NewChangeSet(source, []Patch{
		{Range: Range{Start: 1, End: 4}, Replacement: []byte("x")},
		{Range: Range{Start: 3, End: 5}, Replacement: []byte("y")},
	})
	if !errors.Is(err, ErrOverlappingPatches) {
		t.Fatalf("NewChangeSet() error = %v, want ErrOverlappingPatches", err)
	}
}

func TestNewChangeSetRejectsSameOffsetInsertions(t *testing.T) {
	t.Parallel()

	source := []byte("abc")
	_, err := NewChangeSet(source, []Patch{
		{Range: Range{Start: 1, End: 1}, Replacement: []byte("x")},
		{Range: Range{Start: 1, End: 1}, Replacement: []byte("y")},
	})
	if !errors.Is(err, ErrOverlappingPatches) {
		t.Fatalf("NewChangeSet() error = %v, want ErrOverlappingPatches", err)
	}
}

func TestChangeSetCopiesReplacementBytes(t *testing.T) {
	t.Parallel()

	source := []byte("abc")
	replacement := []byte("X")
	change, err := NewChangeSet(source, []Patch{{
		Range:       Range{Start: 1, End: 2},
		Replacement: replacement,
	}})
	if err != nil {
		t.Fatalf("NewChangeSet() error = %v", err)
	}

	replacement[0] = 'Y'
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, []byte("aXc")) {
		t.Fatalf("Apply() = %q, want %q", got, "aXc")
	}
}

func TestMapTaskMarkerAcceptsGFMStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		marker  string
		checked bool
	}{
		{marker: "[ ]", checked: false},
		{marker: "[\t]", checked: false},
		{marker: "[x]", checked: true},
		{marker: "[X]", checked: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.marker, func(t *testing.T) {
			t.Parallel()

			source := []byte("prefix " + tt.marker + " suffix")
			mapping, err := MapTaskMarker(source, len("prefix "))
			if err != nil {
				t.Fatalf("MapTaskMarker() error = %v", err)
			}
			if mapping.Checked != tt.checked {
				t.Fatalf("Checked = %v, want %v", mapping.Checked, tt.checked)
			}
			if got := string(source[mapping.Range.Start:mapping.Range.End]); got != tt.marker {
				t.Fatalf("marker range = %q, want %q", got, tt.marker)
			}
		})
	}
}

func TestMapTaskMarkerRejectsMalformedSource(t *testing.T) {
	t.Parallel()

	for _, marker := range []string{"[-]", "[xx]", "[x", "x]"} {
		marker := marker
		t.Run(marker, func(t *testing.T) {
			t.Parallel()

			_, err := MapTaskMarker([]byte(marker), 0)
			if !errors.Is(err, ErrUnsupportedTaskMarker) {
				t.Fatalf("MapTaskMarker(%q) error = %v, want ErrUnsupportedTaskMarker", marker, err)
			}
		})
	}
}

func FuzzSinglePatchPreservesOutsideRange(f *testing.F) {
	f.Add([]byte("abcdef"), uint64(1), uint64(4), []byte("X"))
	f.Add([]byte("line1\r\nline2\r\n"), uint64(0), uint64(5), []byte("new"))

	f.Fuzz(func(t *testing.T, original []byte, startSeed, endSeed uint64, replacement []byte) {
		limit := uint64(len(original) + 1)
		start := int(startSeed % limit)
		end := int(endSeed % limit)
		if start > end {
			start, end = end, start
		}

		change, err := NewChangeSet(original, []Patch{{
			Range:       Range{Start: start, End: end},
			Replacement: replacement,
		}})
		if err != nil {
			t.Fatalf("NewChangeSet() error = %v", err)
		}
		got, err := change.Apply(original)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if !bytes.Equal(got[:start], original[:start]) {
			t.Fatal("prefix outside patch changed")
		}
		suffixStart := start + len(replacement)
		if !bytes.Equal(got[suffixStart:], original[end:]) {
			t.Fatal("suffix outside patch changed")
		}
	})
}
