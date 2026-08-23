package marksplice

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/splice"
)

type inlineConstructionKind uint8

const (
	inlineConstructionText inlineConstructionKind = iota + 1
	inlineConstructionCode
	inlineConstructionEmphasis
	inlineConstructionStrong
	inlineConstructionStrikethrough
	inlineConstructionLink
	inlineConstructionImage
	inlineConstructionAutoLink
	inlineConstructionBareAutoLink
	inlineConstructionReferenceLink
	inlineConstructionReferenceImage
)

// Inline is a construction-only typed inline value.
//
// Its zero value is invalid. Use the exported Inline construction functions
// rather than depending on its private representation.
type Inline struct {
	kind           inlineConstructionKind
	text           string
	destination    string
	reference      string
	title          string
	hasTitle       bool
	referenceForm  typedInlineReferenceForm
	referenceScope typedInlineReferenceScope
	children       []Inline
}

// TextInline returns semantic plain text for typed inline construction.
//
// M75 encodes ASCII punctuation with canonical GFM backslash escapes so caller
// text cannot become Markdown syntax implicitly. Validation occurs when the
// value is appended to a DocumentBuilder.
func TextInline(text string) Inline {
	return Inline{kind: inlineConstructionText, text: text}
}

// CodeInline returns one conservative single-line code span construction value.
//
// M76 selects an adaptive backtick delimiter longer than every internal run.
// Leading/trailing horizontal space and leading/trailing backticks are rejected
// because supporting those shapes would require semantic whitespace or delimiter
// normalization beyond the existing source-proven parsed CodeSpan capability.
func CodeInline(code string) Inline {
	return Inline{kind: inlineConstructionCode, text: code}
}

// EmphasisInline returns one conservative emphasis construction value.
// M88 permits bounded nesting of code, emphasis, strong, and strikethrough
// children while keeping links/images/autolinks outside this wrapper slice.
func EmphasisInline(content ...Inline) Inline {
	return Inline{kind: inlineConstructionEmphasis, children: cloneTypedInlineConstruction(content)}
}

// StrongInline returns one conservative strong-emphasis construction value.
// M88 applies the same bounded structured-child policy as EmphasisInline.
func StrongInline(content ...Inline) Inline {
	return Inline{kind: inlineConstructionStrong, children: cloneTypedInlineConstruction(content)}
}

// StrikethroughInline returns one conservative GFM strikethrough construction value.
// M88 permits bounded code/emphasis/strong children but rejects direct
// strikethrough-in-strikethrough nesting because adjacent tilde runs are ambiguous.
func StrikethroughInline(content ...Inline) Inline {
	return Inline{kind: inlineConstructionStrikethrough, children: cloneTypedInlineConstruction(content)}
}

// LinkInline returns one conservative inline-link construction value.
//
// M77 writes the destination in angle brackets, M87 adds the separate WithTitle
// constructor, and M92 permits the reviewed bounded structured-inline children.
func LinkInline(destination string, label ...Inline) Inline {
	return newTypedLinkOrImage(inlineConstructionLink, destination, "", false, label)
}

// LinkInlineWithTitle returns one conservative inline-link construction value
// with a canonical double-quoted title. M87 requires a non-empty title that needs
// no GFM escape or entity interpretation.
func LinkInlineWithTitle(destination, title string, label ...Inline) Inline {
	return newTypedLinkOrImage(inlineConstructionLink, destination, title, true, label)
}

// ImageInline returns one conservative inline-image construction value.
//
// M77 writes the destination in angle brackets, M87 adds the separate WithTitle
// constructor, and M92 permits the reviewed bounded structured-inline children.
func ImageInline(destination string, alt ...Inline) Inline {
	return newTypedLinkOrImage(inlineConstructionImage, destination, "", false, alt)
}

// ImageInlineWithTitle returns one conservative inline-image construction value
// with a canonical double-quoted title. M87 applies the same conservative title
// policy as LinkInlineWithTitle.
func ImageInlineWithTitle(destination, title string, alt ...Inline) Inline {
	return newTypedLinkOrImage(inlineConstructionImage, destination, title, true, alt)
}

