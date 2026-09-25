package rendermarkdown

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestBuildInlineASTPreservesNestedStructure(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 1, End: 2}, Value: "a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 4, End: 6}, Value: "bc"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
	}

	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	if len(ast.nodes) != 5 {
		t.Fatalf("node count = %d, want 5 including synthetic root", len(ast.nodes))
	}

	root := ast.nodes[ast.root]
	emphasis := root.firstChild
	if emphasis <= ast.root {
		t.Fatalf("root first child = %d, want emphasis node", emphasis)
	}
	textA := ast.nodes[emphasis].firstChild
	strong := ast.nodes[textA].nextSibling
	textBC := ast.nodes[strong].firstChild

	if got := ast.events[ast.nodes[emphasis].eventIndex].Kind; got != parser.SemanticEmphasis {
		t.Fatalf("emphasis kind = %v", got)
	}
	if ast.nodes[emphasis].parent != ast.root {
		t.Fatalf("emphasis parent = %d, want root %d", ast.nodes[emphasis].parent, ast.root)
	}
	if got := ast.events[ast.nodes[textA].eventIndex].Value; got != "a" {
		t.Fatalf("first text = %q, want a", got)
	}
	if ast.nodes[strong].parent != emphasis {
		t.Fatalf("strong parent = %d, want %d", ast.nodes[strong].parent, emphasis)
	}
	if got := ast.events[ast.nodes[textBC].eventIndex].Value; got != "bc" {
		t.Fatalf("nested text = %q, want bc", got)
	}
}

func TestInlineASTWorkspaceBuildsIncrementally(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 1, End: 2}, Value: "a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 4, End: 6}, Value: "bc"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
	}

	workspace := newInlineASTWorkspace()
	workspace.beginBuild()
	for _, event := range events {
		if err := workspace.appendEvent(event); err != nil {
			t.Fatalf("appendEvent(%v) error = %v", event.Kind, err)
		}
	}
	ast, err := workspace.finishBuild()
	if err != nil {
		t.Fatalf("finishBuild() error = %v", err)
	}

	var replayed []parser.SemanticEvent
	if err := workspace.walk(ast, func(event parser.SemanticEvent) error {
		replayed = append(replayed, event)
		return nil
	}); err != nil {
		t.Fatalf("workspace.walk() error = %v", err)
	}
	if !reflect.DeepEqual(replayed, events) {
		t.Fatalf("incremental replay differs\n got: %#v\nwant: %#v", replayed, events)
	}
}

func TestWalkInlineASTReplaysExactEvents(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 1, End: 2}, Value: "a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 4, End: 6}, Value: "bc"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong, Range: parser.Range{Start: 2, End: 8}},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 0, End: 9}},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}

	var replayed []parser.SemanticEvent
	if err := walkInlineAST(ast, func(event parser.SemanticEvent) error {
		replayed = append(replayed, event)
		return nil
	}); err != nil {
		t.Fatalf("walkInlineAST() error = %v", err)
	}
	if !reflect.DeepEqual(replayed, events) {
		t.Fatalf("replayed events differ\n got: %#v\nwant: %#v", replayed, events)
	}
}

func TestInlineASTHostVisitorPreservesNativeEventStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
	}{
		{name: "paragraph wrappers", source: "a *b **c** d* [e](</x>)\n"},
		{name: "heading", source: "## a *b* c\n"},
		{name: "table cells", source: "| a *b* | ~~c~~ |\n| --- | --- |\n"},
		{name: "task paragraph", source: "- [x] a *b*\n"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			backend := native.New()
			var direct []parser.SemanticEvent
			if err := backend.WalkSemantic([]byte(test.source), func(event parser.SemanticEvent) error {
				direct = append(direct, event)
				return nil
			}); err != nil {
				t.Fatalf("direct WalkSemantic() error = %v", err)
			}

			var routed []parser.SemanticEvent
			router := newInlineASTHostVisitor(func(event parser.SemanticEvent) error {
				routed = append(routed, event)
				return nil
			})
			if err := backend.WalkSemantic([]byte(test.source), router.visit); err != nil {
				t.Fatalf("AST-routed WalkSemantic() error = %v", err)
			}
			if err := router.finish(); err != nil {
				t.Fatalf("router.finish() error = %v", err)
			}
			if !reflect.DeepEqual(routed, direct) {
				t.Fatalf("AST-routed event stream differs\n got: %#v\nwant: %#v", routed, direct)
			}
		})
	}
}

func TestAnalyzeInlineASTDelimitersPreservesSemanticHierarchy(t *testing.T) {
	t.Parallel()

	source := []byte("*a **b _c_** d*\n")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}

	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	analysis, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	if len(analysis.nodes) != 3 {
		t.Fatalf("delimiter node count = %d, want 3", len(analysis.nodes))
	}

	wantParents := []int{noInlineDelimiterNode, 0, 1}
	wantWidths := []int{1, 2, 1}
	wantMarkers := []byte{'*', '*', '_'}
	for index, node := range analysis.nodes {
		if node.parent != wantParents[index] {
			t.Fatalf("delimiter %d parent = %d, want %d", index, node.parent, wantParents[index])
		}
		if node.width != wantWidths[index] {
			t.Fatalf("delimiter %d width = %d, want %d", index, node.width, wantWidths[index])
		}
		if node.sourceMarker != wantMarkers[index] {
			t.Fatalf("delimiter %d source marker = %q, want %q", index, node.sourceMarker, wantMarkers[index])
		}
	}
}

