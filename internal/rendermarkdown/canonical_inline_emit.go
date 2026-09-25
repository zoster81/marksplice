package rendermarkdown

import (
	"fmt"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

type canonicalInlineASTState struct {
	containerLineStart bool
	bareWWW            canonicalBareWWWContinuation
	extendedAutolink   canonicalExtendedAutolinkTail
}

type canonicalExtendedAutolinkTail struct {
	form   parser.AutoLinkForm
	value  string
	email  bool
	active bool
}

type canonicalBareWWWContinuation struct {
	continuation     parser.BareWWWContinuation
	requiredOwnerEnd int
	active           bool
	invalid          bool
}

func newCanonicalInlineASTState() canonicalInlineASTState {
	return canonicalInlineASTState{containerLineStart: true}
}

func (s *canonicalInlineASTState) observe(value []byte) {
	if len(value) == 0 {
		return
	}
	s.containerLineStart = value[len(value)-1] == '\n'
	s.bareWWW.observe(value)
	s.extendedAutolink = canonicalExtendedAutolinkTail{}
}

func (s *canonicalInlineASTState) startBareWWWContinuation(value string) bool {
	if !s.bareWWW.semanticOwnerValid() {
		return false
	}
	continuation, requiredOwnerEnd, ok := parser.BareWWWContinuationFromSemanticValue(value)
	if !ok {
		s.bareWWW = canonicalBareWWWContinuation{}
		return false
	}
	s.bareWWW = canonicalBareWWWContinuation{
		continuation:     continuation,
		requiredOwnerEnd: requiredOwnerEnd,
		active:           true,
	}
	return true
}

func (s *canonicalBareWWWContinuation) observe(value []byte) {
	if !s.active || s.invalid {
		return
	}
	for _, b := range value {
		if !s.continuation.AppendByte(b) {
			s.active = false
			s.invalid = !s.semanticOwnerMaterialized()
			return
		}
	}
}

func (s canonicalBareWWWContinuation) semanticOwnerMaterialized() bool {
	if s.requiredOwnerEnd == 0 {
		return true
	}
	end, owner := s.continuation.Owner()
	return owner && end == s.requiredOwnerEnd
}

func (s canonicalBareWWWContinuation) semanticOwnerValid() bool {
	if s.invalid {
		return false
	}
	if !s.active {
		return true
	}
	return s.semanticOwnerMaterialized()
}

func (s canonicalBareWWWContinuation) rawTextPrefixEnd(value string) int {
	if !s.active || value == "" {
		return 0
	}
	baselineOwnerEnd := s.requiredOwnerEnd
	if baselineOwnerEnd == 0 {
		return 0
	}
	candidate := s.continuation
	safeEnd := 0
	for index := 0; index < len(value); index++ {
		active := candidate.AppendByte(value[index])
		if ownerEnd, owner := candidate.Owner(); owner && ownerEnd <= baselineOwnerEnd {
			safeEnd = index + 1
		}
		if !active {
			break
		}
	}
	return safeEnd
}

type canonicalInlineReferenceLabelKeyer interface {
	ReferenceLabelKey(string) string
}

type canonicalInlineEmitContext struct {
	referenceLabels canonicalInlineReferenceLabelKeyer
}

type canonicalInlineASTEmitFrame struct {
	node         int
	nextChild    int
	state        canonicalInlineASTState
	event        parser.SemanticEvent
	contentStart int
	close        string
}

type canonicalInlineLeafEmission struct {
	value            []byte
	bareWWWValue     string
	extendedAutolink canonicalExtendedAutolinkTail
}

type canonicalInlineDelimiterSite struct {
	node   int
	open   bool
	range_ parser.Range
	marker byte
}

type canonicalInlineCandidateMetadata struct {
	delimiterSites    []canonicalInlineDelimiterSite
	owners            []parser.Range
	extendedAutolinks []native.ExtendedAutolinkOwner
	bareWWW           canonicalBareWWWContinuation
}

func (m *canonicalInlineCandidateMetadata) finish(bareWWW canonicalBareWWWContinuation) {
	if m != nil {
		m.bareWWW = bareWWW
	}
}

func (m *canonicalInlineCandidateMetadata) observeOpen(
	kind parser.SemanticKind,
	node int,
	marker byte,
	start int,
	end int,
) {
	if m == nil {
		return
	}
	marker, ok := canonicalInlineDelimiterSiteMarker(kind, marker)
	if !ok {
		return
	}
	m.delimiterSites = append(m.delimiterSites, canonicalInlineDelimiterSite{
		node: node, open: true, range_: parser.Range{Start: start, End: end}, marker: marker,
	})
}

func (m *canonicalInlineCandidateMetadata) observeClose(
	kind parser.SemanticKind,
	node int,
	marker byte,
	start int,
	end int,
) {
	if m == nil {
		return
	}
	if marker, ok := canonicalInlineDelimiterSiteMarker(kind, marker); ok {
		m.delimiterSites = append(m.delimiterSites, canonicalInlineDelimiterSite{
			node: node, open: false, range_: parser.Range{Start: start, End: end}, marker: marker,
		})
		return
	}
	if kind == parser.SemanticLink || kind == parser.SemanticImage {
		m.owners = appendCanonicalInlineOwnerRange(m.owners, parser.Range{Start: start, End: end})
	}
}

func (m *canonicalInlineCandidateMetadata) observeLeaf(
	event parser.SemanticEvent,
	start int,
	end int,
) {
	if m == nil {
		return
	}
	if canonicalInlineDelimiterOwnerLeaf(event.Kind) {
		m.owners = appendCanonicalInlineOwnerRange(m.owners, parser.Range{Start: start, End: end})
	}
	switch event.AutoLinkForm {
	case parser.AutoLinkExtendedURL, parser.AutoLinkExtendedEmail, parser.AutoLinkExtendedProtocol:
		m.extendedAutolinks = append(m.extendedAutolinks, native.ExtendedAutolinkOwner{
			Range: parser.Range{Start: start, End: end},
			Form:  event.AutoLinkForm,
			Value: event.Value,
			Email: event.AutoLinkEmail,
		})
	}
}

func canonicalInlineDelimiterSiteMarker(kind parser.SemanticKind, marker byte) (byte, bool) {
	switch kind {
	case parser.SemanticEmphasis, parser.SemanticStrong:
		return marker, true
	case parser.SemanticStrikethrough:
		return '~', true
	default:
		return 0, false
	}
}

func appendCanonicalInlineOwnerRange(owners []parser.Range, next parser.Range) []parser.Range {
	if next.Start >= next.End {
		return owners
	}
	if len(owners) == 0 || owners[len(owners)-1].End < next.Start {
		return append(owners, next)
	}
	if next.End > owners[len(owners)-1].End {
		owners[len(owners)-1].End = next.End
	}
	return owners
}

func canonicalInlineDelimiterOwnerLeaf(kind parser.SemanticKind) bool {
	switch kind {
	case parser.SemanticCodeSpan,
		parser.SemanticFootnoteReference,
		parser.SemanticAutoLink,
		parser.SemanticRawHTML,
		parser.SemanticMath:
		return true
	default:
		return false
	}
}

func emitCanonicalInlineAST(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	plan canonicalInlinePlan,
	context canonicalInlineEmitContext,
	tableCell bool,
) ([]byte, error) {
	return emitCanonicalInlineASTWithMetadata(ast, normalization, plan, context, tableCell, nil)
}

func canonicalInlineEmissionInputValid(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	plan canonicalInlinePlan,
) bool {
	return ast.root >= 0 && ast.root < len(ast.nodes) &&
		len(normalization.nodes) == len(ast.nodes) &&
		len(plan.nodes) == len(ast.nodes)
}

func emitCanonicalInlineASTWithMetadata(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	plan canonicalInlinePlan,
	context canonicalInlineEmitContext,
	tableCell bool,
	metadata *canonicalInlineCandidateMetadata,
) ([]byte, error) {
	if !canonicalInlineEmissionInputValid(ast, normalization, plan) {
		return nil, ErrInvalidInput
	}
	output := make([]byte, 0)
	stack := []canonicalInlineASTEmitFrame{{
		node:      ast.root,
		nextChild: ast.nodes[ast.root].firstChild,
		state:     newCanonicalInlineASTState(),
	}}
	for len(stack) != 0 {
		frame := &stack[len(stack)-1]
		if frame.nextChild == noCanonicalInlineASTNode {
			if frame.node == ast.root {
				if !frame.state.bareWWW.semanticOwnerValid() {
					return nil, fmt.Errorf("%w: bare www semantic owner changed", ErrInvalidInput)
				}
				metadata.finish(frame.state.bareWWW)
				stack = stack[:len(stack)-1]
				continue
			}
			closed := *frame
			stack = stack[:len(stack)-1]
			parent := &stack[len(stack)-1]
			closeStart := len(output)
			close, err := appendCanonicalInlineClose(&output, closed, context)
			if err != nil {
				return nil, err
			}
			metadata.observeClose(
				closed.event.Kind,
				closed.node,
				plan.nodes[closed.node].marker,
				closeStart,
				len(output),
			)
			parent.state.bareWWW = closed.state.bareWWW
			parent.state.observe(close)
			continue
		}
		childIndex, child, event, err := nextCanonicalInlineChild(ast, frame)
		if err != nil {
			return nil, err
		}
		if event.Phase == parser.SemanticLeaf {
			leafStart := len(output)
			if err := appendCanonicalInlineLeaf(
				&output,
				frame,
				event,
				tableCell,
				normalization.nodes[childIndex],
				plan.nodes[childIndex],
			); err != nil {
				return nil, err
			}
			metadata.observeLeaf(event, leafStart, len(output))
			continue
		}
		if event.Phase != parser.SemanticEnter {
			return nil, fmt.Errorf("%w: inline AST node phase %d", ErrInvalidInput, event.Phase)
		}
		if plan.nodes[childIndex].payload != canonicalInlinePreferredPayload {
			return nil, fmt.Errorf("%w: payload choice on inline wrapper kind %d", ErrInvalidInput, event.Kind)
		}
		open, close, err := canonicalInlineWrapperDelimiters(
			event.Kind,
			plan.nodes[childIndex].marker,
			plan.nodes[childIndex].strikethroughWidth,
		)
		if err != nil {
			return nil, err
		}
		openStart := len(output)
		output = append(output, open...)
		metadata.observeOpen(
			event.Kind,
			childIndex,
			plan.nodes[childIndex].marker,
			openStart,
			len(output),
		)
		frame.state.observe([]byte(open))
		childState := newCanonicalInlineASTState()
		childState.bareWWW = frame.state.bareWWW
		stack = append(stack, canonicalInlineASTEmitFrame{
			node:         childIndex,
			nextChild:    child.firstChild,
			state:        childState,
			event:        event,
			contentStart: len(output),
			close:        close,
		})
	}
	return output, nil
}

func appendCanonicalInlineClose(
	output *[]byte,
	frame canonicalInlineASTEmitFrame,
	context canonicalInlineEmitContext,
) ([]byte, error) {
	switch frame.event.Kind {
	case parser.SemanticLink, parser.SemanticImage:
		return appendCanonicalInlineLinkOrImageClose(output, frame, context)
	default:
		close := []byte(frame.close)
		*output = append(*output, close...)
		return close, nil
	}
}

func appendCanonicalInlineLinkOrImageClose(
	output *[]byte,
	frame canonicalInlineASTEmitFrame,
	context canonicalInlineEmitContext,
) ([]byte, error) {
	if frame.contentStart < 0 || frame.contentStart > len(*output) {
		return nil, fmt.Errorf("%w: inline wrapper content start %d", ErrInvalidInput, frame.contentStart)
	}
	event := frame.event
	renderedLabel := string((*output)[frame.contentStart:])
	if event.Label != "" && !referenceLabelUsesFootnoteSyntax(event.Label) {
		if context.referenceLabels == nil {
			return nil, fmt.Errorf("%w: missing reference-label authority", ErrInvalidInput)
		}
		labelKey := context.referenceLabels.ReferenceLabelKey(decodeEscapedASCIIPunctuation(event.Label))
		renderedKey := context.referenceLabels.ReferenceLabelKey(decodeEscapedASCIIPunctuation(renderedLabel))
		if labelKey == "" {
			return nil, fmt.Errorf("%w: empty normalized reference label", ErrInvalidInput)
		}
		if event.Kind == parser.SemanticImage && labelKey == renderedKey {
			*output = (*output)[:frame.contentStart]
			*output = append(*output, event.Label...)
			close := []byte("]")
			*output = append(*output, close...)
			return close, nil
		}
		close := []byte("][" + event.Label + "]")
		*output = append(*output, close...)
		return close, nil
	}
	var close strings.Builder
	close.WriteString("](")
	close.WriteString(renderDestination(event.Destination))
	if event.HasTitle {
		close.WriteString(" \"")
		close.WriteString(escapeDoubleQuotedTitle(event.Title))
		close.WriteByte('"')
	}
	close.WriteByte(')')
	value := []byte(close.String())
	*output = append(*output, value...)
	return value, nil
}

func nextCanonicalInlineChild(
	ast canonicalInlineAST,
	frame *canonicalInlineASTEmitFrame,
) (int, canonicalInlineASTNode, parser.SemanticEvent, error) {
	childIndex := frame.nextChild
	if childIndex < 0 || childIndex >= len(ast.nodes) {
		return 0, canonicalInlineASTNode{}, parser.SemanticEvent{},
			fmt.Errorf("%w: inline AST child index %d", ErrInvalidInput, childIndex)
	}
	child := ast.nodes[childIndex]
	frame.nextChild = child.nextSibling
	if child.eventIndex < 0 || child.eventIndex >= len(ast.events) {
		return 0, canonicalInlineASTNode{}, parser.SemanticEvent{},
			fmt.Errorf("%w: inline AST event index %d", ErrInvalidInput, child.eventIndex)
	}
	return childIndex, child, ast.events[child.eventIndex], nil
}

func appendCanonicalInlineLeaf(
	output *[]byte,
	frame *canonicalInlineASTEmitFrame,
	event parser.SemanticEvent,
	tableCell bool,
	normalization canonicalInlineNodeNormalization,
	plan canonicalInlineNodePlan,
) error {
	afterPhysicalInlineBreak := len(*output) != 0 && (*output)[len(*output)-1] == '\n'
	emission, err := renderCanonicalInlineLeaf(
		event,
		frame.state,
		afterPhysicalInlineBreak,
		tableCell,
		normalization,
		plan,
	)
	if err != nil {
		return err
	}
	*output = append(*output, emission.value...)
	frame.state.observe(emission.value)
	if emission.bareWWWValue != "" && !frame.state.startBareWWWContinuation(emission.bareWWWValue) {
		return fmt.Errorf("%w: invalid bare www autolink continuation value", ErrInvalidInput)
	}
	if emission.extendedAutolink.active {
		frame.state.extendedAutolink = emission.extendedAutolink
	}
	return nil
}

func renderCanonicalInlineLeaf(
	event parser.SemanticEvent,
	state canonicalInlineASTState,
	afterPhysicalInlineBreak bool,
	tableCell bool,
	normalization canonicalInlineNodeNormalization,
	plan canonicalInlineNodePlan,
) (canonicalInlineLeafEmission, error) {
	if event.Kind != parser.SemanticText && plan.payload != canonicalInlinePreferredPayload {
		return canonicalInlineLeafEmission{}, fmt.Errorf("%w: payload choice on inline leaf kind %d", ErrInvalidInput, event.Kind)
	}
	switch event.Kind {
	case parser.SemanticText:
		value, err := renderCanonicalInlineText(event.Value, state, normalization, plan.payload)
		return canonicalInlineLeafEmission{value: value}, err
	case parser.SemanticSoftBreak:
		return canonicalInlineLeafEmission{value: []byte("\n")}, nil
	case parser.SemanticHardBreak:
		return canonicalInlineLeafEmission{value: []byte("\\\n")}, nil
	case parser.SemanticCodeSpan:
		return canonicalInlineLeafEmission{value: []byte(renderCodeSpan(event.Value, tableCell))}, nil
	case parser.SemanticFootnoteReference:
		return canonicalInlineLeafEmission{value: []byte("[^" + event.Label + "]")}, nil
	case parser.SemanticAutoLink:
		return renderCanonicalInlineAutoLink(event)
	case parser.SemanticRawHTML:
		value := normalizeLineEndings(event.Value)
		if afterPhysicalInlineBreak {
			value = "    " + value
		}
		return canonicalInlineLeafEmission{value: []byte(value)}, nil
	case parser.SemanticMath:
		value, block, err := renderMath(event)
		if err != nil {
			return canonicalInlineLeafEmission{}, err
		}
		if block {
			return canonicalInlineLeafEmission{}, fmt.Errorf("%w: block math inside inline AST", ErrInvalidInput)
		}
		return canonicalInlineLeafEmission{value: []byte(value)}, nil
	default:
		return canonicalInlineLeafEmission{}, fmt.Errorf("%w: unsupported clean AST leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

type canonicalAutoLinkRenderSpec struct {
	explicit            bool
	email               bool
	destination         string
	preserveWWWBoundary bool
}

func renderCanonicalInlineAutoLink(event parser.SemanticEvent) (canonicalInlineLeafEmission, error) {
	spec, err := canonicalAutoLinkRenderSpecFor(event)
	if err != nil {
		return canonicalInlineLeafEmission{}, err
	}
	if event.AutoLinkEmail != spec.email || event.Destination != spec.destination {
		return canonicalInlineLeafEmission{}, fmt.Errorf(
			"%w: inconsistent autolink facts for form %d",
			ErrInvalidInput,
			event.AutoLinkForm,
		)
	}

	value := event.Value
	if spec.explicit {
		value = "<" + value + ">"
	}
	emission := canonicalInlineLeafEmission{value: []byte(value)}
	if spec.preserveWWWBoundary {
		emission.bareWWWValue = event.Value
	}
	switch event.AutoLinkForm {
	case parser.AutoLinkExtendedURL, parser.AutoLinkExtendedEmail, parser.AutoLinkExtendedProtocol:
		emission.extendedAutolink = canonicalExtendedAutolinkTail{
			form:   event.AutoLinkForm,
			value:  event.Value,
			email:  event.AutoLinkEmail,
			active: true,
		}
	}
	return emission, nil
}

func canonicalAutoLinkRenderSpecFor(event parser.SemanticEvent) (canonicalAutoLinkRenderSpec, error) {
	switch event.AutoLinkForm {
	case parser.AutoLinkExplicitURI:
		return canonicalAutoLinkRenderSpec{explicit: true, destination: event.Value}, nil
	case parser.AutoLinkExplicitEmail:
		return canonicalAutoLinkRenderSpec{explicit: true, email: true, destination: "mailto:" + event.Value}, nil
	case parser.AutoLinkExtendedWWW:
		return canonicalAutoLinkRenderSpec{destination: "http://" + event.Value, preserveWWWBoundary: true}, nil
	case parser.AutoLinkExtendedURL, parser.AutoLinkExtendedProtocol:
		return canonicalAutoLinkRenderSpec{destination: event.Value}, nil
	case parser.AutoLinkExtendedEmail:
		return canonicalAutoLinkRenderSpec{email: true, destination: "mailto:" + event.Value}, nil
	default:
		return canonicalAutoLinkRenderSpec{}, fmt.Errorf("%w: unknown autolink form %d", ErrInvalidInput, event.AutoLinkForm)
	}
}

func renderCanonicalInlineText(
	value string,
	state canonicalInlineASTState,
	normalization canonicalInlineNodeNormalization,
	choice canonicalInlinePayloadChoice,
) ([]byte, error) {
	switch choice {
	case canonicalInlinePreferredPayload:
		return []byte(renderCanonicalPreferredText(value, state)), nil
	case canonicalInlineRawTabPayload:
		if normalization.trailingAlternative != canonicalInlineRawTabBoundary ||
			!strings.HasSuffix(value, "\t") {
			return nil, fmt.Errorf("%w: unavailable raw TAB payload choice", ErrInvalidInput)
		}
		preferredPrefix := renderCanonicalPreferredText(value[:len(value)-1], state)
		result := make([]byte, 0, len(preferredPrefix)+1)
		result = append(result, preferredPrefix...)
		result = append(result, '\t')
		return result, nil
	default:
		return nil, fmt.Errorf("%w: inline payload choice %d", ErrInvalidInput, choice)
	}
}

func renderCanonicalPreferredText(value string, state canonicalInlineASTState) string {
	if state.bareWWW.active {
		prefixEnd := state.bareWWW.rawTextPrefixEnd(value)
		return value[:prefixEnd] + escapeText(value[prefixEnd:])
	}
	if state.extendedAutolink.active {
		prefixEnd := native.ExtendedAutolinkSafeTrailingTextPrefix(
			state.extendedAutolink.form,
			state.extendedAutolink.value,
			state.extendedAutolink.email,
			value,
		)
		if prefixEnd != 0 {
			return value[:prefixEnd] + escapeText(value[prefixEnd:])
		}
	}
	if state.containerLineStart {
		return escapeLeadingTextSpaces(value)
	}
	return escapeText(value)
}

func canonicalInlineWrapperDelimiters(
	kind parser.SemanticKind,
	marker byte,
	strikethroughWidth uint8,
) (string, string, error) {
	width := 0
	switch kind {
	case parser.SemanticEmphasis:
		width = 1
	case parser.SemanticStrong:
		width = 2
	case parser.SemanticStrikethrough:
		width = int(strikethroughWidth)
		if width != 1 && width != 2 {
			return "", "", fmt.Errorf("%w: invalid strikethrough width %d", ErrInvalidInput, width)
		}
		delimiter := strings.Repeat("~", width)
		return delimiter, delimiter, nil
	case parser.SemanticLink:
		return "[", "", nil
	case parser.SemanticImage:
		return "![", "", nil
	default:
		return "", "", fmt.Errorf("%w: unsupported clean AST wrapper kind %d", ErrInvalidInput, kind)
	}
	if marker != '*' && marker != '_' {
		return "", "", fmt.Errorf("%w: missing clean AST marker for kind %d", ErrInvalidInput, kind)
	}
	delimiter := strings.Repeat(string(marker), width)
	return delimiter, delimiter, nil
}
