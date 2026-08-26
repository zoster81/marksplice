package marksplice

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ExtensionID is the caller-defined namespace of one explicitly registered third-party
// syntax/semantic extension. It is separate from the closed core Kind namespace.
type ExtensionID string

// ExtensionKind is one extension-local semantic kind name.
type ExtensionKind string

// ExtensionAttribute is one extension-defined immutable scalar metadata entry.
// Attribute names must be non-empty tokens; values must be valid UTF-8 without NUL.
type ExtensionAttribute struct {
	Name  string
	Value string
}

// ExtensionMatch is one source-owned observation returned by a third-party recognizer.
// Range must be a non-empty byte range within the exact parsed source snapshot.
type ExtensionMatch struct {
	Kind       ExtensionKind
	Range      Range
	Attributes []ExtensionAttribute
}

// ExtensionSource is the immutable source view supplied to one extension recognizer.
// Text returns the exact parsed source bytes represented as a Go string.
type ExtensionSource struct {
	text string
}

// Text returns the complete immutable source snapshot as a string.
func (s ExtensionSource) Text() string { return s.text }

// ExtensionRecognizer observes extension-specific syntax or semantics over one exact
// source snapshot. Marksplice invokes recognizers synchronously and serially during
// ParseWithOptions and never retains the callback after the call returns. Returned slices
// must not be mutated concurrently after return.
type ExtensionRecognizer func(source ExtensionSource) ([]ExtensionMatch, error)

// Extension registers one explicitly opted-in third-party recognizer under one namespace.
// Registration does not grant Marksplice filesystem, network, command, mutation, or
// construction authority. Recognizers are ordinary statically linked caller code: Marksplice
// validates their returned observations but cannot sandbox or preempt their own CPU, memory,
// goroutine, filesystem, network, or command behavior.
type Extension struct {
	ID        ExtensionID
	Recognize ExtensionRecognizer
}

// ExtensionLimits bounds extension observations retained by one ParseWithOptions call.
// Both limits must be positive when at least one extension is registered. MaxNodes is the
// total retained node count across all extensions. MaxMetadataBytes bounds the total bytes
// retained for each node's extension ID, kind, attribute names, and attribute values; it does
// not attempt to sandbox allocations performed inside third-party recognizers.
type ExtensionLimits struct {
	MaxNodes         int
	MaxMetadataBytes int
}

// ParseOptions configures optional third-party semantic/source overlays.
// Zero options are exactly equivalent to Parse.
type ParseOptions struct {
	Extensions      []Extension
	ExtensionLimits ExtensionLimits
}

// ExtensionNode is one immutable validated third-party source observation attached to a
// Document snapshot. It never replaces or reclassifies a core Node.
type ExtensionNode struct {
	extensionID ExtensionID
	kind        ExtensionKind
	sourceRange Range
	attributes  []ExtensionAttribute
}

// ExtensionID returns the namespace that produced this observation.
func (n ExtensionNode) ExtensionID() ExtensionID { return n.extensionID }

// Kind returns the extension-local semantic kind.
func (n ExtensionNode) Kind() ExtensionKind { return n.kind }

// Range returns the exact snapshot-local source range claimed by the extension.
func (n ExtensionNode) Range() Range { return n.sourceRange }

// Attributes returns caller-owned extension metadata in recognizer-provided order.
func (n ExtensionNode) Attributes() []ExtensionAttribute {
	if len(n.attributes) == 0 {
		return nil
	}
	return append([]ExtensionAttribute(nil), n.attributes...)
}

// Attribute returns the unique extension metadata value named name.
func (n ExtensionNode) Attribute(name string) (string, bool) {
	for _, attribute := range n.attributes {
		if attribute.Name == name {
			return attribute.Value, true
		}
	}
	return "", false
}

// ParseWithOptions copies and parses source using the ordinary Marksplice GFM core, then
// optionally evaluates explicitly registered third-party read-only overlays. Extension
// observations never alter core nodes, parser behavior, mutation authority, or construction.
func ParseWithOptions(source []byte, options ParseOptions) (*Document, error) {
	if err := validateExtensionOptions(options); err != nil {
		return nil, err
	}
	document, err := Parse(source)
	if err != nil {
		return nil, err
	}
	if len(options.Extensions) == 0 {
		return document, nil
	}
	nodes, err := buildExtensionNodes(string(source), options.Extensions, options.ExtensionLimits)
	if err != nil {
		return nil, err
	}
	document.extensionNodes = nodes
	return document, nil
}

// ExtensionNodes returns caller-owned immutable extension observations in registration
// order and each recognizer's returned order. Core structural nodes remain separate.
func (d *Document) ExtensionNodes() []ExtensionNode {
	if d == nil || len(d.extensionNodes) == 0 {
		return nil
	}
	return append([]ExtensionNode(nil), d.extensionNodes...)
}