func TestAnalyzeInlineASTDelimitersRecordsForcedSourceOwnership(t *testing.T) {
	t.Parallel()

	source := []byte("**#*x**\n")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	analysis, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	if len(analysis.nodes) != 2 {
		t.Fatalf("delimiter node count = %d, want 2", len(analysis.nodes))
	}
	if got := analysis.nodes[0].sourceOwnedMarker; got != '*' {
		t.Fatalf("outer source-owned marker = %q, want '*'", got)
	}
	if !analysis.nodes[0].sourceOwnedPrefix {
		t.Fatal("outer source-owned prefix = false, want true")
	}
	if got := analysis.nodes[1].sourceOwnedMarker; got != 0 {
		t.Fatalf("inner source-owned marker = %q, want none", got)
	}
}

func TestAnalyzeInlineASTDelimitersRecordsSharedRunAncestorOwnership(t *testing.T) {
	t.Parallel()

	source := []byte("**!\t*_\x00**\r\nc*")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	analysis, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	if len(analysis.nodes) != 3 {
		t.Fatalf("delimiter node count = %d, want 3", len(analysis.nodes))
	}
	want := []byte{'*', '*', 0}
	for index, marker := range want {
		if got := analysis.nodes[index].sourceOwnedMarker; got != marker {
			t.Fatalf("delimiter %d source-owned marker = %q, want %q", index, got, marker)
		}
	}
}

func TestAnalyzeInlineBoundaryUsesParserOwnedClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  inlineBoundaryFacts
	}{
		{
			name: "empty",
			want: inlineBoundaryFacts{
				empty: true,
				left:  inlineBoundaryEdge{whitespace: true},
				right: inlineBoundaryEdge{whitespace: true},
			},
		},
		{
			name:  "marker runs",
			value: "**alpha_",
			want: inlineBoundaryFacts{
				left:  inlineBoundaryEdge{marker: '*', runLength: 2},
				right: inlineBoundaryEdge{marker: '_', runLength: 1},
			},
		},
		{
			name:  "whitespace beyond run",
			value: "***\t",
			want: inlineBoundaryFacts{
				left:  inlineBoundaryEdge{marker: '*', runLength: 3, whitespace: true},
				right: inlineBoundaryEdge{whitespace: true},
			},
		},
		{
			name:  "unicode symbol beyond run",
			value: "*€",
			want: inlineBoundaryFacts{
				left:  inlineBoundaryEdge{marker: '*', runLength: 1, punctuation: true},
				right: inlineBoundaryEdge{punctuation: true},
			},
		},
		{
			name:  "escaped trailing marker is literal punctuation",
			value: "\\_",
			want: inlineBoundaryFacts{
				left:  inlineBoundaryEdge{punctuation: true},
				right: inlineBoundaryEdge{punctuation: true},
			},
		},
		{
			name:  "escaped marker stops trailing active run",
			value: "\\__",
			want: inlineBoundaryFacts{
				left:  inlineBoundaryEdge{punctuation: true},
				right: inlineBoundaryEdge{marker: '_', runLength: 1, punctuation: true},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := analyzeInlineBoundary([]byte(test.value)); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("analyzeInlineBoundary(%q) = %#v, want %#v", test.value, got, test.want)
			}
		})
	}
}

func TestNormalizeInlineASTMarksPreservableTabBeforeNestedDelimiter(t *testing.T) {
	t.Parallel()

	source := []byte("!\t*_x_*")
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 0, End: 2}, Value: "!\t"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 2, End: 7}},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 3, End: 6}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 4, End: 5}, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 3, End: 6}},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 2, End: 7}},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{source: source}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	textNode := ast.nodes[ast.root].firstChild
	got := normalization.nodes[textNode].boundary.right
	if !got.preservableTab || !got.punctuation || got.whitespace {
		t.Fatalf("trailing TAB boundary = %#v, want canonical punctuation with preservable source TAB", got)
	}

	before := inlineFixedNeighbor(got)
	after := inlineFixedNeighbor(inlineBoundaryEdge{punctuation: true})
	geometry := inlineFixedDelimiterCandidate(before, after, 1, '*')
	if !geometry.resolved || !geometry.canOpen || geometry.canClose {
		t.Fatalf("TAB-preserved geometry = %#v, want open-only delimiter", geometry)
	}
}

