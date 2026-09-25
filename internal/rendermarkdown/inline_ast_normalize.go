package rendermarkdown

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineBoundaryEdge struct {
	marker         byte
	runLength      int
	whitespace     bool
	punctuation    bool
	preservableTab bool
}

type inlineBoundaryFacts struct {
	empty bool
	left  inlineBoundaryEdge
	right inlineBoundaryEdge
}

type inlineNormalizedPayload struct {
	value            string
	boundary         inlineBoundaryFacts
	present          bool
	payloadReady     bool
	boundaryReady    bool
	bareAutoLinkTail bool
}

type inlineChildSequence struct {
	left         inlineBoundaryEdge
	right        inlineBoundaryEdge
	lastResolved int
	blockedAt    int
	hasBoundary  bool
	complete     bool
}

type inlineASTNormalization struct {
	nodes     []inlineNormalizedPayload
	sequences []inlineChildSequence
}

func analyzeInlineBoundary(value []byte) inlineBoundaryFacts {
	if len(value) == 0 {
		whitespace := inlineBoundaryEdge{whitespace: true}
		return inlineBoundaryFacts{
			empty: true,
			left:  whitespace,
			right: whitespace,
		}
	}

	segment := parser.Range{Start: 0, End: len(value)}
	return inlineBoundaryFacts{
		left:  analyzeInlineLeftBoundary(value),
		right: analyzeInlineRightBoundary(value, segment),
	}
}

func analyzeInlineLeftBoundary(value []byte) inlineBoundaryEdge {
	marker, runLength := leadingInlineMarkerRun(value)
	position := 0
	if runLength != 0 {
		position = runLength
	}
	whitespace, punctuation := parser.DelimiterFollowingClass(value, 0, position, len(value))
	return inlineBoundaryEdge{
		marker:      marker,
		runLength:   runLength,
		whitespace:  whitespace,
		punctuation: punctuation,
	}
}

func analyzeInlineRightBoundary(value []byte, segment parser.Range) inlineBoundaryEdge {
	marker, runLength := trailingInlineMarkerRun(value)
	position := len(value)
	if runLength != 0 {
		position -= runLength
	}
	whitespace, punctuation := parser.DelimiterPrecedingClass(value, segment, position)
	return inlineBoundaryEdge{
		marker:      marker,
		runLength:   runLength,
		whitespace:  whitespace,
		punctuation: punctuation,
	}
}

func leadingInlineMarkerRun(value []byte) (byte, int) {
	if len(value) == 0 || (value[0] != '*' && value[0] != '_') {
		return 0, 0
	}
	marker := value[0]
	index := 1
	for index < len(value) && value[index] == marker {
		index++
	}
	return marker, index
}

func trailingInlineMarkerRun(value []byte) (byte, int) {
	if len(value) == 0 {
		return 0, 0
	}
	end := len(value) - 1
	marker := value[end]
	if (marker != '*' && marker != '_') || sourceByteEscapedAt(value, end) {
		return 0, 0
	}
	start := end
	for start > 0 && value[start-1] == marker && !sourceByteEscapedAt(value, start-1) {
		start--
	}
	return marker, end - start + 1
}

func (r *renderer) normalizeInlineAST(ast inlineAST, tableCell bool) (inlineASTNormalization, error) {
	if ast.root < 0 || ast.root >= len(ast.nodes) {
		return inlineASTNormalization{}, ErrInvalidInput
	}
	normalization := inlineASTNormalization{
		nodes:     make([]inlineNormalizedPayload, len(ast.nodes)),
		sequences: make([]inlineChildSequence, len(ast.nodes)),
	}
	for index := range normalization.sequences {
		normalization.sequences[index] = inlineChildSequence{
			lastResolved: noInlineASTNode,
			blockedAt:    noInlineASTNode,
			complete:     true,
		}
	}
	for astNode := ast.root + 1; astNode < len(ast.nodes); astNode++ {
		node := ast.nodes[astNode]
		if node.eventIndex < 0 || node.eventIndex >= len(ast.events) {
			return inlineASTNormalization{}, fmt.Errorf("%w: inline AST event index %d", ErrInvalidInput, node.eventIndex)
		}
		event := ast.events[node.eventIndex]
		switch event.Phase {
		case parser.SemanticLeaf:
			payload, err := r.normalizeInlineLeaf(event, tableCell)
			if err != nil {
				return inlineASTNormalization{}, err
			}
			normalization.nodes[astNode] = payload
		case parser.SemanticEnter:
			payload, err := normalizeInlineWrapperBoundary(event.Kind)
			if err != nil {
				return inlineASTNormalization{}, err
			}
			normalization.nodes[astNode] = payload
		default:
			return inlineASTNormalization{}, fmt.Errorf("%w: inline AST node phase %d", ErrInvalidInput, event.Phase)
		}
	}
	r.resolveInlineChildSequences(ast, &normalization)
	r.markInlinePreservableTabs(ast, &normalization)
	return normalization, nil
}

