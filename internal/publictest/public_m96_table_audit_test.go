package publictest

import (
	"bytes"
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM96DocumentBuilderConstructsHeaderOnlyTables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		aligned    bool
		alignments []marksplice.TableAlignment
		want       []byte
	}{
		{
			name: "default alignment",
			want: []byte("| A | B |\n| --- | --- |\n"),
		},
		{
			name:       "explicit alignment",
			aligned:    true,
			alignments: []marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignmentRight},
			want:       []byte("| A | B |\n| :--- | ---: |\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var builder marksplice.DocumentBuilder
			var err error
			if tt.aligned {
				err = builder.AppendTableWithAlignments([]string{"A", "B"}, tt.alignments)
			} else {
				err = builder.AppendTable([]string{"A", "B"})
			}
			if err != nil {
				t.Fatalf("append header-only table error = %v", err)
			}
			got, err := builder.Markdown()
			if err != nil {
				t.Fatalf("Markdown() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Markdown() = %q, want %q", got, tt.want)
			}

			doc, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(generated) error = %v", err)
			}
			tables := publicTables(t, doc)
			if len(tables) != 1 || tables[0].ColumnCount() != 2 || tables[0].BodyRowCount() != 0 {
				t.Fatalf("generated tables = %+v, want one two-column table with zero body rows", tables)
			}
			rowIDs, ok := doc.TableRowIDs(tables[0].ID())
			if !ok || len(rowIDs) != 0 {
				t.Fatalf("TableRowIDs() = %v/%v, want empty/true", rowIDs, ok)
			}
			alignments, ok := doc.TableAlignments(tables[0].ID())
			wantAlignments := tt.alignments
			if !tt.aligned {
				wantAlignments = []marksplice.TableAlignment{marksplice.TableAlignmentDefault, marksplice.TableAlignmentDefault}
			}
			if !ok || !slices.Equal(alignments, wantAlignments) {
				t.Fatalf("TableAlignments() = %v/%v, want %v/true", alignments, ok, wantAlignments)
			}
			sourceRange, ok := doc.SourceRange(tables[0].Range())
			if !ok || !bytes.Equal(sourceRange, tt.want) {
				t.Fatalf("table source = %q/%v, want %q/true", sourceRange, ok, tt.want)
			}
		})
	}
}
