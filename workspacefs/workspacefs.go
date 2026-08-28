// Package workspacefs loads caller-authorized Markdown workspaces from fs.FS.
//
// The package is read-only. It performs no network access, command execution, or
// filesystem writes. The supplied fs.FS defines the caller's filesystem authority.
package workspacefs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/zoster81/marksplice"
)

var (
	// ErrInvalidInput classifies malformed roots, entries, options, or nil workspaces.
	ErrInvalidInput = errors.New("invalid filesystem workspace input")
	// ErrBudgetExceeded classifies document, byte, depth, or relationship limit exhaustion.
	ErrBudgetExceeded = errors.New("filesystem workspace budget exceeded")
)

const (
	DefaultMaxDocuments           = 10_000
	DefaultMaxBytes         int64 = 256 << 20
	DefaultMaxDepth               = 64
	DefaultMaxRelationships       = 250_000
)

// Limits bounds one filesystem workspace operation. MaxDepth means directory
// depth below root for Scan and relationship-hop depth for Follow.
type Limits struct {
	MaxDocuments     int
	MaxBytes         int64
	MaxDepth         int
	MaxRelationships int
}

// Options configures filesystem workspace loading.
type Options struct {
	Limits Limits
}

// DefaultOptions returns conservative finite limits suitable for ordinary
// documentation workspaces. Callers with different resource envelopes should
// supply explicit limits instead.
func DefaultOptions() Options {
	return Options{Limits: Limits{
		MaxDocuments:     DefaultMaxDocuments,
		MaxBytes:         DefaultMaxBytes,
		MaxDepth:         DefaultMaxDepth,
		MaxRelationships: DefaultMaxRelationships,
	}}
}

// Workspace is an immutable set of parsed documents loaded from an fs.FS.
type Workspace struct {
	documents []marksplice.GraphDocument
	keys      map[marksplice.DocumentKey]struct{}
}

// Scan discovers every .md or .markdown document under root in deterministic
// slash-relative order and parses it with marksplice.Parse.
func Scan(fsys fs.FS, root string, options Options) (*Workspace, error) {
	scoped, limits, err := prepare(fsys, root, options)
	if err != nil {
		return nil, err
	}
	loader := workspaceLoader{fsys: scoped, limits: limits}
	err = fs.WalkDir(scoped, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		return loader.scanEntry(name, entry, walkErr)
	})
	if err != nil {
		return nil, err
	}
	return newWorkspace(loader.documents), nil
}

// Follow loads explicit entry documents and follows reviewed local Markdown
// URI-path relationships. Relative dot segments are normalized inside the supplied
// fs.FS namespace, path components are percent-decoded once, query text is ignored
// for file lookup, fragments are preserved, and cycles are visited once. Absolute,
// scheme/protocol-relative, backslash, encoded-traversal/separator, directory, and
// extensionless targets are not treated as filesystem documents.
func Follow(fsys fs.FS, root string, entries []string, options Options) (*Workspace, error) {
	scoped, limits, err := prepare(fsys, root, options)
	if err != nil {
		return nil, err
	}
	queue, discovered, err := initialQueue(entries)
	if err != nil {
		return nil, err
	}
	loader := workspaceLoader{fsys: scoped, limits: limits}
	availability := make(map[marksplice.DocumentKey]bool)
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		relationships, loadErr := loader.addDocument(string(current.key))
		if loadErr != nil {
			return nil, loadErr
		}
		if err := followRelationships(scoped, relationships, current, limits.MaxDepth, discovered, availability, &queue); err != nil {
			return nil, err
		}
	}
	return newWorkspace(loader.documents), nil
}

// Documents returns a caller-owned copy of the graph inputs in deterministic
// workspace order. Parsed Document values themselves are immutable.
func (w *Workspace) Documents() []marksplice.GraphDocument {
	if w == nil || len(w.documents) == 0 {
		return nil
	}
	result := make([]marksplice.GraphDocument, len(w.documents))
	copy(result, w.documents)
	return result
}

// BuildGraph builds Marksplice's existing immutable DocumentGraph using the
// workspace's reviewed local relationship resolver.
func (w *Workspace) BuildGraph() (*marksplice.DocumentGraph, error) {
	if w == nil {
		return nil, invalidInputf("nil workspace")
	}
	return marksplice.BuildDocumentGraph(w.documents, w.graphResolver())
}

// Validate runs Marksplice's existing workspace validator with filesystem-local
// resolved/missing relationship facts from this workspace.
func (w *Workspace) Validate(options marksplice.WorkspaceValidationOptions) (*marksplice.WorkspaceReport, error) {
	if w == nil {
		return nil, invalidInputf("nil workspace")
	}
	return marksplice.ValidateWorkspace(w.documents, w.workspaceResolver(), options)
}

type workspaceLoader struct {
	fsys          fs.FS
	limits        Limits
	documents     []marksplice.GraphDocument
	bytes         int64
	relationships int
}