func TestNormalizeInlineLeafSeparatesAtomicAndContextualPayloads(t *testing.T) {
	t.Parallel()

	source := []byte("<https://example.test>")
	renderer := renderer{source: source}
	tests := []struct {
		name         string
		event        parser.SemanticEvent
		tableCell    bool
		wantValue    string
		wantReady    bool
		wantBareTail bool
	}{
		{
			name:      "soft break",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticSoftBreak},
			wantValue: "\n",
			wantReady: true,
		},
		{
			name:      "hard break",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticHardBreak},
			wantValue: "\\\n",
			wantReady: true,
		},
		{
			name:      "code span",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticCodeSpan, Value: "a`b"},
			wantValue: renderCodeSpan("a`b", false),
			wantReady: true,
		},
		{
			name:      "autolink",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Range: parser.Range{Start: 0, End: len(source)}, Value: "https://example.test", AutoLinkForm: parser.AutoLinkExplicitURI},
			wantValue: "<https://example.test>",
			wantReady: true,
		},
		{
			name:      "footnote",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticFootnoteReference, Label: "n"},
			wantValue: "[^n]",
			wantReady: true,
		},
		{
			name:      "text remains contextual",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "  a"},
			wantValue: escapeText("  a"),
		},
		{
			name:      "raw html remains contextual",
			event:     parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<i>\r\nx</i>"},
			wantValue: normalizeLineEndings("<i>\r\nx</i>"),
		},
		{
			name:         "bare autolink tail",
			event:        parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "www.example.test", AutoLinkForm: parser.AutoLinkExtendedWWW},
			wantValue:    "www.example.test",
			wantReady:    true,
			wantBareTail: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := renderer.normalizeInlineLeaf(test.event, test.tableCell)
			if err != nil {
				t.Fatalf("normalizeInlineLeaf() error = %v", err)
			}
			if got.value != test.wantValue {
				t.Fatalf("value = %q, want %q", got.value, test.wantValue)
			}
			if got.boundaryReady != test.wantReady {
				t.Fatalf("boundaryReady = %v, want %v", got.boundaryReady, test.wantReady)
			}
			if got.bareAutoLinkTail != test.wantBareTail {
				t.Fatalf("bareAutoLinkTail = %v, want %v", got.bareAutoLinkTail, test.wantBareTail)
			}
			if got.boundaryReady {
				if want := analyzeInlineBoundary([]byte(got.value)); !reflect.DeepEqual(got.boundary, want) {
					t.Fatalf("boundary = %#v, want %#v", got.boundary, want)
				}
			}
		})
	}
}

func TestNormalizeInlineASTSeparatesWrapperAndDelimiterBoundaries(t *testing.T) {
	t.Parallel()

	source := []byte("~~[a](</x>)~~ *b* `c`\n")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}

	renderer := renderer{source: source}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	seen := map[parser.SemanticKind]int{}
	for astNode := ast.root + 1; astNode < len(ast.nodes); astNode++ {
		node := ast.nodes[astNode]
		event := ast.events[node.eventIndex]
		normalized := normalization.nodes[astNode]
		switch event.Kind {
		case parser.SemanticStrikethrough, parser.SemanticLink:
			seen[event.Kind]++
			if !normalized.boundaryReady || normalized.payloadReady {
				t.Fatalf("wrapper kind %v readiness = boundary:%v payload:%v", event.Kind, normalized.boundaryReady, normalized.payloadReady)
			}
		case parser.SemanticEmphasis:
			seen[event.Kind]++
			if normalized.boundaryReady || normalized.payloadReady {
				t.Fatalf("emphasis must remain symbolic: %#v", normalized)
			}
		case parser.SemanticCodeSpan:
			seen[event.Kind]++
			if !normalized.boundaryReady || !normalized.payloadReady {
				t.Fatalf("code span must be fully normalized: %#v", normalized)
			}
		}
	}
	for _, kind := range []parser.SemanticKind{
		parser.SemanticStrikethrough,
		parser.SemanticLink,
		parser.SemanticEmphasis,
		parser.SemanticCodeSpan,
	} {
		if seen[kind] == 0 {
			t.Fatalf("did not observe semantic kind %v", kind)
		}
	}
}

func TestResolveContextualInlineLeafUsesSequenceStateAndDefersDelimiterSensitiveText(t *testing.T) {
	t.Parallel()

	renderer := renderer{}
	state := inlineSequenceState{lineStart: true}

	leading := parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "  a"}
	got, ready := renderer.resolveContextualInlineLeaf(leading, parser.SemanticEvent{}, state)
	if !ready || got.value != escapeLeadingTextSpaces(leading.Value) {
		t.Fatalf("leading text = %#v ready=%v", got, ready)
	}
	state.append(got)

	bare := inlineNormalizedPayload{
		value:            "www.example.test",
		present:          true,
		payloadReady:     true,
		boundaryReady:    true,
		bareAutoLinkTail: true,
	}
	state.append(bare)
	tail := parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: ".next"}
	got, ready = renderer.resolveContextualInlineLeaf(tail, parser.SemanticEvent{}, state)
	if !ready || got.value != escapeTextAfterBareAutoLink(tail.Value) {
		t.Fatalf("bare-autolink tail = %#v ready=%v", got, ready)
	}

	state = inlineSequenceState{hasOutput: true, lineStart: true}
	raw := parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<i>\r\nx</i>"}
	got, ready = renderer.resolveContextualInlineLeaf(raw, parser.SemanticEvent{}, state)
	if !ready || got.value != "    "+normalizeLineEndings(raw.Value) {
		t.Fatalf("line-start raw HTML = %#v ready=%v", got, ready)
	}

	previous := parser.SemanticEvent{
		Phase: parser.SemanticEnter,
		Kind:  parser.SemanticEmphasis,
		Range: parser.Range{Start: 0, End: 3},
	}
	sensitive := parser.SemanticEvent{
		Phase: parser.SemanticLeaf,
		Kind:  parser.SemanticText,
		Range: parser.Range{Start: 3, End: 5},
		Value: "_x",
	}
	if got, ready = renderer.resolveContextualInlineLeaf(sensitive, previous, inlineSequenceState{}); ready {
		t.Fatalf("delimiter-sensitive text unexpectedly finalized: %#v", got)
	}
}

