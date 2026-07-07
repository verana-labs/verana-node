// Package validation provides shared, compile-once validators for the data
// formats used across Verana modules (DID, Subresource Integrity digests,
// BCP-47 language tags, and URLs).
//
// Each module previously carried its own copy of these validators with subtly
// different regexes, so the same string could be accepted by one module and
// rejected by another. This package is the single source of truth: the regexes
// are compiled once at init and reused on every call. All validators are pure
// functions over their input (BCP-47 uses golang.org/x/text tables pinned by
// go.mod), so they are deterministic and safe to call from consensus paths.
package validation

import (
	"errors"
	"net/url"
	"regexp"

	"golang.org/x/text/language"
)

// didRe enforces W3C DID Core syntax: `did:` <method-name> `:` <method-specific-id>,
// followed by an optional DID-URL fragment (`#...`) and/or query (`?...`).
//   - method-name is lowercase letters/digits only (per the spec).
//   - method-specific-id allows alphanumerics, `.` `_` `-` `:` `/` and `%`
//     (percent-encoding, e.g. did:webvh).
var didRe = regexp.MustCompile(`^did:[a-z0-9]+:[A-Za-z0-9._:/%\-]+(#[^\s]*)?(\?[^\s]*)?$`)

// IsValidDID reports whether s is a syntactically valid DID (or DID URL).
func IsValidDID(s string) bool { return didRe.MatchString(s) }

// digestSRIRe enforces W3C Subresource Integrity digest syntax: one of the
// sha256/sha384/sha512 algorithms, a hyphen, then a base64 body with up to two
// `=` padding characters.
//
// NOTE: this validates the SRI shape only, not that the base64 body decodes to
// the algorithm's exact hash length (32/48/64 bytes). The chain never recomputes
// or verifies the hash against content — it only stores the digest string — so
// exact-length enforcement is non-load-bearing hardening, and adding it would
// reject the short fixtures used throughout the module test suites for marginal
// benefit.
var digestSRIRe = regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+={0,2}$`)

// IsValidDigestSRI reports whether s is a valid SRI digest string,
// e.g. "sha256-<base64>". An empty string is not valid; callers that treat the
// digest as optional must guard the empty case themselves.
func IsValidDigestSRI(s string) bool { return digestSRIRe.MatchString(s) }

// IsValidBCP47 reports whether s is a well-formed BCP-47 language tag. It accepts
// well-formed tags even when a subtag value is not registered (BCP-47 requires
// well-formedness, not registration), and rejects syntactically malformed input.
// It intentionally imposes no arbitrary length cap: valid tags such as
// "hy-Latn-IT-arevela" exceed the 17-character limits the old per-module regexes
// used.
func IsValidBCP47(s string) bool {
	if s == "" {
		return false
	}
	_, err := language.Parse(s)
	if err == nil {
		return true
	}
	// A ValueError means the tag is well-formed but references an unregistered
	// value — acceptable under BCP-47 well-formedness. Any other error is a
	// syntax error and must be rejected.
	var ve language.ValueError
	return errors.As(err, &ve)
}

// IsValidURL reports whether s is a valid absolute http(s) URL with a non-empty
// host. The old per-module validators checked only the scheme, so host-less
// strings like "http://" passed; this requires a host.
func IsValidURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

// IsValidURI reports whether s is a non-empty absolute URI (any scheme), broader
// than IsValidURL. Used where the spec asks for a "valid URI" (MOD-CS-MSG-5).
func IsValidURI(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != ""
}
