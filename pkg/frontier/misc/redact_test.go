package misc

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	for _, meta := range []string{`{"token":"secret"}`, `["secret"]`, `{"payload":"{\"token\":\"secret\"}"}`, `{"hostname":"also-untrusted"}`, "not JSON", strings.Repeat(" ", 16384), "  ", "null"} {
		if got := Redact(meta); got != "<redacted: meta omitted>" {
			t.Fatalf("metadata leaked: %q", got)
		}
	}
	if Redact("") != "" {
		t.Fatal("empty metadata")
	}
}