func TestNormalizeInlineASTResolvesDirectChildrenUntilSymbolicBoundary(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 0, End: 3}, Value: "  a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticLink, Range: parser.Range{Start: 3, End: 11}, Destination: "/x"},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 4, End: 5}, Value: "b"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticLink, Range: parser.Range{Start: 3, End: 11}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 11, End: 13}, Value: " c"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 13, End: 16}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 14, End: 15}, Value: "d"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis, Range: parser.Range{Start: 13, End: 16}},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: 16, End: 18}, Value: " e"},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}

	rootChildren := []int{}
	for child := ast.nodes[ast.root].firstChild; child != noInlineASTNode; child = ast.nodes[child].nextSibling {
		rootChildren = append(rootChildren, child)
	}
	if len(rootChildren) != 5 {
		t.Fatalf("root child count = %d, want 5", len(rootChildren))
	}

	first := normalization.nodes[rootChildren[0]]
	if !first.payloadReady || first.value != escapeLeadingTextSpaces("  a") {
		t.Fatalf("first text = %#v", first)
	}
	link := rootChildren[1]
	linkChild := ast.nodes[link].firstChild
	if got := normalization.nodes[linkChild]; !got.payloadReady || got.value != escapeLeadingTextSpaces("b") {
		t.Fatalf("link child text = %#v", got)
	}
	afterLink := normalization.nodes[rootChildren[2]]
	if !afterLink.payloadReady || afterLink.value != escapeText(" c") {
		t.Fatalf("text after link = %#v", afterLink)
	}
	if got := normalization.nodes[rootChildren[4]]; got.payloadReady {
		t.Fatalf("text after symbolic emphasis unexpectedly finalized: %#v", got)
	}

	rootSequence := normalization.sequences[ast.root]
	if rootSequence.complete || rootSequence.blockedAt != rootChildren[3] || rootSequence.lastResolved != rootChildren[2] {
		t.Fatalf("root sequence = %#v", rootSequence)
	}
	linkSequence := normalization.sequences[link]
	if !linkSequence.complete || linkSequence.blockedAt != noInlineASTNode || !linkSequence.hasBoundary {
		t.Fatalf("link sequence = %#v", linkSequence)
	}
	wantLinkBoundary := analyzeInlineBoundary([]byte(normalization.nodes[linkChild].value))
	if !reflect.DeepEqual(linkSequence.left, wantLinkBoundary.left) ||
		!reflect.DeepEqual(linkSequence.right, wantLinkBoundary.right) {
		t.Fatalf("link sequence boundary = %#v, want %#v", linkSequence, wantLinkBoundary)
	}

	emphasis := rootChildren[3]
	emphasisChild := ast.nodes[emphasis].firstChild
	emphasisSequence := normalization.sequences[emphasis]
	if !emphasisSequence.complete || !emphasisSequence.hasBoundary {
		t.Fatalf("emphasis child sequence = %#v", emphasisSequence)
	}
	wantEmphasisBoundary := analyzeInlineBoundary([]byte(normalization.nodes[emphasisChild].value))
	if !reflect.DeepEqual(emphasisSequence.left, wantEmphasisBoundary.left) ||
		!reflect.DeepEqual(emphasisSequence.right, wantEmphasisBoundary.right) {
		t.Fatalf("emphasis child boundary = %#v, want %#v", emphasisSequence, wantEmphasisBoundary)
	}
}