func validateExtensionOptions(options ParseOptions) error {
	if len(options.Extensions) == 0 {
		return nil
	}
	if options.ExtensionLimits.MaxNodes <= 0 {
		return fmt.Errorf("%w: extension node limit must be positive", ErrInvalidExtension)
	}
	if options.ExtensionLimits.MaxMetadataBytes <= 0 {
		return fmt.Errorf("%w: extension metadata-byte limit must be positive", ErrInvalidExtension)
	}
	seen := make(map[ExtensionID]struct{}, len(options.Extensions))
	for position, extension := range options.Extensions {
		if !validExtensionToken(string(extension.ID)) {
			return fmt.Errorf("%w: invalid extension id at position %d", ErrInvalidExtension, position)
		}
		if extension.Recognize == nil {
			return fmt.Errorf("%w: extension %q has nil recognizer", ErrInvalidExtension, extension.ID)
		}
		if _, exists := seen[extension.ID]; exists {
			return fmt.Errorf("%w: duplicate extension id %q", ErrInvalidExtension, extension.ID)
		}
		seen[extension.ID] = struct{}{}
	}
	return nil
}

func buildExtensionNodes(sourceText string, extensions []Extension, limits ExtensionLimits) ([]ExtensionNode, error) {
	source := ExtensionSource{text: sourceText}
	result := make([]ExtensionNode, 0)
	metadataBytes := 0
	for _, extension := range extensions {
		matches, err := callExtensionRecognizer(extension, source)
		if err != nil {
			return nil, err
		}
		if len(matches) > limits.MaxNodes-len(result) {
			return nil, fmt.Errorf("%w: extension node limit exceeded", ErrInvalidExtension)
		}
		for position, match := range matches {
			node, retainedBytes, err := validateExtensionMatch(extension.ID, match, len(source.text), position, limits.MaxMetadataBytes-metadataBytes)
			if err != nil {
				return nil, err
			}
			metadataBytes += retainedBytes
			result = append(result, node)
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func callExtensionRecognizer(extension Extension, source ExtensionSource) (matches []ExtensionMatch, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recoveredErr, ok := recovered.(error); ok {
				err = fmt.Errorf("%w: extension %q recognizer panicked: %w", ErrInvalidExtension, extension.ID, recoveredErr)
				return
			}
			err = fmt.Errorf("%w: extension %q recognizer panicked: %v", ErrInvalidExtension, extension.ID, recovered)
		}
	}()
	matches, err = extension.Recognize(source)
	if err != nil {
		return nil, fmt.Errorf("%w: extension %q recognizer failed: %w", ErrInvalidExtension, extension.ID, err)
	}
	return matches, nil
}

func validateExtensionMatch(extensionID ExtensionID, match ExtensionMatch, sourceLen, position, metadataBudget int) (ExtensionNode, int, error) {
	if !validExtensionToken(string(match.Kind)) {
		return ExtensionNode{}, 0, fmt.Errorf("%w: extension %q returned invalid kind at position %d", ErrInvalidExtension, extensionID, position)
	}
	if !match.Range.Valid(sourceLen) || match.Range.Start == match.Range.End {
		return ExtensionNode{}, 0, fmt.Errorf("%w: extension %q returned invalid range [%d,%d) at position %d", ErrInvalidExtension, extensionID, match.Range.Start, match.Range.End, position)
	}
	baseBytes, ok := extensionMetadataSize(metadataBudget, string(extensionID), string(match.Kind))
	if !ok {
		return ExtensionNode{}, 0, fmt.Errorf("%w: extension metadata-byte limit exceeded", ErrInvalidExtension)
	}
	attributes, attributeBytes, err := cloneExtensionAttributes(extensionID, match.Attributes, position, metadataBudget-baseBytes)
	if err != nil {
		return ExtensionNode{}, 0, err
	}
	return ExtensionNode{
		extensionID: extensionID,
		kind:        match.Kind,
		sourceRange: match.Range,
		attributes:  attributes,
	}, baseBytes + attributeBytes, nil
}

func cloneExtensionAttributes(extensionID ExtensionID, attributes []ExtensionAttribute, position, metadataBudget int) ([]ExtensionAttribute, int, error) {
	if len(attributes) == 0 {
		return nil, 0, nil
	}
	result := make([]ExtensionAttribute, len(attributes))
	seen := make(map[string]struct{}, len(attributes))
	retainedBytes := 0
	for index, attribute := range attributes {
		if !validExtensionToken(attribute.Name) {
			return nil, 0, fmt.Errorf("%w: extension %q returned invalid attribute name at node %d attribute %d", ErrInvalidExtension, extensionID, position, index)
		}
		if !utf8.ValidString(attribute.Value) || strings.ContainsRune(attribute.Value, '\x00') {
			return nil, 0, fmt.Errorf("%w: extension %q returned invalid attribute value at node %d attribute %d", ErrInvalidExtension, extensionID, position, index)
		}
		if _, exists := seen[attribute.Name]; exists {
			return nil, 0, fmt.Errorf("%w: extension %q returned duplicate attribute %q at node %d", ErrInvalidExtension, extensionID, attribute.Name, position)
		}
		attributeBytes, ok := extensionMetadataSize(metadataBudget-retainedBytes, attribute.Name, attribute.Value)
		if !ok {
			return nil, 0, fmt.Errorf("%w: extension metadata-byte limit exceeded", ErrInvalidExtension)
		}
		seen[attribute.Name] = struct{}{}
		result[index] = attribute
		retainedBytes += attributeBytes
	}
	return result, retainedBytes, nil
}

func extensionMetadataSize(limit int, values ...string) (int, bool) {
	if limit < 0 {
		return 0, false
	}
	used := 0
	for _, value := range values {
		if len(value) > limit-used {
			return 0, false
		}
		used += len(value)
	}
	return used, true
}

func validExtensionToken(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}
