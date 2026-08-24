package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// AlertKind identifies one reviewed GitHub alert semantic kind.
type AlertKind uint8

const (
	AlertKindUnknown AlertKind = iota
	AlertKindNote
	AlertKindTip
	AlertKindImportant
	AlertKindWarning
	AlertKindCaution
)

// Alert is immutable semantic detail layered over one promoted top-level blockquote.
// Its ID is the underlying blockquote NodeID; alerts do not introduce a second identity namespace.
type Alert struct {
	id          NodeID
	kind        AlertKind
	sourceRange Range
	markerRange Range
}

// ID returns the underlying blockquote's snapshot-scoped identity.
func (a Alert) ID() NodeID { return a.id }

// Kind returns the exact reviewed GitHub alert kind.
func (a Alert) Kind() AlertKind { return a.kind }

// Range returns the exact complete physical source owned by the underlying top-level blockquote.
func (a Alert) Range() Range { return a.sourceRange }

// MarkerRange returns the exact inner-source range containing the alert marker such as [!NOTE].
func (a Alert) MarkerRange() Range { return a.markerRange }

// Alert returns semantic alert detail when id identifies a promoted top-level blockquote
// whose first inner physical line is one exact reviewed GitHub alert marker and whose
// remaining owned source contains at least one non-empty body segment.
func (d *Document) Alert(id NodeID) (Alert, bool) {
	node, err := d.promotedNode(id, splice.KindBlockquote, true)
	if err != nil {
		return Alert{}, false
	}
	return d.alertFromBlockquoteNode(node)
}

// Alerts returns all recognized top-level GitHub alerts in source order.
// The returned slice is caller-owned. Recognition adds no persistent semantic index.
func (d *Document) Alerts() []Alert {
	if d == nil || d.document == nil {
		return nil
	}
	alerts := make([]Alert, 0)
	for index := 0; index < d.document.NodeCount(); index++ {
		summary, ok := d.document.NodeSummaryAt(index)
		if !ok || summary.Kind != splice.KindBlockquote || !summary.TopLevel || !summary.Editable {
			continue
		}
		node, ok := d.document.Node(summary.ID)
		if !ok {
			continue
		}
		alert, ok := d.alertFromBlockquoteNode(node)
		if ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// AlertBodyRanges returns caller-owned inner source segments after the alert marker line.
// Marker-only blank lines are represented by valid empty ranges and lazy continuation
// lines retain their source-proven M94 inner ranges.
func (d *Document) AlertBodyRanges(id NodeID) ([]Range, bool) {
	node, err := d.promotedNode(id, splice.KindBlockquote, true)
	if err != nil {
		return nil, false
	}
	if _, ok := d.alertFromBlockquoteNode(node); !ok {
		return nil, false
	}
	return publicRanges(node.BlockquoteSource.ContentRanges[1:]), true
}

func (d *Document) alertFromBlockquoteNode(node splice.Node) (Alert, bool) {
	ranges := node.BlockquoteSource.ContentRanges
	if len(ranges) < 2 {
		return Alert{}, false
	}
	markerRange := Range{Start: ranges[0].Start, End: ranges[0].End}
	marker, ok := d.SourceRange(markerRange)
	if !ok {
		return Alert{}, false
	}
	kind := alertKindFromMarker(marker)
	if kind == AlertKindUnknown || !alertHasBody(ranges[1:]) {
		return Alert{}, false
	}
	mapping := node.BlockquoteSource
	return Alert{
		id:          publicNodeID(node.ID),
		kind:        kind,
		sourceRange: Range{Start: mapping.LineRange.Start, End: mapping.LineRange.End},
		markerRange: markerRange,
	}, true
}

func alertHasBody(ranges []splice.Range) bool {
	for _, range_ := range ranges {
		if range_.Start < range_.End {
			return true
		}
	}
	return false
}

func alertKindFromMarker(marker []byte) AlertKind {
	switch string(marker) {
	case "[!NOTE]":
		return AlertKindNote
	case "[!TIP]":
		return AlertKindTip
	case "[!IMPORTANT]":
		return AlertKindImportant
	case "[!WARNING]":
		return AlertKindWarning
	case "[!CAUTION]":
		return AlertKindCaution
	default:
		return AlertKindUnknown
	}
}

func alertMarker(kind AlertKind) (string, bool) {
	switch kind {
	case AlertKindNote:
		return "[!NOTE]", true
	case AlertKindTip:
		return "[!TIP]", true
	case AlertKindImportant:
		return "[!IMPORTANT]", true
	case AlertKindWarning:
		return "[!WARNING]", true
	case AlertKindCaution:
		return "[!CAUTION]", true
	default:
		return "", false
	}
}
