// Package differential provides the parser-backend parity harness used while
// Marksplice replaces its temporary semantic parser.
package differential

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
)

// Harness compares one oracle backend with one candidate backend without exposing
// either implementation through Marksplice public APIs.
type Harness struct {
	Oracle    parser.Backend
	Candidate parser.Backend
}

// CompareDocument requires both backends to agree on parse success and every
// parser-independent observation. Nil and empty observation slices are equivalent.
func (h Harness) CompareDocument(source []byte) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource := bytes.Clone(source)
	candidateSource := bytes.Clone(source)
	oracle, oracleErr := h.Oracle.ParseDocument(oracleSource)
	candidate, candidateErr := h.Candidate.ParseDocument(candidateSource)
	if err := compareSourceMutation("document parse", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if err := compareErrorOutcome("document parse", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr != nil {
		return nil
	}
	return compareDocumentObservations(oracle, candidate)
}

// CompareNestedBlockquoteBlocks compares construction-only nested blockquote proof.
func (h Harness) CompareNestedBlockquoteBlocks(source []byte, outer parser.Range, innerSource []byte, depth int) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource, candidateSource := bytes.Clone(source), bytes.Clone(source)
	oracleInner, candidateInner := bytes.Clone(innerSource), bytes.Clone(innerSource)
	oracleErr := h.Oracle.ValidateNestedBlockquoteBlocks(oracleSource, outer, oracleInner, depth)
	candidateErr := h.Candidate.ValidateNestedBlockquoteBlocks(candidateSource, outer, candidateInner, depth)
	if err := compareSourceMutation("nested blockquote blocks", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if !bytes.Equal(oracleInner, innerSource) || !bytes.Equal(candidateInner, innerSource) {
		return fmt.Errorf("parser differential: nested blockquote blocks backend mutated inner source")
	}
	return compareErrorOutcome("nested blockquote blocks", oracleErr, candidateErr)
}

// CompareNestedBlockquoteParagraph compares construction-only nested paragraph proof.
func (h Harness) CompareNestedBlockquoteParagraph(source []byte, outer parser.Range, contentLines []parser.Range, depth int) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource, candidateSource := bytes.Clone(source), bytes.Clone(source)
	oracleLines, candidateLines := slices.Clone(contentLines), slices.Clone(contentLines)
	oracleErr := h.Oracle.ValidateNestedBlockquoteParagraph(oracleSource, outer, oracleLines, depth)
	candidateErr := h.Candidate.ValidateNestedBlockquoteParagraph(candidateSource, outer, candidateLines, depth)
	if err := compareSourceMutation("nested blockquote paragraph", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if !slices.Equal(oracleLines, contentLines) || !slices.Equal(candidateLines, contentLines) {
		return fmt.Errorf("parser differential: nested blockquote paragraph backend mutated content ranges")
	}
	return compareErrorOutcome("nested blockquote paragraph", oracleErr, candidateErr)
}

// CompareConstructionInlineHierarchy compares typed-inline hierarchy proof outcomes.
func (h Harness) CompareConstructionInlineHierarchy(source []byte, expected []parser.ConstructionInlineExpectation, references []parser.ConstructionReferenceInlineExpectation) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource, candidateSource := bytes.Clone(source), bytes.Clone(source)
	oracleExpected, candidateExpected := slices.Clone(expected), slices.Clone(expected)
	oracleReferences, candidateReferences := slices.Clone(references), slices.Clone(references)
	oracleErr := h.Oracle.ValidateConstructionInlineHierarchy(oracleSource, oracleExpected, oracleReferences)
	candidateErr := h.Candidate.ValidateConstructionInlineHierarchy(candidateSource, candidateExpected, candidateReferences)
	if err := compareSourceMutation("construction inline hierarchy", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if !slices.Equal(oracleExpected, expected) || !slices.Equal(candidateExpected, expected) ||
		!slices.Equal(oracleReferences, references) || !slices.Equal(candidateReferences, references) {
		return fmt.Errorf("parser differential: construction inline hierarchy backend mutated expectations")
	}
	return compareErrorOutcome("construction inline hierarchy", oracleErr, candidateErr)
}

// CompareConstructionLinkImages compares direct link/image proof outcomes.
func (h Harness) CompareConstructionLinkImages(source []byte, expected []parser.ConstructionLinkImageExpectation) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource, candidateSource := bytes.Clone(source), bytes.Clone(source)
	oracleExpected, candidateExpected := slices.Clone(expected), slices.Clone(expected)
	oracleErr := h.Oracle.ValidateConstructionLinkImages(oracleSource, oracleExpected)
	candidateErr := h.Candidate.ValidateConstructionLinkImages(candidateSource, candidateExpected)
	if err := compareSourceMutation("construction link/image", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if !slices.Equal(oracleExpected, expected) || !slices.Equal(candidateExpected, expected) {
		return fmt.Errorf("parser differential: construction link/image backend mutated expectations")
	}
	return compareErrorOutcome("construction link/image", oracleErr, candidateErr)
}

// CompareConstructionReferenceInlines compares reference-link/image proof outcomes.
func (h Harness) CompareConstructionReferenceInlines(source []byte, expected []parser.ConstructionReferenceInlineExpectation) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleSource, candidateSource := bytes.Clone(source), bytes.Clone(source)
	oracleExpected, candidateExpected := slices.Clone(expected), slices.Clone(expected)
	oracleErr := h.Oracle.ValidateConstructionReferenceInlines(oracleSource, oracleExpected)
	candidateErr := h.Candidate.ValidateConstructionReferenceInlines(candidateSource, candidateExpected)
	if err := compareSourceMutation("construction reference inline", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if !slices.Equal(oracleExpected, expected) || !slices.Equal(candidateExpected, expected) {
		return fmt.Errorf("parser differential: construction reference inline backend mutated expectations")
	}
	return compareErrorOutcome("construction reference inline", oracleErr, candidateErr)
}

// CompareConstructionReferenceResolution compares construction reference resolution.
func (h Harness) CompareConstructionReferenceResolution(label string, definitions []parser.ConstructionReferenceDefinition) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracleDefinitions, candidateDefinitions := slices.Clone(definitions), slices.Clone(definitions)
	oracle, oracleErr := h.Oracle.ResolveConstructionReference(label, oracleDefinitions)
	candidate, candidateErr := h.Candidate.ResolveConstructionReference(label, candidateDefinitions)
	if !slices.Equal(oracleDefinitions, definitions) || !slices.Equal(candidateDefinitions, definitions) {
		return fmt.Errorf("parser differential: construction reference resolution backend mutated definitions")
	}
	if err := compareErrorOutcome("construction reference resolution", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr == nil && oracle != candidate {
		return fmt.Errorf("parser differential: construction reference resolution differs: oracle=%+v candidate=%+v", oracle, candidate)
	}
	return nil
}

// CompareReferenceLabelKey compares the exact internal normalization key used for
// GFM reference labels. The representation remains internal and non-persistent.
func (h Harness) CompareReferenceLabelKey(label string) error {
	if err := h.validate(); err != nil {
		return err
	}
	oracle := h.Oracle.ReferenceLabelKey(label)
	candidate := h.Candidate.ReferenceLabelKey(label)
	if oracle != candidate {
		return fmt.Errorf("parser differential: reference label key differs: oracle=%q candidate=%q", oracle, candidate)
	}
	return nil
}

func (h Harness) validate() error {
	if h.Oracle == nil || h.Candidate == nil {
		return fmt.Errorf("parser differential: oracle and candidate backends are required")
	}
	return nil
}

func compareSourceMutation(operation string, source, oracleSource, candidateSource []byte) error {
	if !bytes.Equal(source, oracleSource) || !bytes.Equal(source, candidateSource) {
		return fmt.Errorf("parser differential: %s backend mutated source", operation)
	}
	return nil
}

func compareErrorOutcome(operation string, oracleErr, candidateErr error) error {
	if (oracleErr == nil) == (candidateErr == nil) {
		return nil
	}
	return fmt.Errorf("parser differential: %s outcome differs: oracle=%v candidate=%v", operation, oracleErr, candidateErr)
}

func compareDocumentObservations(oracle, candidate parser.DocumentObservations) error {
	if len(oracle.Nodes) != len(candidate.Nodes) {
		return fmt.Errorf("parser differential: DocumentObservations.Nodes length differs: oracle=%d candidate=%d", len(oracle.Nodes), len(candidate.Nodes))
	}
	for index := range oracle.Nodes {
		if !sameNode(oracle.Nodes[index], candidate.Nodes[index]) {
			return fmt.Errorf("parser differential: DocumentObservations.Nodes[%d] differs", index)
		}
	}
	if !slices.Equal(oracle.LinkUsages, candidate.LinkUsages) {
		return fmt.Errorf("parser differential: DocumentObservations.LinkUsages differs")
	}
	if !slices.Equal(oracle.UnresolvedReferenceUsages, candidate.UnresolvedReferenceUsages) {
		return fmt.Errorf("parser differential: DocumentObservations.UnresolvedReferenceUsages differs")
	}
	if len(oracle.FootnoteDefinitions) != len(candidate.FootnoteDefinitions) {
		return fmt.Errorf("parser differential: DocumentObservations.FootnoteDefinitions length differs: oracle=%d candidate=%d", len(oracle.FootnoteDefinitions), len(candidate.FootnoteDefinitions))
	}
	for index := range oracle.FootnoteDefinitions {
		if !sameFootnoteDefinition(oracle.FootnoteDefinitions[index], candidate.FootnoteDefinitions[index]) {
			return fmt.Errorf("parser differential: DocumentObservations.FootnoteDefinitions[%d] differs", index)
		}
	}
	if !slices.Equal(oracle.FootnoteReferences, candidate.FootnoteReferences) {
		return fmt.Errorf("parser differential: DocumentObservations.FootnoteReferences differs")
	}
	if !slices.Equal(oracle.MathExpressions, candidate.MathExpressions) {
		return fmt.Errorf("parser differential: DocumentObservations.MathExpressions differs")
	}
	return nil
}

func sameNode(left, right parser.Node) bool {
	leftBlockquote := left.BlockquoteSemanticRanges
	rightBlockquote := right.BlockquoteSemanticRanges
	leftFence := left.FencedCodeContentRanges
	rightFence := right.FencedCodeContentRanges
	leftAlignments := left.TableAlignments
	rightAlignments := right.TableAlignments
	left.BlockquoteSemanticRanges = nil
	right.BlockquoteSemanticRanges = nil
	left.FencedCodeContentRanges = nil
	right.FencedCodeContentRanges = nil
	left.TableAlignments = nil
	right.TableAlignments = nil
	return reflect.DeepEqual(left, right) &&
		slices.Equal(leftBlockquote, rightBlockquote) &&
		slices.Equal(leftFence, rightFence) &&
		slices.Equal(leftAlignments, rightAlignments)
}

func sameFootnoteDefinition(left, right parser.FootnoteDefinitionObservation) bool {
	return left.Anchor == right.Anchor && left.Label == right.Label && slices.Equal(left.BodyRanges, right.BodyRanges)
}
