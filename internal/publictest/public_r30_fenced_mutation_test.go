package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestR30PrepareSetFencedBlockInfoPreservesContainerTrivia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  []byte
		info []byte
		want []byte
	}{
		{
			name: "replace longer info",
			src:  []byte("  ~~~~  go old  \nbody\n ~~~~~~   \n"),
			info: []byte("typescript module"),
			want: []byte("  ~~~~  typescript module  \nbody\n ~~~~~~   \n"),
		},
		{
			name: "clear info CRLF",
			src:  []byte("```  go extra  \r\nbody\r\n```\r\n"),
			info: nil,
			want: []byte("```    \r\nbody\r\n```\r\n"),
		},
		{
			name: "set absent info",
			src:  []byte("```   \nbody\n```\n"),
			info: []byte("json"),
			want: []byte("```   json\nbody\n```\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			blocks := doc.FencedBlocks()
			if len(blocks) != 1 {
				t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
			}
			before := blocks[0]
			change, err := doc.PrepareSetFencedBlockInfo(before.ID(), tt.info)
			if err != nil {
				t.Fatalf("PrepareSetFencedBlockInfo() error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			reparsed, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
			after := reparsed.FencedBlocks()
			if len(after) != 1 {
				t.Fatalf("applied len(FencedBlocks()) = %d, want 1", len(after))
			}
			if after[0].FenceChar() != before.FenceChar() || after[0].OpeningFenceLength() != before.OpeningFenceLength() || after[0].OpeningIndent() != before.OpeningIndent() || after[0].Closed() != before.Closed() {
				t.Fatalf("fence shape changed: before=%+v after=%+v", before, after[0])
			}
		})
	}
}

func TestR30PrepareReplaceFencedCodePopulatesProvenEmptyClosedBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  []byte
		body []byte
		want []byte
	}{
		{
			name: "LF",
			src:  []byte("```math\n```\n"),
			body: []byte("x + y"),
			want: []byte("```math\nx + y\n```\n"),
		},
		{
			name: "CRLF multiline",
			src:  []byte("  ~~~~  text  \r\n ~~~~~\r\n"),
			body: []byte("first\r\nsecond"),
			want: []byte("  ~~~~  text  \r\nfirst\r\nsecond\r\n ~~~~~\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			blocks := doc.FencedBlocks()
			if len(blocks) != 1 {
				t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
			}
			if _, ok := doc.FencedCode(blocks[0].ID()); ok {
				t.Fatal("empty block unexpectedly has legacy FencedCode edit authority")
			}
			change, err := doc.PrepareReplaceFencedCode(blocks[0].ID(), tt.body)
			if err != nil {
				t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			reparsed, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
			if blocks := reparsed.FencedBlocks(); len(blocks) != 1 {
				t.Fatalf("applied len(FencedBlocks()) = %d, want 1", len(blocks))
			}
		})
	}
}

func TestR30FencedMutationsRemainSourceBoundAndFailClosed(t *testing.T) {
	t.Parallel()

	source := []byte("```go\nbody\n```\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	block := doc.FencedBlocks()[0]
	change, err := doc.PrepareSetFencedBlockInfo(block.ID(), []byte("rust"))
	if err != nil {
		t.Fatalf("PrepareSetFencedBlockInfo() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
	if _, err := doc.PrepareSetFencedBlockInfo(block.ID(), []byte("bad\ninfo")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareSetFencedBlockInfo(multiline) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceFencedCode(block.ID(), nil); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFencedCode(empty replacement) error = %v, want ErrInvalidReplacement", err)
	}
}
