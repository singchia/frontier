package misc

import (
	"bytes"
	"encoding/json"
	"strings"
)

// redactedValue is the uniform replacement for sensitive field values.
const redactedValue = "***"

// unparsableMetaPlaceholder is the fail-closed output for non-JSON input:
// the raw value is never echoed back, so malformed payloads cannot smuggle
// credentials into logs.
const unparsableMetaPlaceholder = "<redacted: unparsable meta>"

// nonObjectMetaPlaceholder is the fail-closed output for JSON input whose
// top-level value is not an object or an array (e.g. a bare string). A
// top-level string is a common client mistake that carries an
// already-serialized object, so it is masked as a whole instead of being
// passed through.
const nonObjectMetaPlaceholder = "<redacted: non-object meta>"

// oversizedMetaPlaceholder is the fail-closed output for oversized meta.
// Meta is client-controlled and should not flood logs anyway; returning a
// placeholder upfront also avoids parsing hostile large JSON payloads.
const oversizedMetaPlaceholder = "<redacted: meta too long>"

// maxMetaLen is the length limit for meta accepted by the log redaction path.
const maxMetaLen = 8 * 1024

// sensitiveKeyFragments holds lowercase substrings used to detect
// credential-like keys. Clients commonly carry access_key/secret_key/token
// and friends in connection meta, and those must be masked before logging.
// Substring matching covers naming variants (accessKey, secret_key_2, ...).
// It intentionally errs on the side of over-masking: innocuous keys such as
// "keyword" or "author" are masked too, which is preferred over leaking.
var sensitiveKeyFragments = []string{
	"key",
	"secret",
	"token",
	"password",
	"passwd",
	"pass",
	"pwd",
	"credential",
	"auth",
	"signature",
	"bearer",
	"session",
	"cookie",
	"cert",
	"private",
}

// Redact masks credential-like fields in a JSON meta string and returns the
// result, for use at every call site that logs meta. Contract:
//   - empty/whitespace-only input is returned as is;
//   - input longer than maxMetaLen fails closed with a placeholder;
//   - values of sensitive keys (case-insensitive substring match) are
//     replaced with "***" at any level (objects, nested objects, objects
//     inside arrays), regardless of whether the value is a string or a
//     composite;
//   - a top-level value that is not an object or an array fails closed with
//     a placeholder;
//   - non-JSON input fails closed with a placeholder, never echoing the raw
//     value. If valid JSON is followed by trailing garbage, only the first
//     JSON value is parsed and the trailing content is dropped.
func Redact(meta string) string {
	if strings.TrimSpace(meta) == "" {
		return meta
	}
	if len(meta) > maxMetaLen {
		return oversizedMetaPlaceholder
	}
	var v any
	dec := json.NewDecoder(bytes.NewReader([]byte(meta)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return unparsableMetaPlaceholder
	}
	switch v.(type) {
	case map[string]any, []any:
	default:
		return nonObjectMetaPlaceholder
	}
	redactValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return unparsableMetaPlaceholder
	}
	return string(out)
}

// redactValue recursively masks in place: map keys are matched one by one
// (a hit replaces the whole value), slices are descended element-wise, and
// other scalars are left untouched.
func redactValue(v any) {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if isSensitiveKey(k) {
				val[k] = redactedValue
				continue
			}
			redactValue(item)
		}
	case []any:
		for _, item := range val {
			redactValue(item)
		}
	}
}

// isSensitiveKey matches the lowercased key against the fragment table.
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, frag := range sensitiveKeyFragments {
		if strings.Contains(lower, frag) {
			return true
		}
	}
	return false
}