func TestBuildInlineDelimiterNeighborhoodLinksNestedAndAdjacentEndpoints(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "y"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	delimiters, err := analyzeInlineASTDelimiters(nil, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	graph, err := buildInlineDelimiterNeighborhood(ast, normalization, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterNeighborhood() error = %v", err)
	}
	if len(graph.sites) != 6 {
		t.Fatalf("site count = %d, want 6", len(graph.sites))
	}

	assertEndpointNeighbor := func(site inlineDelimiterSite, inner bool, delimiter int, side inlineDelimiterSide) {
		t.Helper()
		neighbor := site.outer
		if inner {
			neighbor = site.inner
		}
		if neighbor.kind != inlineDelimiterNeighborEndpoint ||
			neighbor.delimiter != delimiter || neighbor.side != side {
			t.Fatalf("neighbor = %#v, want delimiter=%d side=%v", neighbor, delimiter, side)
		}
	}

	outerOpen := graph.site(0, inlineDelimiterOpen)
	outerClose := graph.site(0, inlineDelimiterClose)
	strongOpen := graph.site(1, inlineDelimiterOpen)
	strongClose := graph.site(1, inlineDelimiterClose)
	siblingOpen := graph.site(2, inlineDelimiterOpen)

	assertEndpointNeighbor(outerOpen, true, 1, inlineDelimiterOpen)
	assertEndpointNeighbor(outerClose, true, 1, inlineDelimiterClose)
	assertEndpointNeighbor(strongOpen, false, 0, inlineDelimiterOpen)
	assertEndpointNeighbor(strongClose, false, 0, inlineDelimiterClose)
	assertEndpointNeighbor(outerClose, false, 2, inlineDelimiterOpen)
	assertEndpointNeighbor(siblingOpen, false, 0, inlineDelimiterClose)

	if strongOpen.inner.kind != inlineDelimiterNeighborFixed ||
		strongClose.inner.kind != inlineDelimiterNeighborFixed {
		t.Fatalf("strong inner neighbors = open:%#v close:%#v", strongOpen.inner, strongClose.inner)
	}
}

func TestInlineDelimiterFixedGeometryUsesParserFlanking(t *testing.T) {
	t.Parallel()

	site := inlineDelimiterSite{
		side:  inlineDelimiterOpen,
		outer: inlineFixedNeighbor(inlineBoundaryEdge{}),
		inner: inlineFixedNeighbor(inlineBoundaryEdge{}),
	}
	geometry := analyzeInlineDelimiterFixedGeometry(site, 1)
	if !geometry.star.resolved || !geometry.star.canOpen || !geometry.star.canClose {
		t.Fatalf("star geometry = %#v", geometry.star)
	}
	if !geometry.underscore.resolved || geometry.underscore.canOpen || geometry.underscore.canClose {
		t.Fatalf("underscore intraword geometry = %#v", geometry.underscore)
	}
	if geometry.allowedOpen != inlineMarkerStar {
		t.Fatalf("allowed open mask = %d, want star-only", geometry.allowedOpen)
	}

	extended := inlineDelimiterSite{
		side:  inlineDelimiterOpen,
		outer: inlineFixedNeighbor(inlineBoundaryEdge{whitespace: true}),
		inner: inlineFixedNeighbor(inlineBoundaryEdge{marker: '*', runLength: 2}),
	}
	geometry = analyzeInlineDelimiterFixedGeometry(extended, 1)
	if geometry.star.runLength != 3 {
		t.Fatalf("coalesced star run length = %d, want 3", geometry.star.runLength)
	}
	if geometry.underscore.runLength != 1 {
		t.Fatalf("underscore run length = %d, want 1", geometry.underscore.runLength)
	}
}

func TestBuildInlineDelimiterCoalescencesTracksAdjacentEndpointRuns(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "y"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	delimiters, err := analyzeInlineASTDelimiters(nil, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	neighborhood, err := buildInlineDelimiterNeighborhood(ast, normalization, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterNeighborhood() error = %v", err)
	}
	edges := buildInlineDelimiterCoalescences(neighborhood, delimiters)
	if len(edges) != 3 {
		t.Fatalf("coalescence edge count = %d, want 3", len(edges))
	}

	baseLengths := map[int]int{}
	resolved := 0
	for _, edge := range edges {
		baseLengths[edge.baseRunLength]++
		if !edge.resolved {
			continue
		}
		resolved++
		if edge.starRunLength != 3 || edge.underscoreRunLength != 3 {
			t.Fatalf("resolved two-endpoint run = %#v, want length 3 for both markers", edge)
		}
	}
	if baseLengths[3] != 2 || baseLengths[2] != 1 {
		t.Fatalf("base coalesced lengths = %#v, want two 3-runs and one 2-run", baseLengths)
	}
	if resolved != 1 {
		t.Fatalf("resolved edge count = %d, want 1; endpoint chains must remain unresolved", resolved)
	}

	components, err := buildInlineDelimiterRunComponents(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterRunComponents() error = %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("run component count = %d, want 2", len(components))
	}
	var openOnly, mixed *inlineDelimiterRunComponent
	for index := range components {
		component := &components[index]
		switch {
		case component.requiredOpen && !component.requiredClose:
			openOnly = component
		case component.requiredOpen && component.requiredClose:
			mixed = component
		}
	}
	if openOnly == nil || !openOnly.star.allowed || !openOnly.underscore.allowed ||
		openOnly.star.runLength != 3 || openOnly.underscore.runLength != 3 {
		t.Fatalf("open-only component = %#v", openOnly)
	}
	if mixed == nil || !mixed.star.allowed || mixed.underscore.allowed ||
		mixed.star.runLength != 4 || mixed.underscore.runLength != 4 {
		t.Fatalf("mixed component = %#v", mixed)
	}
}

func TestNormalizeInlineASTPreservesTableCellCodeSpanPolicy(t *testing.T) {
	t.Parallel()

	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticCodeSpan, Value: "a|b"},
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{}
	normalization, err := renderer.normalizeInlineAST(ast, true)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	child := ast.nodes[ast.root].firstChild
	got := normalization.nodes[child]
	want := renderCodeSpan("a|b", true)
	if !got.payloadReady || got.value != want {
		t.Fatalf("table-cell code span = %#v, want %q", got, want)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsDerivesUnaryMasks(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{eventIndex: noInlineASTNode},
			{eventIndex: 0, exitIndex: 1},
		},
		root: 0,
	}
	delimiters := inlineDelimiterAnalysis{
		nodes: []inlineDelimiterNode{{astNode: 1, width: 1}},
	}
	letter := inlineFixedNeighbor(inlineBoundaryEdge{})
	neighborhood := inlineDelimiterNeighborhood{
		sites: []inlineDelimiterSite{
			{delimiter: 0, side: inlineDelimiterOpen, outer: letter, inner: letter},
			{delimiter: 0, side: inlineDelimiterClose, inner: letter, outer: letter},
		},
	}

	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	if got := constraints.nodes[0].allowed; got != inlineMarkerStar {
		t.Fatalf("unary allowed mask = %d, want star-only", got)
	}
	if len(constraints.pairs) != 0 {
		t.Fatalf("pair constraint count = %d, want 0", len(constraints.pairs))
	}
}

func TestBuildInlineDelimiterBooleanConstraintsDerivesResolvedPairMask(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{eventIndex: noInlineASTNode},
			{eventIndex: 0, exitIndex: 1},
			{eventIndex: 2, exitIndex: 3},
		},
		root: 0,
	}
	delimiters := inlineDelimiterAnalysis{
		nodes: []inlineDelimiterNode{
			{astNode: 1, width: 1},
			{astNode: 2, width: 1},
		},
	}
	letter := inlineFixedNeighbor(inlineBoundaryEdge{})
	neighborhood := inlineDelimiterNeighborhood{
		sites: []inlineDelimiterSite{
			{delimiter: 0, side: inlineDelimiterOpen},
			{
				delimiter: 0,
				side:      inlineDelimiterClose,
				inner:     letter,
				outer:     inlineEndpointNeighbor(1, inlineDelimiterOpen),
			},
			{
				delimiter: 1,
				side:      inlineDelimiterOpen,
				outer:     inlineEndpointNeighbor(0, inlineDelimiterClose),
				inner:     letter,
			},
			{delimiter: 1, side: inlineDelimiterClose},
		},
	}

	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	if len(constraints.pairs) != 1 {
		t.Fatalf("pair constraint count = %d, want 1", len(constraints.pairs))
	}
	pair := constraints.pairs[0]
	want := inlinePairStarStar | inlinePairStarUnderscore | inlinePairUnderscoreStar
	if pair.first != 0 || pair.second != 1 || pair.allowed != want {
		t.Fatalf("pair constraint = %#v, want first=0 second=1 allowed=%d", pair, want)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsAppliesFixedModuloThree(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{eventIndex: noInlineASTNode},
			{eventIndex: 0, exitIndex: 1},
		},
		root: 0,
	}
	delimiters := inlineDelimiterAnalysis{
		nodes: []inlineDelimiterNode{{astNode: 1, width: 1}},
	}
	letter := inlineFixedNeighbor(inlineBoundaryEdge{})
	starTail := inlineFixedNeighbor(inlineBoundaryEdge{marker: '*', runLength: 1})
	neighborhood := inlineDelimiterNeighborhood{
		sites: []inlineDelimiterSite{
			{delimiter: 0, side: inlineDelimiterOpen, outer: letter, inner: letter},
			{delimiter: 0, side: inlineDelimiterClose, inner: letter, outer: starTail},
		},
	}

	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	if got := constraints.nodes[0].allowed; got != 0 {
		t.Fatalf("allowed mask = %d, want 0 after modulo-three removes the star candidate", got)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsFoldsSelfCoalescenceIntoUnary(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{eventIndex: noInlineASTNode},
			{eventIndex: 0, exitIndex: 1},
		},
		root: 0,
	}
	delimiters := inlineDelimiterAnalysis{
		nodes: []inlineDelimiterNode{{astNode: 1, width: 1}},
	}
	letter := inlineFixedNeighbor(inlineBoundaryEdge{})
	neighborhood := inlineDelimiterNeighborhood{
		sites: []inlineDelimiterSite{
			{
				delimiter: 0,
				side:      inlineDelimiterOpen,
				outer:     letter,
				inner:     inlineEndpointNeighbor(0, inlineDelimiterClose),
			},
			{
				delimiter: 0,
				side:      inlineDelimiterClose,
				inner:     inlineEndpointNeighbor(0, inlineDelimiterOpen),
				outer:     letter,
			},
		},
	}

	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	if got := constraints.nodes[0].allowed; got != inlineMarkerStar {
		t.Fatalf("self-coalesced unary mask = %d, want star-only", got)
	}
	if len(constraints.pairs) != 0 {
		t.Fatalf("self-coalescence produced %d pair constraints, want 0", len(constraints.pairs))
	}
}

