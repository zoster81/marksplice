package marksplice

import "fmt"

// NodeQuery selects promoted structural nodes from one immutable document snapshot.
//
// Limit must be positive so result allocation is always caller-bounded. An empty
// Kinds slice selects every currently promoted public node kind. Within, when
// non-nil, requires the selected node's existing typed Range() span to be fully
// contained by that snapshot-local byte range.
type NodeQuery struct {
	Kinds  []Kind
	Within *Range
	Limit  int
}

// NodeMatch is one immutable structural query result.
type NodeMatch struct {
	node        Node
	sourceRange Range
}

// Node returns the promoted structural node summary.
func (m NodeMatch) Node() Node { return m.node }

// Range returns the same operation-oriented source range already exposed by the
// matched node kind's typed Range() accessor. It is a query-selection span, not
// independent mutation authority.
func (m NodeMatch) Range() Range { return m.sourceRange }

// SectionQuery selects derived document sections from one immutable snapshot.
//
// Limit must be positive. An empty Levels slice selects every heading level.
// Within, when non-nil, requires the complete Section.Range() to be fully
// contained by that snapshot-local byte range.
type SectionQuery struct {
	Levels []int
	Within *Range
	Limit  int
}

// QueryNodes returns at most query.Limit promoted nodes in existing structural
// source order. The returned slice is caller-owned and no query state is retained.
func (d *Document) QueryNodes(query NodeQuery) ([]NodeMatch, error) {
	sourceLen, err := d.querySourceLen(query.Limit)
	if err != nil {
		return nil, err
	}
	filter, err := newNodeQueryKindFilter(query.Kinds)
	if err != nil {
		return nil, err
	}
	within, hasWithin, err := queryWithinRange(sourceLen, query.Within)
	if err != nil {
		return nil, err
	}

	nodeCount := d.document.NodeCount()
	matches := make([]NodeMatch, 0, queryCapacity(query.Limit, nodeCount))
	for index := 0; index < nodeCount && len(matches) < query.Limit; index++ {
		summary, selectionRange, ok := d.document.NodeSelectionAt(index)
		if !ok {
			continue
		}
		node, ok := publicNodeSummary(summary)
		if !ok || !filter.matches(node.kind) {
			continue
		}
		selection := Range{Start: selectionRange.Start, End: selectionRange.End}
		if hasWithin && !rangeContains(within, selection) {
			continue
		}
		matches = append(matches, NodeMatch{node: node, sourceRange: selection})
	}
	return matches, nil
}

// QuerySections returns at most query.Limit derived sections in source order.
// The returned slice is caller-owned and contains the existing immutable Section
// representation rather than a second query-specific section model.
func (d *Document) QuerySections(query SectionQuery) ([]Section, error) {
	sourceLen, err := d.querySourceLen(query.Limit)
	if err != nil {
		return nil, err
	}
	filter, err := newSectionQueryLevelFilter(query.Levels)
	if err != nil {
		return nil, err
	}
	within, hasWithin, err := queryWithinRange(sourceLen, query.Within)
	if err != nil {
		return nil, err
	}

	sectionCount := d.document.SectionCount()
	matches := make([]Section, 0, queryCapacity(query.Limit, sectionCount))
	for index := 0; index < sectionCount && len(matches) < query.Limit; index++ {
		internal, ok := d.document.SectionAt(index)
		if !ok || !filter.matches(internal.Level) {
			continue
		}
		section := publicSection(internal)
		if hasWithin && !rangeContains(within, section.sourceRange) {
			continue
		}
		matches = append(matches, section)
	}
	return matches, nil
}

func (d *Document) querySourceLen(limit int) (int, error) {
	if d == nil || d.document == nil {
		return 0, fmt.Errorf("%w: nil document", ErrInvalidQuery)
	}
	if limit <= 0 {
		return 0, fmt.Errorf("%w: limit must be positive", ErrInvalidQuery)
	}
	return d.document.SourceLen(), nil
}

type nodeQueryKindFilter struct {
	all    bool
	values [KindMathExpression + 1]bool
}

func newNodeQueryKindFilter(kinds []Kind) (nodeQueryKindFilter, error) {
	if len(kinds) > int(KindMathExpression) {
		return nodeQueryKindFilter{}, fmt.Errorf("%w: too many node-kind filters: %d", ErrInvalidQuery, len(kinds))
	}
	filter := nodeQueryKindFilter{all: len(kinds) == 0}
	for _, kind := range kinds {
		if kind <= KindUnknown || kind > KindMathExpression {
			return nodeQueryKindFilter{}, fmt.Errorf("%w: unsupported node kind %d", ErrInvalidQuery, kind)
		}
		filter.values[kind] = true
	}
	return filter, nil
}

func (f nodeQueryKindFilter) matches(kind Kind) bool {
	return f.all || kind > KindUnknown && kind <= KindMathExpression && f.values[kind]
}

type sectionQueryLevelFilter struct {
	all    bool
	values [7]bool
}

func newSectionQueryLevelFilter(levels []int) (sectionQueryLevelFilter, error) {
	if len(levels) > 6 {
		return sectionQueryLevelFilter{}, fmt.Errorf("%w: too many section-level filters: %d", ErrInvalidQuery, len(levels))
	}
	filter := sectionQueryLevelFilter{all: len(levels) == 0}
	for _, level := range levels {
		if level < 1 || level > 6 {
			return sectionQueryLevelFilter{}, fmt.Errorf("%w: unsupported section level %d", ErrInvalidQuery, level)
		}
		filter.values[level] = true
	}
	return filter, nil
}

func (f sectionQueryLevelFilter) matches(level int) bool {
	return f.all || level >= 1 && level <= 6 && f.values[level]
}

func queryWithinRange(sourceLen int, within *Range) (Range, bool, error) {
	if within == nil {
		return Range{}, false, nil
	}
	result := *within
	if !result.Valid(sourceLen) {
		return Range{}, false, fmt.Errorf("%w: within range [%d,%d) outside source length %d", ErrInvalidQuery, result.Start, result.End, sourceLen)
	}
	return result, true, nil
}

func rangeContains(outer, inner Range) bool {
	return inner.Start >= outer.Start && inner.End <= outer.End
}

func queryCapacity(limit, available int) int {
	if available < limit {
		return available
	}
	return limit
}
