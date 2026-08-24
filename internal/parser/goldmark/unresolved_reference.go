package goldmark

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

type unresolvedTextRun struct {
	start int
	end   int
}

func unresolvedReferenceUsages(source []byte, root ast.Node, context goldmarkparser.Context) []markparser.UnresolvedReferenceUsage {
	runs := unresolvedReferenceTextRuns(source, root)
	result := make([]markparser.UnresolvedReferenceUsage, 0)
	for _, run := range runs {
		result = append(result, scanUnresolvedReferenceRun(source, run, context)...)
	}
	return result
}

func unresolvedReferenceTextRuns(source []byte, root ast.Node) []unresolvedTextRun {
	result := make([]unresolvedTextRun, 0)
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		textNode, ok := node.(*ast.Text)
		if !ok || !unresolvedReferenceTextEligible(textNode) {
			return ast.WalkContinue, nil
		}
		segment := textNode.Segment
		if segment.Start < 0 || segment.Stop < segment.Start || segment.Stop > len(source) {
			return ast.WalkContinue, nil
		}
		if len(result) != 0 && result[len(result)-1].end == segment.Start {
			result[len(result)-1].end = segment.Stop
			return ast.WalkContinue, nil
		}
		result = append(result, unresolvedTextRun{start: segment.Start, end: segment.Stop})
		return ast.WalkContinue, nil
	})
	return result
}

func unresolvedReferenceTextEligible(textNode *ast.Text) bool {
	for parent := textNode.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.(type) {
		case *ast.CodeSpan, *ast.Link, *ast.Image, *ast.AutoLink, *ast.RawHTML:
			return false
		}
	}
	return true
}

func scanUnresolvedReferenceRun(source []byte, run unresolvedTextRun, context goldmarkparser.Context) []markparser.UnresolvedReferenceUsage {
	result := make([]markparser.UnresolvedReferenceUsage, 0)
	for offset := run.start; offset < run.end; offset++ {
		usage, end, ok := unresolvedReferenceAt(source, offset, run.end, context)
		if !ok {
			continue
		}
		result = append(result, usage)
		offset = end - 1
	}
	return result
}

func unresolvedReferenceAt(source []byte, anchor, limit int, context goldmarkparser.Context) (markparser.UnresolvedReferenceUsage, int, bool) {
	kind := markparser.KindInlineLink
	open := anchor
	if source[anchor] == '!' {
		if sourceByteEscaped(source, anchor) {
			return markparser.UnresolvedReferenceUsage{}, anchor, false
		}
		kind = markparser.KindImage
		open++
	}
	if open >= limit || source[open] != '[' || sourceByteEscaped(source, open) {
		return markparser.UnresolvedReferenceUsage{}, anchor, false
	}
	first, firstEnd, ok := plainReferenceLabel(source, open, limit)
	if !ok || firstEnd >= limit || source[firstEnd] != '[' {
		return markparser.UnresolvedReferenceUsage{}, anchor, false
	}
	second, secondEnd, ok := plainReferenceLabel(source, firstEnd, limit)
	if !ok {
		return markparser.UnresolvedReferenceUsage{}, anchor, false
	}

	form := markparser.LinkUsageFull
	reference := second
	if len(second) == 0 {
		form = markparser.LinkUsageCollapsed
		reference = first
	}
	if len(reference) == 0 || len(reference) > 999 || context == nil {
		return markparser.UnresolvedReferenceUsage{}, anchor, false
	}
	if _, exists := context.Reference(ReferenceLabelKey(string(reference))); exists {
		return markparser.UnresolvedReferenceUsage{}, anchor, false
	}
	return markparser.UnresolvedReferenceUsage{
		Kind:      kind,
		Form:      form,
		Anchor:    anchor,
		Reference: string(reference),
	}, secondEnd, true
}

func sourceByteEscaped(source []byte, index int) bool {
	backslashes := 0
	for position := index - 1; position >= 0 && source[position] == '\\'; position-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func plainReferenceLabel(source []byte, open, limit int) ([]byte, int, bool) {
	if open >= limit || source[open] != '[' {
		return nil, open, false
	}
	for index := open + 1; index < limit; index++ {
		switch source[index] {
		case '\r', '\n', '[', '\\':
			return nil, open, false
		case ']':
			label := source[open+1 : index]
			if len(label) != 0 && len(bytes.TrimSpace(label)) == 0 {
				return nil, open, false
			}
			return label, index + 1, true
		}
	}
	return nil, open, false
}