func (l *workspaceLoader) scanEntry(name string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return fmt.Errorf("workspacefs: walk %q: %w", name, walkErr)
	}
	if entry.IsDir() {
		if name != "." && directoryDepth(name) > l.limits.MaxDepth {
			return budgetExceededf("depth", int64(directoryDepth(name)), int64(l.limits.MaxDepth))
		}
		return nil
	}
	if !isMarkdownPath(name) {
		return nil
	}
	if documentDepth(name) > l.limits.MaxDepth {
		return budgetExceededf("depth", int64(documentDepth(name)), int64(l.limits.MaxDepth))
	}
	_, err := l.addDocument(name)
	return err
}

func (l *workspaceLoader) addDocument(name string) ([]marksplice.LinkRelationship, error) {
	if len(l.documents) >= l.limits.MaxDocuments {
		return nil, budgetExceededf("documents", int64(len(l.documents)+1), int64(l.limits.MaxDocuments))
	}
	source, err := readBounded(l.fsys, name, l.bytes, l.limits.MaxBytes)
	if err != nil {
		return nil, err
	}
	l.bytes += int64(len(source))
	document, err := marksplice.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("workspacefs: parse %q: %w", name, err)
	}
	relationships := document.LinkRelationships()
	remaining := l.limits.MaxRelationships - l.relationships
	if len(relationships) > remaining {
		return nil, budgetExceededf("relationships", int64(l.relationships+len(relationships)), int64(l.limits.MaxRelationships))
	}
	l.relationships += len(relationships)
	l.documents = append(l.documents, marksplice.GraphDocument{Key: marksplice.DocumentKey(name), Document: document})
	return relationships, nil
}

type followItem struct {
	key   marksplice.DocumentKey
	depth int
}

func initialQueue(entries []string) ([]followItem, map[marksplice.DocumentKey]bool, error) {
	if len(entries) == 0 {
		return nil, nil, invalidInputf("no entry documents")
	}
	unique := make(map[string]struct{}, len(entries))
	ordered := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !fs.ValidPath(entry) || entry == "." || !isMarkdownPath(entry) {
			return nil, nil, invalidInputf("invalid entry %q", entry)
		}
		if _, exists := unique[entry]; exists {
			continue
		}
		unique[entry] = struct{}{}
		ordered = append(ordered, entry)
	}
	sort.Strings(ordered)
	queue := make([]followItem, len(ordered))
	discovered := make(map[marksplice.DocumentKey]bool, len(ordered))
	for index, entry := range ordered {
		key := marksplice.DocumentKey(entry)
		queue[index] = followItem{key: key}
		discovered[key] = true
	}
	return queue, discovered, nil
}

func followRelationships(fsys fs.FS, relationships []marksplice.LinkRelationship, current followItem, maxDepth int, discovered, availability map[marksplice.DocumentKey]bool, queue *[]followItem) error {
	for _, relationship := range relationships {
		target, _, local := localTarget(current.key, relationship)
		if !local || discovered[target] {
			continue
		}
		exists, err := targetExists(fsys, target, availability)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		nextDepth := current.depth + 1
		if nextDepth > maxDepth {
			return budgetExceededf("depth", int64(nextDepth), int64(maxDepth))
		}
		discovered[target] = true
		*queue = append(*queue, followItem{key: target, depth: nextDepth})
	}
	return nil
}

func targetExists(fsys fs.FS, target marksplice.DocumentKey, availability map[marksplice.DocumentKey]bool) (bool, error) {
	if exists, checked := availability[target]; checked {
		return exists, nil
	}
	info, err := fs.Stat(fsys, string(target))
	if errors.Is(err, fs.ErrNotExist) {
		availability[target] = false
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("workspacefs: stat %q: %w", target, err)
	}
	exists := !info.IsDir()
	availability[target] = exists
	return exists, nil
}

func prepare(fsys fs.FS, root string, options Options) (fs.FS, Limits, error) {
	if fsys == nil {
		return nil, Limits{}, invalidInputf("nil filesystem")
	}
	if !fs.ValidPath(root) {
		return nil, Limits{}, invalidInputf("invalid root %q", root)
	}
	if err := validateLimits(options.Limits); err != nil {
		return nil, Limits{}, err
	}
	if root == "." {
		return fsys, options.Limits, nil
	}
	scoped, err := fs.Sub(fsys, root)
	if err != nil {
		return nil, Limits{}, fmt.Errorf("workspacefs: root %q: %w", root, err)
	}
	return scoped, options.Limits, nil
}

func validateLimits(limits Limits) error {
	if limits.MaxDocuments <= 0 {
		return invalidInputf("MaxDocuments must be positive")
	}
	if limits.MaxBytes <= 0 {
		return invalidInputf("MaxBytes must be positive")
	}
	if limits.MaxDepth < 0 {
		return invalidInputf("MaxDepth must not be negative")
	}
	if limits.MaxRelationships <= 0 {
		return invalidInputf("MaxRelationships must be positive")
	}
	return nil
}

