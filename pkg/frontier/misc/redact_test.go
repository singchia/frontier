package misc

import (
	"strings"
	"testing"
)

// Table-driven tests: values of credential-like JSON keys must be masked,
// non-sensitive fields must be preserved, and every non-object input shape
// (non-JSON, oversized, top-level scalar) must fail closed without echoing
// the raw value.
func TestRedact(t *testing.T) {
	testcases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string returned as is",
			in:   "",
			want: "",
		},
		{
			name: "whitespace-only string returned as is",
			in:   "  ",
			want: "  ",
		},
		{
			name: "access_key and secret_key masked",
			in:   `{"access_key":"AK123","secret_key":"SK456"}`,
			want: `{"access_key":"***","secret_key":"***"}`,
		},
		{
			name: "non-sensitive fields preserved",
			in:   `{"hostname":"edge5","version":"1.0"}`,
			want: `{"hostname":"edge5","version":"1.0"}`,
		},
		{
			name: "token/password/passwd/credential variants masked",
			in:   `{"token":"t1","password":"p1","passwd":"p2","credentials":"c1"}`,
			want: `{"credentials":"***","passwd":"***","password":"***","token":"***"}`,
		},
		{
			name: "extended fragment table: pwd/passphrase/bearer/session/cookie/cert/private masked",
			in:   `{"pwd":"p","passphrase":"pp","bearer":"bt","session_id":"s1","cookie":"ck","client_cert":"c","private":"pr"}`,
			want: `{"bearer":"***","client_cert":"***","cookie":"***","passphrase":"***","private":"***","pwd":"***","session_id":"***"}`,
		},
		{
			name: "case-insensitive matching",
			in:   `{"AccessKey":"AK","SECRET_KEY":"SK"}`,
			want: `{"AccessKey":"***","SECRET_KEY":"***"}`,
		},
		{
			name: "sensitive field inside nested object masked",
			in:   `{"client":{"secret_key":"SK","name":"edge"},"keep":"v"}`,
			want: `{"client":{"name":"edge","secret_key":"***"},"keep":"v"}`,
		},
		{
			name: "sensitive field inside object in array masked",
			in:   `[{"token":"t"},{"ok":1}]`,
			want: `[{"token":"***"},{"ok":1}]`,
		},
		{
			name: "non-string value under sensitive key also masked",
			in:   `{"access_key":{"nested":"x"},"secret_key":[1,2]}`,
			want: `{"access_key":"***","secret_key":"***"}`,
		},
		{
			name: "key fragment matches anywhere in key name",
			in:   `{"edge_access_key_2":"AK","region":"cn"}`,
			want: `{"edge_access_key_2":"***","region":"cn"}`,
		},
		{
			name: "authorization and signature variants masked",
			in:   `{"authorization":"Bearer eyJ","signature":"sig1","region":"cn"}`,
			want: `{"authorization":"***","region":"cn","signature":"***"}`,
		},
		{
			name: "over-redaction of innocuous lookalike keys is accepted",
			in:   `{"keyword":"k","author":"a","monkey":"m"}`,
			want: `{"author":"***","keyword":"***","monkey":"***"}`,
		},
		{
			name: "non-JSON input fails closed",
			in:   `not a json`,
			want: `<redacted: unparsable meta>`,
		},
		{
			name: "top-level credential-bearing JSON string fails closed",
			in:   `"plain-secret-token-value"`,
			want: `<redacted: non-object meta>`,
		},
		{
			name: "double-encoded JSON with credentials fails closed",
			in:   `"{\"secret_key\":\"SK\"}"`,
			want: `<redacted: non-object meta>`,
		},
		{
			name: "top-level scalar number fails closed",
			in:   `123`,
			want: `<redacted: non-object meta>`,
		},
		{
			name: "top-level null fails closed",
			in:   `null`,
			want: `<redacted: non-object meta>`,
		},
		{
			name: "trailing garbage after valid JSON is dropped",
			in:   `{"a":1} trailing-secret`,
			want: `{"a":1}`,
		},
		{
			name: "input at exactly maxMetaLen bytes still redacts",
			in:   `{"filler":"` + strings.Repeat("x", maxMetaLen-13) + `"}`,
			want: `{"filler":"` + strings.Repeat("x", maxMetaLen-13) + `"}`,
		},
		{
			name: "input one byte over maxMetaLen fails closed",
			in:   `{"filler":"` + strings.Repeat("x", maxMetaLen-12) + `"}`,
			want: `<redacted: meta too long>`,
		},
		{
			name: "oversized meta fails closed",
			in:   `{"filler":"` + strings.Repeat("x", 9*1024) + `"}`,
			want: `<redacted: meta too long>`,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.in)
			if got != tc.want {
				t.Errorf("Redact(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