func (r *renderer) markInlinePreservableTabs(ast inlineAST, normalization *inlineASTNormalization) {
	if normalization == nil || len(normalization.nodes) != len(ast.nodes) {
		return
	}
	for astNode := ast.root + 1; astNode < len(ast.nodes); astNode++ {
		next := ast.nodes[astNode].nextSibling
		if next == noInlineASTNode || !inlineASTNodeHasOneDirectEmphasisChild(ast, next) {
			continue
		}
		event := ast.events[ast.nodes[astNode].eventIndex]
		nextEvent := ast.events[ast.nodes[next].eventIndex]
		if event.Kind != parser.SemanticText || event.Value == "" || event.Value[len(event.Value)-1] != '\t' ||
			event.Range.End != nextEvent.Range.Start || event.Range.End > len(r.source) ||
			event.Range.Start >= event.Range.End || r.source[event.Range.End-1] != '\t' {
			continue
		}
		payload := &normalization.nodes[astNode]
		if payload.boundaryReady && !payload.boundary.empty {
			payload.boundary.right.preservableTab = true
		}
	}
}

func inlineASTNodeHasOneDirectEmphasisChild(ast inlineAST, astNode int) bool {
	if astNode < 0 || astNode >= len(ast.nodes) {
		return false
	}
	count := 0
	for child := ast.nodes[astNode].firstChild; child != noInlineASTNode; child = ast.nodes[child].nextSibling {
		event := ast.events[ast.nodes[child].eventIndex]
		if event.Phase == parser.SemanticEnter &&
			(event.Kind == parser.SemanticEmphasis || event.Kind == parser.SemanticStrong) {
			count++
			if count > 1 {
				return false
			}
		}
	}
	return count == 1
}

func (r *renderer) resolveInlineChildSequences(ast inlineAST, normalization *inlineASTNormalization) {
	if normalization == nil || len(normalization.nodes) != len(ast.nodes) ||
		len(normalization.sequences) != len(ast.nodes) {
		return
	}
	for parent := ast.root; parent < len(ast.nodes); parent++ {
		r.resolveInlineChildSequence(ast, normalization, parent)
	}
}

func (r *renderer) resolveInlineChildSequence(ast inlineAST, normalization *inlineASTNormalization, parent int) {
	if parent < 0 || parent >= len(ast.nodes) {
		return
	}
	state := inlineSequenceState{lineStart: true}
	previous := parser.SemanticEvent{}
	for child := ast.nodes[parent].firstChild; child != noInlineASTNode; child = ast.nodes[child].nextSibling {
		event := ast.events[ast.nodes[child].eventIndex]
		payload := normalization.nodes[child]
		if !payload.payloadReady && event.Phase == parser.SemanticLeaf {
			resolved, ready := r.resolveContextualInlineLeaf(event, previous, state)
			if ready {
				normalization.nodes[child] = resolved
				payload = resolved
			}
		}
		if !state.observe(payload) {
			normalization.sequences[parent].complete = false
			normalization.sequences[parent].blockedAt = child
			return
		}
		sequence := &normalization.sequences[parent]
		if payload.boundaryReady && !payload.boundary.empty {
			if !sequence.hasBoundary {
				sequence.left = payload.boundary.left
				sequence.hasBoundary = true
			}
			sequence.right = payload.boundary.right
		}
		sequence.lastResolved = child
		previous = event
	}
}

func normalizeInlineWrapperBoundary(kind parser.SemanticKind) (inlineNormalizedPayload, error) {
	payload := inlineNormalizedPayload{present: true}
	switch kind {
	case parser.SemanticEmphasis, parser.SemanticStrong:
		return payload, nil
	case parser.SemanticStrikethrough:
		payload.boundary = analyzeInlineBoundary([]byte("~~"))
	case parser.SemanticLink:
		payload.boundary = analyzeInlineBoundary([]byte("[]"))
	case parser.SemanticImage:
		payload.boundary = analyzeInlineBoundary([]byte("![]"))
	default:
		return inlineNormalizedPayload{}, fmt.Errorf("%w: unsupported normalized inline wrapper kind %d", ErrInvalidInput, kind)
	}
	payload.boundaryReady = true
	return payload, nil
}