func readBounded(fsys fs.FS, name string, used, maximum int64) ([]byte, error) {
	remaining := maximum - used
	if remaining < 0 {
		return nil, fmt.Errorf("%w: bytes exceed %d", ErrBudgetExceeded, maximum)
	}
	file, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("workspacefs: open %q: %w", name, err)
	}
	limit := remaining
	if limit < math.MaxInt64 {
		limit++
	}
	data, readErr := io.ReadAll(io.LimitReader(file, limit))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("workspacefs: read %q: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("workspacefs: close %q: %w", name, closeErr)
	}
	if int64(len(data)) > remaining {
		return nil, fmt.Errorf("%w: bytes exceed %d", ErrBudgetExceeded, maximum)
	}
	return data, nil
}

func newWorkspace(documents []marksplice.GraphDocument) *Workspace {
	keys := make(map[marksplice.DocumentKey]struct{}, len(documents))
	for _, document := range documents {
		keys[document.Key] = struct{}{}
	}
	return &Workspace{documents: documents, keys: keys}
}

func (w *Workspace) graphResolver() marksplice.DocumentResolver {
	return func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		target, fragment, local := localTarget(source, relationship)
		if !local {
			return marksplice.DocumentResolution{}, false
		}
		if _, exists := w.keys[target]; !exists {
			return marksplice.DocumentResolution{}, false
		}
		return marksplice.DocumentResolution{Target: target, Fragment: fragment}, true
	}
}

func (w *Workspace) workspaceResolver() marksplice.WorkspaceResolver {
	return func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		target, fragment, local := localTarget(source, relationship)
		if !local {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
		}
		if _, exists := w.keys[target]; exists {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: target, Fragment: fragment}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionMissing, Target: target, Fragment: fragment}
	}
}

func localTarget(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentKey, string, bool) {
	if relationship.IsEmail() {
		return "", "", false
	}
	rawPath, fragment := splitLocalDestination(relationship.Destination())
	decodedPath, ok := decodeLocalPath(rawPath)
	if !ok {
		return "", "", false
	}
	targetText := path.Clean(path.Join(path.Dir(string(source)), decodedPath))
	if !fs.ValidPath(targetText) || targetText == "." || !isMarkdownPath(targetText) {
		return "", "", false
	}
	return marksplice.DocumentKey(targetText), fragment, true
}

func splitLocalDestination(destination string) (string, string) {
	pathAndQuery := destination
	fragment := ""
	if index := strings.IndexByte(pathAndQuery, '#'); index >= 0 {
		fragment = pathAndQuery[index:]
		pathAndQuery = pathAndQuery[:index]
	}
	if index := strings.IndexByte(pathAndQuery, '?'); index >= 0 {
		pathAndQuery = pathAndQuery[:index]
	}
	return pathAndQuery, fragment
}

func decodeLocalPath(rawPath string) (string, bool) {
	if rawPath == "" || rawPath[0] == '/' || strings.HasSuffix(rawPath, "/") || strings.Contains(rawPath, "//") || strings.Contains(rawPath, `\`) || hasURIScheme(rawPath) {
		return "", false
	}
	if !strings.Contains(rawPath, "%") {
		return rawPath, true
	}
	var decoded strings.Builder
	decoded.Grow(len(rawPath))
	remaining := rawPath
	first := true
	for {
		segment, rest, found := strings.Cut(remaining, "/")
		value, err := url.PathUnescape(segment)
		if err != nil || !validDecodedPathSegment(segment, value) {
			return "", false
		}
		if !first {
			decoded.WriteByte('/')
		}
		decoded.WriteString(value)
		if !found {
			break
		}
		first = false
		remaining = rest
	}
	return decoded.String(), true
}

func validDecodedPathSegment(raw, decoded string) bool {
	if decoded == "" || strings.ContainsAny(decoded, `/\`) || strings.IndexByte(decoded, 0) >= 0 {
		return false
	}
	if (decoded == "." || decoded == "..") && raw != decoded {
		return false
	}
	return true
}

func hasURIScheme(value string) bool {
	colon := strings.IndexByte(value, ':')
	if colon <= 0 {
		return false
	}
	if slash := strings.IndexByte(value, '/'); slash >= 0 && slash < colon {
		return false
	}
	for index := 0; index < colon; index++ {
		if !uriSchemeByte(value[index], index == 0) {
			return false
		}
	}
	return true
}

func uriSchemeByte(value byte, first bool) bool {
	if value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' {
		return true
	}
	if first {
		return false
	}
	return value >= '0' && value <= '9' || value == '+' || value == '-' || value == '.'
}

func isMarkdownPath(name string) bool {
	extension := path.Ext(name)
	return strings.EqualFold(extension, ".md") || strings.EqualFold(extension, ".markdown")
}

func directoryDepth(name string) int {
	return strings.Count(name, "/") + 1
}

func documentDepth(name string) int {
	return strings.Count(name, "/")
}

func invalidInputf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}

func budgetExceededf(name string, value, maximum int64) error {
	return fmt.Errorf("%w: %s %d exceeds %d", ErrBudgetExceeded, name, value, maximum)
}
