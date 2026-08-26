package marksplice

import (
	"fmt"
	"strings"
)

// AppendFootnoteDefinition appends one canonical top-level footnote definition.
// Body is one non-empty physical line; broader multiline parsed definitions remain
// readable but are not synthesized by this conservative construction contract.
func (b *DocumentBuilder) AppendFootnoteDefinition(label, body string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendFootnoteDefinition(constructionBlock{kind: constructionFootnoteDefinition, label: label, inlineGFM: body}, false)
}

// DeferFootnoteDefinition schedules one canonical top-level footnote definition
// after ordinary body blocks and deferred ordinary reference definitions.
func (b *DocumentBuilder) DeferFootnoteDefinition(label, body string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendFootnoteDefinition(constructionBlock{kind: constructionFootnoteDefinition, label: label, inlineGFM: body}, true)
}

// FootnoteReferenceInline returns one typed `[^label]` reference. The label must
// resolve to exactly one already-appended or explicitly deferred footnote definition
// in the destination DocumentBuilder when the value is appended.
func FootnoteReferenceInline(label string) Inline {
	return Inline{kind: inlineConstructionFootnoteReference, reference: label}
}

func (b *DocumentBuilder) appendFootnoteDefinition(block constructionBlock, deferred bool) error {
	if err := validateConstructionBlockStandalone(block); err != nil {
		return err
	}
	if footnoteDefinitionLabelCollides(block.label, b.blocks) || footnoteDefinitionLabelCollides(block.label, b.deferredFootnotes) {
		return fmt.Errorf("%w: duplicate footnote definition label", ErrInvalidConstruction)
	}
	if deferred {
		b.deferredFootnotes = append(b.deferredFootnotes, block)
		return nil
	}
	b.blocks = append(b.blocks, block)
	return nil
}

func footnoteDefinitionLabelCollides(label string, blocks []constructionBlock) bool {
	for _, block := range blocks {
		if block.kind == constructionFootnoteDefinition && block.label == label {
			return true
		}
	}
	return false
}

func constructionFootnoteLabels(blocks []constructionBlock) []string {
	result := make([]string, 0)
	for _, block := range blocks {
		if block.kind == constructionFootnoteDefinition {
			result = append(result, block.label)
		}
	}
	return result
}

func availableConstructionFootnoteLabels(prior, deferred []string) []string {
	result := make([]string, 0, len(prior)+len(deferred))
	result = append(result, prior...)
	result = append(result, deferred...)
	return result
}

type typedFootnoteReferenceExpectation struct {
	sourceRange Range
	labelRange  Range
	label       string
}

func writeTypedInlineFootnoteReference(output *strings.Builder, inline Inline, context typedInlineWriteContext) error {
	if err := validateConstructionFootnoteLabel(inline.reference); err != nil {
		return err
	}
	labels := availableConstructionFootnoteLabels(context.footnoteDefinitions, context.deferredFootnoteDefinitions)
	if !uniqueConstructionFootnoteLabel(inline.reference, labels) {
		return fmt.Errorf("footnote reference %q must resolve to exactly one available definition", inline.reference)
	}
	syntaxStart := output.Len()
	output.WriteString("[^")
	labelStart := output.Len()
	output.WriteString(inline.reference)
	labelEnd := output.Len()
	output.WriteByte(']')
	*context.footnoteExpected = append(*context.footnoteExpected, typedFootnoteReferenceExpectation{
		sourceRange: Range{Start: syntaxStart, End: output.Len()},
		labelRange:  Range{Start: labelStart, End: labelEnd},
		label:       inline.reference,
	})
	return nil
}

func validateTypedFootnoteReferences(source string, expected []typedFootnoteReferenceExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	var proof strings.Builder
	proof.WriteString(source)
	proof.WriteString("\n\n")
	seen := make(map[string]struct{}, len(expected))
	for _, want := range expected {
		if _, ok := seen[want.label]; ok {
			continue
		}
		seen[want.label] = struct{}{}
		proof.WriteString("[^")
		proof.WriteString(want.label)
		proof.WriteString("]: proof\n")
	}
	document, err := Parse([]byte(proof.String()))
	if err != nil {
		return fmt.Errorf("%w: typed footnote reference proof parse: %v", ErrInvalidConstruction, err)
	}
	references := document.FootnoteReferences()
	if len(references) != len(expected) {
		return fmt.Errorf("%w: typed footnote reference count changed", ErrInvalidConstruction)
	}
	occurrences := make(map[string]int, len(seen))
	for index, want := range expected {
		got := references[index]
		if got.Range() != want.sourceRange || got.LabelRange() != want.labelRange || got.Label() != want.label || got.Occurrence() != occurrences[want.label] {
			return fmt.Errorf("%w: typed footnote reference semantic proof changed", ErrInvalidConstruction)
		}
		if _, ok := got.DefinitionID(); !ok {
			return fmt.Errorf("%w: typed footnote reference lost its definition", ErrInvalidConstruction)
		}
		occurrences[want.label]++
	}
	return nil
}

func uniqueConstructionFootnoteLabel(label string, labels []string) bool {
	count := 0
	for _, candidate := range labels {
		if candidate == label {
			count++
		}
	}
	return count == 1
}
