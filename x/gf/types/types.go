package types

import "github.com/verana-labs/verana-node/util/validation"

// IsValidBCP47 reports whether s is a well-formed BCP 47 language tag.
func IsValidBCP47(s string) bool { return validation.IsValidBCP47(s) }

// IsValidURL reports whether s is a valid absolute http(s) URL with a host.
func IsValidURL(s string) bool { return validation.IsValidURL(s) }
