package marksplice

import (
	"github.com/zoster81/marksplice/internal/source"
	"github.com/zoster81/marksplice/internal/splice"
)

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
	if d == nil || d.document == nil {
		return Alert{}, false
	}
	metadata, ok := d.document.AlertMetadata(internalNodeID(id))
	if !ok {
		return Alert{}, false
	}
	return Alert{
		id:          id,
		kind:        publicAlertKind(metadata.Kind),
		sourceRange: Range{Start: metadata.Range.Start, End: metadata.Range.End},
		markerRange: Range{Start: metadata.MarkerRange.Start, End: metadata.MarkerRange.End},
	}, true
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
		alert, ok := d.Alert(publicNodeID(summary.ID))
		if ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// AlertBodyRanges returns caller-owned inner source segments after the alert marker line.
// Marker-only blank lines are represented by valid empty ranges and lazy continuation
// lines retain their source-proven blockquote inner ranges.
func (d *Document) AlertBodyRanges(id NodeID) ([]Range, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	ranges, ok := d.document.AlertBodyRanges(internalNodeID(id))
	if !ok {
		return nil, false
	}
	return publicRanges(ranges), true
}

func publicAlertKind(kind source.AlertKind) AlertKind {
	switch kind {
	case source.AlertNote:
		return AlertKindNote
	case source.AlertTip:
		return AlertKindTip
	case source.AlertImportant:
		return AlertKindImportant
	case source.AlertWarning:
		return AlertKindWarning
	case source.AlertCaution:
		return AlertKindCaution
	default:
		return AlertKindUnknown
	}
}

func sourceAlertKind(kind AlertKind) source.AlertKind {
	switch kind {
	case AlertKindNote:
		return source.AlertNote
	case AlertKindTip:
		return source.AlertTip
	case AlertKindImportant:
		return source.AlertImportant
	case AlertKindWarning:
		return source.AlertWarning
	case AlertKindCaution:
		return source.AlertCaution
	default:
		return source.AlertUnknown
	}
}

func alertMarker(kind AlertKind) (string, bool) {
	return source.AlertMarker(sourceAlertKind(kind))
}