func TestBuildInlineDelimiterBooleanConstraintsAppliesSourceOwnership(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{eventIndex: noInlineASTNode},
			{eventIndex: 0, exitIndex: 1},
		},
		root: 0,
	}
	delimiters := inlineDelimiterAnalysis{
		nodes: []inlineDelimiterNode{
			{astNode: 1, width: 1, sourceMarker: '_', sourceOwnedMarker: '_'},
		},
	}
	neighborhood := inlineDelimiterNeighborhood{
		sites: []inlineDelimiterSite{
			{delimiter: 0, side: inlineDelimiterOpen},
			{delimiter: 0, side: inlineDelimiterClose},
		},
	}

	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	if got := constraints.nodes[0].allowed; got != inlineMarkerUnderscore {
		t.Fatalf("source-owned unary mask = %d, want underscore-only", got)
	}
}

func TestInlineOuterNeighborUsesFacingSiblingBoundary(t *testing.T) {
	t.Parallel()

	ast := inlineAST{
		nodes: []inlineASTNode{
			{firstChild: 1},
			{parent: 0, nextSibling: 2},
			{parent: 0, nextSibling: 3},
			{parent: 0, nextSibling: noInlineASTNode},
		},
		root: 0,
	}
	normalization := inlineASTNormalization{
		nodes: []inlineNormalizedPayload{
			{},
			{
				boundaryReady: true,
				boundary: inlineBoundaryFacts{
					left:  inlineBoundaryEdge{punctuation: true},
					right: inlineBoundaryEdge{whitespace: true},
				},
			},
			{},
			{
				boundaryReady: true,
				boundary: inlineBoundaryFacts{
					left:  inlineBoundaryEdge{whitespace: true},
					right: inlineBoundaryEdge{punctuation: true},
				},
			},
		},
	}
	delimiterByAST := []int{
		noInlineDelimiterNode,
		noInlineDelimiterNode,
		0,
		noInlineDelimiterNode,
	}
	previousSibling := []int{
		noInlineASTNode,
		noInlineASTNode,
		1,
		2,
	}

	openOuter := inlineOuterNeighbor(
		ast,
		normalization,
		delimiterByAST,
		previousSibling,
		2,
		inlineDelimiterOpen,
	)
	if openOuter.kind != inlineDelimiterNeighborFixed || !openOuter.edge.whitespace || openOuter.edge.punctuation {
		t.Fatalf("open outer = %#v, want previous sibling right whitespace boundary", openOuter)
	}

	closeOuter := inlineOuterNeighbor(
		ast,
		normalization,
		delimiterByAST,
		previousSibling,
		2,
		inlineDelimiterClose,
	)
	if closeOuter.kind != inlineDelimiterNeighborFixed || !closeOuter.edge.whitespace || closeOuter.edge.punctuation {
		t.Fatalf("close outer = %#v, want next sibling left whitespace boundary", closeOuter)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsAppliesPairModuloThree(t *testing.T) {
	t.Parallel()

	source := []byte("**!\t*_\x00**\r\nc*")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{source: source}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	delimiters, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	neighborhood, err := buildInlineDelimiterNeighborhood(ast, normalization, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterNeighborhood() error = %v", err)
	}
	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}

	found := false
	allowed := inlinePairStarStar |
		inlinePairStarUnderscore |
		inlinePairUnderscoreStar |
		inlinePairUnderscoreUnderscore
	for _, pair := range constraints.pairs {
		if pair.first != 1 || pair.second != 2 {
			continue
		}
		found = true
		allowed &= pair.allowed
	}
	if !found {
		t.Fatal("middle/leaf pair constraint not found")
	}
	if allowed&inlinePairStarStar != 0 {
		t.Fatalf("middle/leaf constraints allow star-star, want modulo-three rejection: %04b", allowed)
	}
	if allowed&inlinePairStarUnderscore == 0 {
		t.Fatalf("middle/leaf constraints reject star-underscore, want allowed: %04b", allowed)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsCombinesDualEdgePair(t *testing.T) {
	t.Parallel()

	source := []byte("*_!\t_*\x00*_\r\nc*")
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{source: source}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	delimiters, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	wantOwnership := []byte{'*', '_', 0}
	for index, marker := range wantOwnership {
		if got := delimiters.nodes[index].sourceOwnedMarker; got != marker {
			t.Fatalf("delimiter %d source-owned marker = %q, want %q", index, got, marker)
		}
	}
	neighborhood, err := buildInlineDelimiterNeighborhood(ast, normalization, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterNeighborhood() error = %v", err)
	}
	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}

	allowed := inlinePairStarStar |
		inlinePairStarUnderscore |
		inlinePairUnderscoreStar |
		inlinePairUnderscoreUnderscore
	found := false
	for _, pair := range constraints.pairs {
		if pair.first != 1 || pair.second != 2 {
			continue
		}
		found = true
		allowed &= pair.allowed
	}
	if !found {
		t.Fatal("middle/leaf pair constraint not found")
	}
	wantPair := inlinePairStarUnderscore | inlinePairUnderscoreStar
	if allowed != wantPair {
		t.Fatalf("middle/leaf combined pair mask = %04b, want different-marker relation %04b", allowed, wantPair)
	}
}

func TestBuildInlineDelimiterBooleanConstraintsComposesOwnedSharedOpenRelation(t *testing.T) {
	t.Parallel()

	allPairs := inlinePairStarStar |
		inlinePairStarUnderscore |
		inlinePairUnderscoreStar |
		inlinePairUnderscoreUnderscore
	samePairs := inlinePairStarStar | inlinePairUnderscoreUnderscore
	tests := []struct {
		name   string
		source []byte
		want   inlineDelimiterPairMask
	}{
		{
			name:   "pair_forces_opposite_neighbor",
			source: []byte("__!\t*_\x00*_\r\nc_"),
			want:   samePairs,
		},
		{
			name:   "unary_and_pair_force_opposite_neighbor",
			source: []byte("__:a*a*_\n\xfe_"),
			want:   samePairs,
		},
		{
			name:   "owned_prefix_keeps_relation_unresolved",
			source: []byte("**!\t__\x00_*\r\nc*"),
			want:   allPairs,
		},
		{
			name:   "neighbor_not_forced_opposite",
			source: []byte("**! _*\x00_*\r\nc*"),
			want:   allPairs,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := testInlineDelimiterPairMask(t, test.source, 0, 1)
			if !ok {
				t.Fatal("outer/middle pair constraint not found")
			}
			if got != test.want {
				t.Fatalf("outer/middle combined pair mask = %04b, want %04b", got, test.want)
			}
		})
	}
}

func TestBuildInlineDelimiterBooleanConstraintsComposesOwnedOpenCloseBridge(t *testing.T) {
	t.Parallel()

	allPairs := inlinePairStarStar |
		inlinePairStarUnderscore |
		inlinePairUnderscoreStar |
		inlinePairUnderscoreUnderscore
	tests := []struct {
		name   string
		source []byte
		want   inlineDelimiterPairMask
	}{
		{
			name:   "owned_star_forbids_outer_underscore_with_star_child",
			source: []byte("**! _*\x00_*\r\nc*"),
			want:   allPairs &^ inlinePairUnderscoreStar,
		},
		{
			name:   "owned_underscore_forbids_outer_star_with_underscore_child",
			source: []byte("__! *_\x00*_\r\nc_"),
			want:   allPairs &^ inlinePairStarUnderscore,
		},
		{
			name:   "owned_prefix_keeps_bridge_unresolved",
			source: []byte("**!\t__\x00_*\r\nc*"),
			want:   allPairs,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := testInlineDelimiterPairMask(t, test.source, 0, 2)
			if !ok {
				if test.want == allPairs {
					return
				}
				t.Fatal("outer/leaf pair constraint not found")
			}
			if got != test.want {
				t.Fatalf("outer/leaf combined pair mask = %04b, want %04b", got, test.want)
			}
		})
	}
}