func newTypedLinkOrImage(kind inlineConstructionKind, destination, title string, hasTitle bool, content []Inline) Inline {
	return Inline{
		kind:        kind,
		destination: destination,
		title:       title,
		hasTitle:    hasTitle,
		children:    cloneTypedInlineConstruction(content),
	}
}

// AutoLinkInline returns one canonical angle-autolink construction value.
// Validation succeeds only when reparsing produces the existing source-proven AutoLink capability.
func AutoLinkInline(value string) Inline {
	return Inline{kind: inlineConstructionAutoLink, text: value}
}

// BareAutoLinkInline returns one parser-proven GFM extended autolink token
// without adding angle brackets. The complete requested token must be owned by
// one AutoLink observation after reparsing or construction fails closed.
func BareAutoLinkInline(value string) Inline {
	return Inline{kind: inlineConstructionBareAutoLink, text: value}
}

// AppendHeadingContent appends one top-level ATX heading from typed inline content.
func (b *DocumentBuilder) AppendHeadingContent(level int, content ...Inline) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	inlineGFM, err := b.renderTypedInlineConstruction(content)
	if err != nil {
		return err
	}
	return b.AppendHeading(level, inlineGFM)
}

// AppendParagraphContent appends one top-level single-line paragraph from typed inline content.
func (b *DocumentBuilder) AppendParagraphContent(content ...Inline) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	inlineGFM, err := b.renderTypedInlineConstruction(content)
	if err != nil {
		return err
	}
	return b.AppendParagraph(inlineGFM)
}

// AppendBlockquoteContent appends one simple top-level blockquote from typed inline content.
func (b *DocumentBuilder) AppendBlockquoteContent(content ...Inline) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	inlineGFM, err := b.renderTypedInlineConstruction(content)
	if err != nil {
		return err
	}
	return b.AppendBlockquote(inlineGFM)
}

// AppendNestedBlockquoteContent appends one explicitly nested blockquote from typed inline content.
func (b *DocumentBuilder) AppendNestedBlockquoteContent(depth int, content ...Inline) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if depth < 2 || depth > maxConstructionBlockquoteDepth {
		return fmt.Errorf("%w: nested blockquote depth must be between 2 and %d", ErrInvalidConstruction, maxConstructionBlockquoteDepth)
	}
	inlineGFM, err := b.renderTypedInlineConstruction(content)
	if err != nil {
		return err
	}
	return b.AppendNestedBlockquote(depth, inlineGFM)
}

type typedInlineExpectation struct {
	kind             Kind
	contentRange     Range
	labelRange       Range
	titleRange       Range
	destination      string
	title            string
	angleDestination bool
	hasTitle         bool
}

const maxTypedInlineNestingDepth = 64

type typedInlineWritePolicy uint8

const (
	typedInlineTopLevel typedInlineWritePolicy = iota
	typedInlineStructuredNested
)

type typedInlineWriteContext struct {
	expected                     *[]typedInlineExpectation
	hierarchy                    *[]splice.ConstructionInlineExpectation
	linkImageExpected            *[]splice.ConstructionLinkImageExpectation
	referenceExpected            *[]splice.ConstructionReferenceInlineExpectation
	referenceDefinitions         []typedInlineReferenceDefinition
	deferredReferenceDefinitions []typedInlineReferenceDefinition
	policy                       typedInlineWritePolicy
	parent                       int
	depth                        int
	emphasisMarker               byte
	parentKind                   inlineConstructionKind
}

func (b *DocumentBuilder) renderTypedInlineConstruction(content []Inline) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("%w: typed inline content is empty", ErrInvalidConstruction)
	}
	var output strings.Builder
	expected := make([]typedInlineExpectation, 0, len(content))
	hierarchy := make([]splice.ConstructionInlineExpectation, 0, len(content))
	linkImageExpected := make([]splice.ConstructionLinkImageExpectation, 0, len(content))
	referenceExpected := make([]splice.ConstructionReferenceInlineExpectation, 0, len(content))
	context := typedInlineWriteContext{
		expected:                     &expected,
		hierarchy:                    &hierarchy,
		linkImageExpected:            &linkImageExpected,
		referenceExpected:            &referenceExpected,
		referenceDefinitions:         typedInlineReferenceDefinitions(b.blocks),
		deferredReferenceDefinitions: typedInlineReferenceDefinitions(b.deferredReferences),
		policy:                       typedInlineTopLevel,
		parent:                       -1,
	}
	for index, inline := range content {
		if err := writeTypedInlineConstruction(&output, inline, context); err != nil {
			return "", fmt.Errorf("%w: typed inline at index %d: %v", ErrInvalidConstruction, index, err)
		}
	}
	if output.Len() == 0 {
		return "", fmt.Errorf("%w: typed inline content is empty", ErrInvalidConstruction)
	}
	source := output.String()
	if err := validateTypedInlineConstruction(source, expected, hierarchy, linkImageExpected, referenceExpected); err != nil {
		return "", err
	}
	return source, nil
}

