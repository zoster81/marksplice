package goldmark

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/zoster81/marksplice/internal/parser"
)

// ValidateConstructionInlineHierarchy proves typed construction-only semantic
// nesting without widening the observations returned by Adapter.Parse.
func ValidateConstructionInlineHierarchy(source []byte, expected []parser.ConstructionInlineExpectation, references []parser.ConstructionReferenceInlineExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateConstructionInlineExpectationInput(source, want, index); err != nil {
			return err
		}
	}
	proofSource, err := constructionInlineProofSource(source, references)
	if err != nil {
		return err
	}
	root := newMarkdown().Parser().Parse(text.NewReader(normalizeIsolatedCR(proofSource)))
	paragraph, err := constructionInlineProofParagraph(proofSource, len(source), root)
	if err != nil {
		return err
	}
	actual, err := matchConstructionInlineNodes(paragraph, expected)
	if err != nil {
		return err
	}
	for index, want := range expected {
		if err := validateConstructionInlineNode(source, want, actual[index]); err != nil {
			return fmt.Errorf("typed inline hierarchy node %d: %w", index, err)
		}
		parent := ast.Node(paragraph)
		if want.Parent >= 0 {
			parent = actual[want.Parent]
		}
		if actual[index].Parent() != parent {
			return fmt.Errorf("typed inline hierarchy parent %d changed", index)
		}
	}
	rootChildren, childrenByParent := constructionInlineExpectedChildren(expected)
	actualIndex := constructionInlineActualIndexes(actual)
	if err := validateConstructionInlineRootChildren(paragraph, rootChildren, actual, actualIndex); err != nil {
		return err
	}
	for index := range expected {
		if err := validateConstructionInlineChildren(actual[index], index, childrenByParent[index], actual, actualIndex); err != nil {
			return err
		}
	}
	return nil
}

func constructionInlineProofSource(source []byte, references []parser.ConstructionReferenceInlineExpectation) ([]byte, error) {
	if len(references) == 0 {
		return source, nil
	}
	return constructionReferenceProofSource(source, references)
}

func constructionInlineProofParagraph(source []byte, originalLength int, root ast.Node) (*ast.Paragraph, error) {
	paragraph, ok := root.FirstChild().(*ast.Paragraph)
	if !ok || paragraph == nil {
		return nil, fmt.Errorf("typed inline hierarchy is not contained by a paragraph")
	}
	lines := paragraph.Lines()
	if lines.Len() != 1 {
		return nil, fmt.Errorf("typed inline hierarchy paragraph line count changed")
	}
	line := lines.At(0)
	if line.Start != 0 || paragraphContentEnd(source, line.Stop) != originalLength {
		return nil, fmt.Errorf("typed inline hierarchy paragraph range changed")
	}
	return paragraph, nil
}

func validateConstructionInlineExpectationInput(source []byte, want parser.ConstructionInlineExpectation, index int) error {
	switch want.Kind {
	case parser.KindInlineLink, parser.KindImage:
		return validateConstructionInlineOwnerInput(source, want, index)
	default:
		return validateConstructionDelimitedInlineInput(source, want, index)
	}
}

func validateConstructionDelimitedInlineInput(source []byte, want parser.ConstructionInlineExpectation, index int) error {
	if !want.SyntaxRange.Valid(len(source)) || !want.ContentRange.Valid(len(source)) || want.ContentRange.Start == want.ContentRange.End ||
		want.DelimiterLength < 1 || want.SyntaxRange.Start+want.DelimiterLength != want.ContentRange.Start ||
		want.ContentRange.End+want.DelimiterLength != want.SyntaxRange.End || want.Parent < -1 || want.Parent >= index {
		return fmt.Errorf("invalid typed inline hierarchy expectation %d", index)
	}
	if want.Marker == 0 {
		return fmt.Errorf("typed inline hierarchy marker %d is empty", index)
	}
	for offset := 0; offset < want.DelimiterLength; offset++ {
		if source[want.SyntaxRange.Start+offset] != want.Marker || source[want.ContentRange.End+offset] != want.Marker {
			return fmt.Errorf("typed inline hierarchy delimiter %d changed", index)
		}
	}
	return nil
}

func validateConstructionInlineOwnerInput(source []byte, want parser.ConstructionInlineExpectation, index int) error {
	if !want.SyntaxRange.Valid(len(source)) || !want.ContentRange.Valid(len(source)) || want.ContentRange.Start == want.ContentRange.End ||
		want.SyntaxRange.Start == want.SyntaxRange.End || want.Parent != -1 || want.Marker != 0 || want.DelimiterLength != 0 {
		return fmt.Errorf("invalid typed inline owner expectation %d", index)
	}
	prefixLength := 1
	if want.Kind == parser.KindImage {
		prefixLength = 2
		if source[want.SyntaxRange.Start] != '!' {
			return fmt.Errorf("typed inline owner image prefix %d changed", index)
		}
	}
	if want.ContentRange.Start != want.SyntaxRange.Start+prefixLength || source[want.ContentRange.Start-1] != '[' ||
		want.ContentRange.End >= want.SyntaxRange.End || source[want.ContentRange.End] != ']' {
		return fmt.Errorf("typed inline owner boundary %d changed", index)
	}
	return nil
}