func TestBuildInlineDelimiterBooleanConstraintsComposesOwnedAlternatingChain(t *testing.T) {
	t.Parallel()

	samePairs := inlinePairStarStar | inlinePairUnderscoreUnderscore
	tests := []struct {
		name   string
		source []byte
	}{
		{
			name:   "star_outer_underscore_middle",
			source: []byte("*_! *_\x00*_\r\nc*"),
		},
		{
			name:   "underscore_outer_star_middle",
			source: []byte("_*! _*\x00_*\r\nc_"),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := testInlineDelimiterPairMask(t, test.source, 0, 2)
			if !ok {
				t.Fatal("outer/leaf pair constraint not found")
			}
			if got != samePairs {
				t.Fatalf("outer/leaf combined pair mask = %04b, want same-marker relation %04b", got, samePairs)
			}
		})
	}
}

func testInlineDelimiterPairMask(
	t *testing.T,
	source []byte,
	first, second int,
) (inlineDelimiterPairMask, bool) {
	t.Helper()
	var events []parser.SemanticEvent
	inParagraph := false
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				inParagraph = true
			case parser.SemanticExit:
				inParagraph = false
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	ast, err := buildInlineAST(events)
	if err != nil {
		t.Fatalf("buildInlineAST() error = %v", err)
	}
	renderer := renderer{source: source}
	normalization, err := renderer.normalizeInlineAST(ast, false)
	if err != nil {
		t.Fatalf("normalizeInlineAST() error = %v", err)
	}
	delimiters, err := analyzeInlineASTDelimiters(source, ast)
	if err != nil {
		t.Fatalf("analyzeInlineASTDelimiters() error = %v", err)
	}
	neighborhood, err := buildInlineDelimiterNeighborhood(ast, normalization, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterNeighborhood() error = %v", err)
	}
	constraints, err := buildInlineDelimiterBooleanConstraints(ast, neighborhood, delimiters)
	if err != nil {
		t.Fatalf("buildInlineDelimiterBooleanConstraints() error = %v", err)
	}
	allowed := inlinePairStarStar |
		inlinePairStarUnderscore |
		inlinePairUnderscoreStar |
		inlinePairUnderscoreUnderscore
	found := false
	for _, pair := range constraints.pairs {
		if pair.first != first || pair.second != second {
			continue
		}
		found = true
		allowed &= pair.allowed
	}
	return allowed, found
}

func TestBuildInlineASTRejectsMismatchedExit(t *testing.T) {
	t.Parallel()

	_, err := buildInlineAST([]parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
	})
	if err == nil {
		t.Fatal("buildInlineAST() error = nil, want mismatch error")
	}
}