func writeTypedInlineConstruction(output *strings.Builder, inline Inline, context typedInlineWriteContext) error {
	if !typedInlineKindAllowed(context.policy, inline.kind) {
		return fmt.Errorf("nested typed inline kind is not supported")
	}
	switch inline.kind {
	case inlineConstructionText:
		return writeTypedInlineText(output, inline.text)
	case inlineConstructionCode:
		return writeTypedInlineCode(output, inline.text, context)
	case inlineConstructionEmphasis, inlineConstructionStrong, inlineConstructionStrikethrough:
		return writeTypedInlineDelimited(output, inline, context)
	case inlineConstructionLink, inlineConstructionImage:
		return writeTypedInlineLinkOrImage(output, inline, context)
	case inlineConstructionReferenceLink, inlineConstructionReferenceImage:
		return writeTypedInlineReferenceLinkOrImage(output, inline, context)
	case inlineConstructionAutoLink:
		return writeTypedInlineAutoLink(output, inline.text, true, context.expected)
	case inlineConstructionBareAutoLink:
		return writeTypedInlineAutoLink(output, inline.text, false, context.expected)
	default:
		return fmt.Errorf("unsupported typed inline kind")
	}
}

func typedInlineKindAllowed(policy typedInlineWritePolicy, kind inlineConstructionKind) bool {
	switch policy {
	case typedInlineTopLevel:
		return true
	case typedInlineStructuredNested:
		switch kind {
		case inlineConstructionText, inlineConstructionCode, inlineConstructionEmphasis, inlineConstructionStrong, inlineConstructionStrikethrough:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func writeTypedInlineText(output *strings.Builder, text string) error {
	if err := validateTypedInlineText(text); err != nil {
		return err
	}
	writeEscapedInlineText(output, text)
	return nil
}

func writeTypedInlineCode(output *strings.Builder, code string, context typedInlineWriteContext) error {
	if err := validateTypedInlineCode(code); err != nil {
		return err
	}
	fenceLength := typedInlineCodeFenceLength(code)
	fence := strings.Repeat("`", fenceLength)
	syntaxStart := output.Len()
	output.WriteString(fence)
	contentStart := output.Len()
	output.WriteString(code)
	contentEnd := output.Len()
	output.WriteString(fence)
	*context.expected = append(*context.expected, typedInlineExpectation{
		kind:         KindCodeSpan,
		contentRange: Range{Start: contentStart, End: contentEnd},
	})
	*context.hierarchy = append(*context.hierarchy, splice.ConstructionInlineExpectation{
		Kind:            splice.KindCodeSpan,
		SyntaxRange:     splice.Range{Start: syntaxStart, End: output.Len()},
		ContentRange:    splice.Range{Start: contentStart, End: contentEnd},
		Marker:          '`',
		DelimiterLength: fenceLength,
		Parent:          context.parent,
	})
	return nil
}

func writeTypedInlineDelimited(output *strings.Builder, inline Inline, context typedInlineWriteContext) error {
	if len(inline.children) == 0 {
		return fmt.Errorf("delimited inline content is empty")
	}
	if context.depth >= maxTypedInlineNestingDepth {
		return fmt.Errorf("typed inline nesting exceeds %d", maxTypedInlineNestingDepth)
	}
	if inline.kind == inlineConstructionStrikethrough && context.parentKind == inlineConstructionStrikethrough {
		return fmt.Errorf("direct nested strikethrough is not supported")
	}

	marker, delimiterLength, publicKind, proofKind := typedInlineDelimitedSyntax(inline.kind, context.emphasisMarker)
	markerText := strings.Repeat(string(marker), delimiterLength)
	syntaxStart := output.Len()
	proofIndex := len(*context.hierarchy)
	*context.hierarchy = append(*context.hierarchy, splice.ConstructionInlineExpectation{
		Kind:            proofKind,
		Marker:          marker,
		DelimiterLength: delimiterLength,
		Parent:          context.parent,
	})

	output.WriteString(markerText)
	contentStart := output.Len()
	childContext := context
	childContext.policy = typedInlineStructuredNested
	childContext.parent = proofIndex
	childContext.depth++
	childContext.parentKind = inline.kind
	if inline.kind == inlineConstructionEmphasis || inline.kind == inlineConstructionStrong {
		childContext.emphasisMarker = marker
	}
	for _, child := range inline.children {
		if err := writeTypedInlineConstruction(output, child, childContext); err != nil {
			return err
		}
	}
	contentEnd := output.Len()
	output.WriteString(markerText)
	(*context.hierarchy)[proofIndex].SyntaxRange = splice.Range{Start: syntaxStart, End: output.Len()}
	(*context.hierarchy)[proofIndex].ContentRange = splice.Range{Start: contentStart, End: contentEnd}

	if typedInlineChildrenAreText(inline.children) {
		*context.expected = append(*context.expected, typedInlineExpectation{
			kind:         publicKind,
			contentRange: Range{Start: contentStart, End: contentEnd},
		})
	}
	return nil
}

func typedInlineDelimitedSyntax(kind inlineConstructionKind, parentMarker byte) (byte, int, Kind, splice.Kind) {
	if kind == inlineConstructionStrikethrough {
		return '~', 2, KindStrikethrough, splice.KindStrikethrough
	}
	marker := byte('*')
	if parentMarker == '*' {
		marker = '_'
	}
	if kind == inlineConstructionStrong {
		return marker, 2, KindStrong, splice.KindStrong
	}
	return marker, 1, KindEmphasis, splice.KindEmphasis
}

func typedInlineChildrenAreText(children []Inline) bool {
	for _, child := range children {
		if child.kind != inlineConstructionText {
			return false
		}
	}
	return true
}

func writeTypedInlineLinkOrImage(output *strings.Builder, inline Inline, context typedInlineWriteContext) error {
	if len(inline.children) == 0 {
		return fmt.Errorf("link or image label is empty")
	}
	if err := validateTypedInlineDestination(inline.destination); err != nil {
		return err
	}
	if inline.hasTitle {
		if err := validateTypedInlineTitle(inline.title); err != nil {
			return err
		}
	}

	kind := KindInlineLink
	proofKind := splice.KindInlineLink
	syntaxStart := output.Len()
	if inline.kind == inlineConstructionImage {
		kind = KindImage
		proofKind = splice.KindImage
		output.WriteByte('!')
	}
	structured := !typedInlineChildrenAreText(inline.children)
	proofIndex := -1
	if structured {
		proofIndex = len(*context.hierarchy)
		*context.hierarchy = append(*context.hierarchy, splice.ConstructionInlineExpectation{
			Kind:   proofKind,
			Parent: context.parent,
		})
	}

	output.WriteByte('[')
	labelStart := output.Len()
	childContext := context
	childContext.policy = typedInlineStructuredNested
	childContext.parent = proofIndex
	for _, child := range inline.children {
		if err := writeTypedInlineConstruction(output, child, childContext); err != nil {
			return err
		}
	}
	labelEnd := output.Len()
	output.WriteString("](<")
	destinationStart := output.Len()
	output.WriteString(inline.destination)
	destinationEnd := output.Len()
	output.WriteByte('>')
	titleRange := Range{}
	if inline.hasTitle {
		output.WriteString(" \"")
		titleStart := output.Len()
		output.WriteString(inline.title)
		titleRange = Range{Start: titleStart, End: output.Len()}
		output.WriteByte('"')
	}
	output.WriteByte(')')
	if structured {
		(*context.hierarchy)[proofIndex].SyntaxRange = splice.Range{Start: syntaxStart, End: output.Len()}
		(*context.hierarchy)[proofIndex].ContentRange = splice.Range{Start: labelStart, End: labelEnd}
		*context.linkImageExpected = append(*context.linkImageExpected, splice.ConstructionLinkImageExpectation{
			Kind:        proofKind,
			SyntaxRange: splice.Range{Start: syntaxStart, End: output.Len()},
			LabelRange:  splice.Range{Start: labelStart, End: labelEnd},
			Destination: inline.destination,
			Title:       inline.title,
			HasTitle:    inline.hasTitle,
		})
		return nil
	}
	*context.expected = append(*context.expected, typedInlineExpectation{
		kind:             kind,
		contentRange:     Range{Start: destinationStart, End: destinationEnd},
		labelRange:       Range{Start: labelStart, End: labelEnd},
		titleRange:       titleRange,
		destination:      inline.destination,
		title:            inline.title,
		angleDestination: true,
		hasTitle:         inline.hasTitle,
	})
	return nil
}

func writeTypedInlineAutoLink(output *strings.Builder, value string, angle bool, expected *[]typedInlineExpectation) error {
	if err := validateTypedInlineAutoLinkValue(value); err != nil {
		return err
	}
	if angle {
		output.WriteByte('<')
	}
	contentStart := output.Len()
	output.WriteString(value)
	contentEnd := output.Len()
	if angle {
		output.WriteByte('>')
	}
	*expected = append(*expected, typedInlineExpectation{
		kind:             KindAutoLink,
		contentRange:     Range{Start: contentStart, End: contentEnd},
		destination:      value,
		angleDestination: angle,
	})
	return nil
}

func validateTypedInlineConstruction(source string, expected []typedInlineExpectation, hierarchy []splice.ConstructionInlineExpectation, linkImages []splice.ConstructionLinkImageExpectation, references []splice.ConstructionReferenceInlineExpectation) error {
	if err := splice.ValidateConstructionLinkImages([]byte(source), linkImages); err != nil {
		return fmt.Errorf("%w: typed link/image semantic proof: %v", ErrInvalidConstruction, err)
	}
	if err := splice.ValidateConstructionReferenceInlines([]byte(source), references); err != nil {
		return fmt.Errorf("%w: typed reference inline proof: %v", ErrInvalidConstruction, err)
	}
	if err := splice.ValidateConstructionInlineHierarchy([]byte(source), hierarchy, references); err != nil {
		return fmt.Errorf("%w: typed inline hierarchy proof: %v", ErrInvalidConstruction, err)
	}
	document, err := Parse([]byte(source + "\n"))
	if err != nil {
		return fmt.Errorf("%w: typed inline candidate parse: %v", ErrInvalidConstruction, err)
	}
	actual, err := collectTypedInlineCandidate(document, len(source), len(expected))
	if err != nil {
		return err
	}
	if !sameTypedInlineExpectations(actual, expected) {
		return fmt.Errorf("%w: typed inline semantic proof changed", ErrInvalidConstruction)
	}
	return nil
}

func collectTypedInlineCandidate(document *Document, sourceLength, expectedCount int) ([]typedInlineExpectation, error) {
	actual := make([]typedInlineExpectation, 0, expectedCount)
	paragraphs := 0
	for _, node := range document.Nodes() {
		if node.Kind() == KindParagraph {
			paragraph, ok := document.Paragraph(node.ID())
			if !ok || paragraph.Range() != (Range{Start: 0, End: sourceLength}) {
				return nil, fmt.Errorf("%w: typed inline candidate paragraph changed", ErrInvalidConstruction)
			}
			paragraphs++
			continue
		}
		if constructionOnlyTypedInlineNode(document, node) {
			continue
		}
		expectation, ok, err := typedInlineExpectationFromNode(document, node)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%w: typed inline candidate introduced public kind %d", ErrInvalidConstruction, node.Kind())
		}
		actual = append(actual, expectation)
	}
	if paragraphs != 1 || len(actual) != expectedCount {
		return nil, fmt.Errorf("%w: typed inline candidate structure changed", ErrInvalidConstruction)
	}
	return actual, nil
}

func constructionOnlyTypedInlineNode(document *Document, node Node) bool {
	switch node.Kind() {
	case KindEmphasis, KindStrong, KindStrikethrough:
		internal, ok := document.internalNode(node.ID())
		return ok && !internal.Editable
	default:
		return false
	}
}

func typedInlineExpectationFromNode(document *Document, node Node) (typedInlineExpectation, bool, error) {
	switch node.Kind() {
	case KindStrikethrough:
		detail, ok := document.Strikethrough(node.ID())
		if !ok {
			return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline strikethrough is not source-proven", ErrInvalidConstruction)
		}
		return typedInlineExpectation{kind: KindStrikethrough, contentRange: detail.Range()}, true, nil
	case KindCodeSpan:
		detail, ok := document.CodeSpan(node.ID())
		if !ok {
			return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline code span is not source-proven", ErrInvalidConstruction)
		}
		return typedInlineExpectation{kind: KindCodeSpan, contentRange: detail.Range()}, true, nil
	case KindEmphasis:
		detail, ok := document.Emphasis(node.ID())
		if !ok {
			return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline emphasis is not source-proven", ErrInvalidConstruction)
		}
		return typedInlineExpectation{kind: KindEmphasis, contentRange: detail.Range()}, true, nil
	case KindStrong:
		detail, ok := document.Strong(node.ID())
		if !ok {
			return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline strong emphasis is not source-proven", ErrInvalidConstruction)
		}
		return typedInlineExpectation{kind: KindStrong, contentRange: detail.Range()}, true, nil
	case KindInlineLink:
		return typedInlineLinkExpectation(document, node)
	case KindImage:
		return typedInlineImageExpectation(document, node)
	case KindAutoLink:
		return typedInlineAutoLinkExpectation(document, node)
	default:
		return typedInlineExpectation{}, false, nil
	}
}

func typedInlineLinkExpectation(document *Document, node Node) (typedInlineExpectation, bool, error) {
	detail, ok := document.InlineLink(node.ID())
	if !ok {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline link is not source-proven", ErrInvalidConstruction)
	}
	internal, ok := document.internalNode(node.ID())
	if !ok || internal.InlineLinkSource.LabelRange.Start == internal.InlineLinkSource.LabelRange.End {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline link mapping is incomplete", ErrInvalidConstruction)
	}
	return typedInlineExpectation{
		kind:             KindInlineLink,
		contentRange:     detail.Range(),
		labelRange:       publicRange(internal.InlineLinkSource.LabelRange),
		titleRange:       publicRange(internal.InlineLinkSource.TitleRange),
		destination:      internal.Destination,
		title:            internal.Title,
		angleDestination: internal.InlineLinkSource.AngleDestination,
		hasTitle:         internal.InlineLinkSource.HasTitle,
	}, true, nil
}

func typedInlineImageExpectation(document *Document, node Node) (typedInlineExpectation, bool, error) {
	detail, ok := document.Image(node.ID())
	if !ok {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline image is not source-proven", ErrInvalidConstruction)
	}
	internal, ok := document.internalNode(node.ID())
	if !ok || internal.ImageSource.AltRange.Start == internal.ImageSource.AltRange.End {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline image mapping is incomplete", ErrInvalidConstruction)
	}
	destination, ok := document.SourceRange(detail.Range())
	if !ok {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline image destination is not readable", ErrInvalidConstruction)
	}
	titleRange := publicRange(internal.ImageSource.TitleRange)
	title := ""
	if internal.ImageSource.HasTitle {
		titleSource, ok := document.SourceRange(titleRange)
		if !ok {
			return typedInlineExpectation{}, false, fmt.Errorf("%w: typed inline image title is not readable", ErrInvalidConstruction)
		}
		title = string(titleSource)
	}
	return typedInlineExpectation{
		kind:             KindImage,
		contentRange:     detail.Range(),
		labelRange:       publicRange(internal.ImageSource.AltRange),
		titleRange:       titleRange,
		destination:      string(destination),
		title:            title,
		angleDestination: internal.ImageSource.AngleDestination,
		hasTitle:         internal.ImageSource.HasTitle,
	}, true, nil
}

func typedInlineAutoLinkExpectation(document *Document, node Node) (typedInlineExpectation, bool, error) {
	detail, ok := document.AutoLink(node.ID())
	if !ok {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed autolink is not source-proven", ErrInvalidConstruction)
	}
	internal, ok := document.internalNode(node.ID())
	if !ok {
		return typedInlineExpectation{}, false, fmt.Errorf("%w: typed autolink mapping is incomplete", ErrInvalidConstruction)
	}
	return typedInlineExpectation{
		kind:             KindAutoLink,
		contentRange:     detail.Range(),
		destination:      internal.Value,
		angleDestination: internal.AutoLinkSource.Angle,
	}, true, nil
}

func publicRange(value splice.Range) Range {
	return Range{Start: value.Start, End: value.End}
}

func sameTypedInlineExpectations(actual, expected []typedInlineExpectation) bool {
	if len(actual) != len(expected) {
		return false
	}
	matched := make([]bool, len(actual))
	for _, want := range expected {
		found := false
		for index, got := range actual {
			if matched[index] || got != want {
				continue
			}
			matched[index] = true
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

func cloneTypedInlineConstruction(content []Inline) []Inline {
	return append([]Inline(nil), content...)
}

func validateTypedInlineText(text string) error {
	switch {
	case text == "":
		return fmt.Errorf("typed inline text is empty")
	case !utf8.ValidString(text):
		return fmt.Errorf("typed inline text is not valid UTF-8")
	case strings.ContainsAny(text, "\r\n"):
		return fmt.Errorf("typed inline text must stay on one physical line")
	case strings.IndexByte(text, 0) >= 0:
		return fmt.Errorf("typed inline text contains NUL")
	default:
		return nil
	}
}

func validateTypedInlineCode(code string) error {
	if err := validateTypedInlineText(code); err != nil {
		return fmt.Errorf("code span %v", err)
	}
	if isHorizontalInlineSpace(code[0]) || isHorizontalInlineSpace(code[len(code)-1]) {
		return fmt.Errorf("code span leading or trailing horizontal space requires normalization")
	}
	if code[0] == '`' || code[len(code)-1] == '`' {
		return fmt.Errorf("code span leading or trailing backtick is not source-proven")
	}
	return nil
}

func validateTypedInlineDestination(destination string) error {
	switch {
	case destination == "":
		return fmt.Errorf("link destination is empty")
	case !utf8.ValidString(destination):
		return fmt.Errorf("link destination is not valid UTF-8")
	case strings.ContainsAny(destination, "\r\n"):
		return fmt.Errorf("link destination must stay on one physical line")
	case strings.IndexByte(destination, 0) >= 0:
		return fmt.Errorf("link destination contains NUL")
	case strings.ContainsAny(destination, "<>\\&"):
		return fmt.Errorf("link destination requires GFM escaping or entity interpretation")
	default:
		return nil
	}
}

func validateTypedInlineTitle(title string) error {
	if err := validateTypedInlineText(title); err != nil {
		return fmt.Errorf("link title %v", err)
	}
	if strings.ContainsAny(title, "\"\\&") {
		return fmt.Errorf("link title requires GFM escaping or entity interpretation")
	}
	return nil
}

func validateTypedInlineAutoLinkValue(value string) error {
	if err := validateTypedInlineText(value); err != nil {
		return fmt.Errorf("autolink %v", err)
	}
	if strings.ContainsAny(value, "<>") {
		return fmt.Errorf("autolink value contains an angle bracket")
	}
	return nil
}

func typedInlineCodeFenceLength(code string) int {
	maxRun := 0
	for index := 0; index < len(code); {
		if code[index] != '`' {
			index++
			continue
		}
		end := index + 1
		for end < len(code) && code[end] == '`' {
			end++
		}
		if run := end - index; run > maxRun {
			maxRun = run
		}
		index = end
	}
	return maxRun + 1
}

func writeEscapedInlineText(output *strings.Builder, text string) {
	for index := 0; index < len(text); index++ {
		value := text[index]
		if isASCIIPunctuation(value) {
			output.WriteByte('\\')
		}
		output.WriteByte(value)
	}
}

func isASCIIPunctuation(value byte) bool {
	return value >= '!' && value <= '/' ||
		value >= ':' && value <= '@' ||
		value >= '[' && value <= '`' ||
		value >= '{' && value <= '~'
}

func isHorizontalInlineSpace(value byte) bool {
	return value == ' ' || value == '\t'
}
