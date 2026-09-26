package splice

import "github.com/zoster81/marksplice/internal/source"

// AlertMetadata is the scalar projection of a source-proven top-level alert.
type AlertMetadata struct {
	Kind        source.AlertKind
	Range       Range
	MarkerRange Range
}

// AlertMetadata recognizes an alert without copying its source or body ranges.
func (d *Document) AlertMetadata(id NodeID) (AlertMetadata, bool) {
	mapping, kind, ok := d.alertSource(id)
	if !ok {
		return AlertMetadata{}, false
	}
	return AlertMetadata{Kind: kind, Range: mapping.LineRange, MarkerRange: mapping.ContentRanges[0]}, true
}

// AlertBodyRanges returns caller-owned physical body segments after the marker.
func (d *Document) AlertBodyRanges(id NodeID) ([]Range, bool) {
	mapping, _, ok := d.alertSource(id)
	if !ok {
		return nil, false
	}
	return append([]Range(nil), mapping.ContentRanges[1:]...), true
}

func (d *Document) alertSource(id NodeID) (source.BlockquoteMapping, source.AlertKind, bool) {
	if d == nil {
		return source.BlockquoteMapping{}, source.AlertUnknown, false
	}
	node, ok := d.nodeByID(id)
	if !ok {
		return source.BlockquoteMapping{}, source.AlertUnknown, false
	}
	mapping, ok := d.blockquoteSource(node)
	if !ok {
		return source.BlockquoteMapping{}, source.AlertUnknown, false
	}
	kind := blockquoteMappingAlertKind(d.source, mapping)
	if kind != source.AlertUnknown {
		// A valid marker alone does not make a readable alert. Retain the
		// physical-range rule, including empty lines and lazy continuation.
		for _, range_ := range mapping.ContentRanges[1:] {
			if range_.Start < range_.End {
				return mapping, kind, true
			}
		}
	}
	return source.BlockquoteMapping{}, source.AlertUnknown, false
}
