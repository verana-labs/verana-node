package types

import "github.com/verana-labs/verana-node/util/validation"

// IsValidBCP47 reports whether s is a well-formed BCP 47 tag within the spec's
// string(17) bound for the Corporation language field (MOD-CO-MSG-1).
func IsValidBCP47(s string) bool { return len(s) <= 17 && validation.IsValidBCP47(s) }

// IsValidURL reports whether s is a valid http(s) URL, matching the shared
// validator x/gf genesis applies to the seeded governance-framework document.
func IsValidURL(s string) bool { return validation.IsValidURL(s) }