func (r *renderer) normalizeInlineLeaf(event parser.SemanticEvent, tableCell bool) (inlineNormalizedPayload, error) {
	if event.Phase != parser.SemanticLeaf {
		return inlineNormalizedPayload{}, fmt.Errorf("%w: inline normalization phase %d", ErrInvalidInput, event.Phase)
	}

	payload := inlineNormalizedPayload{present: true}
	switch event.Kind {
	case parser.SemanticText:
		payload.value = escapeText(event.Value)
	case parser.SemanticSoftBreak:
		payload.value = "\n"
		payload.payloadReady = true
		payload.boundaryReady = true
	case parser.SemanticHardBreak:
		payload.value = "\\\n"
		payload.payloadReady = true
		payload.boundaryReady = true
	case parser.SemanticCodeSpan:
		payload.value = renderCodeSpan(event.Value, tableCell)
		payload.payloadReady = true
		payload.boundaryReady = true
	case parser.SemanticAutoLink:
		payload.value, payload.bareAutoLinkTail = r.renderAutoLink(event)
		payload.payloadReady = true
		payload.boundaryReady = true
	case parser.SemanticRawHTML:
		payload.value = normalizeLineEndings(event.Value)
	case parser.SemanticFootnoteReference:
		payload.value = "[^" + event.Label + "]"
		payload.payloadReady = true
		payload.boundaryReady = true
	default:
		return inlineNormalizedPayload{}, fmt.Errorf("%w: unsupported normalized inline leaf kind %d", ErrInvalidInput, event.Kind)
	}

	if payload.boundaryReady {
		payload.boundary = analyzeInlineBoundary([]byte(payload.value))
	}
	return payload, nil
}

type inlineSequenceState struct {
	hasOutput        bool
	lineStart        bool
	bareAutoLinkTail bool
}

func (s *inlineSequenceState) append(payload inlineNormalizedPayload) {
	if s == nil || !payload.payloadReady {
		return
	}
	if payload.value != "" {
		s.hasOutput = true
		s.lineStart = payload.value[len(payload.value)-1] == '\n'
	}
	s.bareAutoLinkTail = payload.bareAutoLinkTail
}

func (s *inlineSequenceState) observe(payload inlineNormalizedPayload) bool {
	if s == nil || !payload.present {
		return false
	}
	if payload.payloadReady {
		s.append(payload)
		return true
	}
	if !payload.boundaryReady {
		return false
	}
	if !payload.boundary.empty {
		s.hasOutput = true
		s.lineStart = false
	}
	s.bareAutoLinkTail = false
	return true
}

func (r *renderer) resolveContextualInlineLeaf(
	event parser.SemanticEvent,
	previous parser.SemanticEvent,
	state inlineSequenceState,
) (inlineNormalizedPayload, bool) {
	if event.Phase != parser.SemanticLeaf {
		return inlineNormalizedPayload{}, false
	}
	switch event.Kind {
	case parser.SemanticText:
		if contextualTextNeedsDelimiterBoundary(event, previous) {
			return inlineNormalizedPayload{present: true, value: escapeText(event.Value)}, false
		}
		value := escapeText(event.Value)
		if state.lineStart {
			value = escapeLeadingTextSpaces(event.Value)
		}
		if state.bareAutoLinkTail {
			value = escapeTextAfterBareAutoLink(event.Value)
		}
		return finalInlinePayload(value, false), true
	case parser.SemanticRawHTML:
		prefix := ""
		if state.hasOutput && state.lineStart {
			prefix = "    "
		}
		return finalInlinePayload(prefix+normalizeLineEndings(event.Value), false), true
	default:
		return inlineNormalizedPayload{}, false
	}
}

func contextualTextNeedsDelimiterBoundary(event, previous parser.SemanticEvent) bool {
	if event.Value == "" || previous.Range.End != event.Range.Start {
		return false
	}
	switch previous.Kind {
	case parser.SemanticEmphasis, parser.SemanticStrong:
	default:
		return false
	}
	switch event.Value[0] {
	case '*', '_', '\t':
		return true
	default:
		return false
	}
}

func finalInlinePayload(value string, bareAutoLinkTail bool) inlineNormalizedPayload {
	return inlineNormalizedPayload{
		value:            value,
		boundary:         analyzeInlineBoundary([]byte(value)),
		present:          true,
		payloadReady:     true,
		boundaryReady:    true,
		bareAutoLinkTail: bareAutoLinkTail,
	}
}
