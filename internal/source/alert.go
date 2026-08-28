package source

// AlertKind identifies one reviewed GitHub alert marker at the source layer.
type AlertKind uint8

const (
	AlertUnknown AlertKind = iota
	AlertNote
	AlertTip
	AlertImportant
	AlertWarning
	AlertCaution
)

// AlertKindFromMarker recognizes one exact reviewed GitHub alert marker.
func AlertKindFromMarker(marker []byte) AlertKind {
	switch string(marker) {
	case "[!NOTE]":
		return AlertNote
	case "[!TIP]":
		return AlertTip
	case "[!IMPORTANT]":
		return AlertImportant
	case "[!WARNING]":
		return AlertWarning
	case "[!CAUTION]":
		return AlertCaution
	default:
		return AlertUnknown
	}
}

// AlertMarker returns the canonical marker for one reviewed alert kind.
func AlertMarker(kind AlertKind) (string, bool) {
	switch kind {
	case AlertNote:
		return "[!NOTE]", true
	case AlertTip:
		return "[!TIP]", true
	case AlertImportant:
		return "[!IMPORTANT]", true
	case AlertWarning:
		return "[!WARNING]", true
	case AlertCaution:
		return "[!CAUTION]", true
	default:
		return "", false
	}
}
