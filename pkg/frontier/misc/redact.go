package misc

// Redact omits opaque peer metadata from logs. Field-name heuristics cannot
// establish whether values, array elements or encoded strings contain secrets.
// Connection and stream IDs are logged separately by callers.
func Redact(meta string) string {
	if meta == "" {
		return ""
	}
	return "<redacted: meta omitted>"
}
