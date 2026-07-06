package types

import "github.com/verana-labs/verana-node/util/validation"

// IsValidBCP47 reports whether tag is a well-formed BCP 47 language tag that
// also fits the spec's string(17) length bound for the Ecosystem language
// field (MOD-ES-MSG-1), matching the cap enforced by x/co.
func IsValidBCP47(tag string) bool { return len(tag) <= 17 && validation.IsValidBCP47(tag) }

func isValidBCP47(tag string) bool { return IsValidBCP47(tag) }