func matchConstructionInlineNodes(paragraph *ast.Paragraph, expected []parser.ConstructionInlineExpectation) ([]ast.Node, error) {
	byStart := make(map[int][]int, len(expected))
	for index, want := range expected {
		byStart[want.SyntaxRange.Start] = append(byStart[want.SyntaxRange.Start], index)
	}
	actual := make([]ast.Node, len(expected))
	if err := ast.Walk(paragraph, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == paragraph {
			return ast.WalkContinue, nil
		}
		for _, index := range byStart[node.Pos()] {
			if actual[index] == nil && constructionInlineNodeMatchesKind(node, expected[index].Kind) {
				actual[index] = node
				break
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return nil, fmt.Errorf("walk typed inline hierarchy: %w", err)
	}
	for index, node := range actual {
		if node == nil {
			return nil, fmt.Errorf("typed inline hierarchy node %d changed", index)
		}
	}
	return actual, nil
}

func constructionInlineNodeMatchesKind(node ast.Node, kind parser.Kind) bool {
	switch want := kind; want {
	case parser.KindCodeSpan:
		_, ok := node.(*ast.CodeSpan)
		return ok
	case parser.KindEmphasis, parser.KindStrong:
		emphasis, ok := node.(*ast.Emphasis)
		if !ok {
			return false
		}
		level := 1
		if want == parser.KindStrong {
			level = 2
		}
		return emphasis.Level == level
	case parser.KindStrikethrough:
		_, ok := node.(*extensionast.Strikethrough)
		return ok
	case parser.KindInlineLink:
		_, ok := node.(*ast.Link)
		return ok
	case parser.KindImage:
		_, ok := node.(*ast.Image)
		return ok
	default:
		return false
	}
}

func validateConstructionInlineNode(source []byte, want parser.ConstructionInlineExpectation, node ast.Node) error {
	if node.Pos() != want.SyntaxRange.Start || !constructionInlineNodeMatchesKind(node, want.Kind) {
		return fmt.Errorf("semantic kind or anchor changed")
	}
	switch want.Kind {
	case parser.KindCodeSpan:
		return validateConstructionCodeSpan(source, want, node)
	case parser.KindEmphasis:
		return validateConstructionEmphasisDelimiter(want, 1, "emphasis")
	case parser.KindStrong:
		return validateConstructionEmphasisDelimiter(want, 2, "strong")
	case parser.KindStrikethrough:
		return validateConstructionStrikethroughDelimiter(want)
	case parser.KindInlineLink, parser.KindImage:
		return nil
	default:
		return fmt.Errorf("unsupported typed inline hierarchy kind %d", want.Kind)
	}
}

func validateConstructionCodeSpan(source []byte, want parser.ConstructionInlineExpectation, node ast.Node) error {
	span, ok := node.(*ast.CodeSpan)
	if !ok || want.Marker != '`' {
		return fmt.Errorf("code-span shape changed")
	}
	content, ok := simplePlainTextInlineRange(source, span)
	if !ok || content != want.ContentRange {
		return fmt.Errorf("code-span content range changed")
	}
	return nil
}

func validateConstructionEmphasisDelimiter(want parser.ConstructionInlineExpectation, delimiterLength int, description string) error {
	if want.DelimiterLength != delimiterLength || want.Marker != '*' && want.Marker != '_' {
		return fmt.Errorf("%s delimiter changed", description)
	}
	return nil
}

func validateConstructionStrikethroughDelimiter(want parser.ConstructionInlineExpectation) error {
	if want.DelimiterLength != 2 || want.Marker != '~' {
		return fmt.Errorf("strikethrough delimiter changed")
	}
	return nil
}

func constructionInlineExpectedChildren(expected []parser.ConstructionInlineExpectation) ([]int, [][]int) {
	root := make([]int, 0, len(expected))
	children := make([][]int, len(expected))
	for index, want := range expected {
		if want.Parent == -1 {
			root = append(root, index)
			continue
		}
		children[want.Parent] = append(children[want.Parent], index)
	}
	return root, children
}

func constructionInlineActualIndexes(actual []ast.Node) map[ast.Node]int {
	result := make(map[ast.Node]int, len(actual))
	for index, node := range actual {
		result[node] = index
	}
	return result
}

func validateConstructionInlineRootChildren(paragraph *ast.Paragraph, wantChildren []int, actual []ast.Node, actualIndex map[ast.Node]int) error {
	gotChildren := make([]ast.Node, 0, len(wantChildren))
	for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.Text); ok {
			continue
		}
		if _, ok := actualIndex[child]; ok {
			gotChildren = append(gotChildren, child)
			continue
		}
		if constructionInlineNestedSemanticNode(child) {
			return fmt.Errorf("typed inline hierarchy root gained unexpected child %s", child.Kind())
		}
	}
	if len(gotChildren) != len(wantChildren) {
		return fmt.Errorf("typed inline hierarchy root child count changed")
	}
	for index, expectedIndex := range wantChildren {
		if gotChildren[index] != actual[expectedIndex] {
			return fmt.Errorf("typed inline hierarchy root child order changed")
		}
	}
	return nil
}

func validateConstructionInlineChildren(node ast.Node, parent int, wantChildren []int, actual []ast.Node, actualIndex map[ast.Node]int) error {
	gotChildren := make([]ast.Node, 0, len(wantChildren))
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.Text); ok {
			continue
		}
		if _, ok := actualIndex[child]; ok {
			gotChildren = append(gotChildren, child)
			continue
		}
		return fmt.Errorf("typed inline hierarchy parent %d gained unsupported child %s", parent, child.Kind())
	}
	if len(gotChildren) != len(wantChildren) {
		return fmt.Errorf("typed inline hierarchy child count %d changed", parent)
	}
	for index, expectedIndex := range wantChildren {
		if gotChildren[index] != actual[expectedIndex] {
			return fmt.Errorf("typed inline hierarchy child order %d changed", parent)
		}
	}
	return nil
}

func constructionInlineNestedSemanticNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.CodeSpan, *ast.Emphasis, *extensionast.Strikethrough:
		return true
	default:
		return false
	}
}
