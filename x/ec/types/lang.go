package types

import "github.com/verana-labs/verana-node/util/validation"

// IsValidBCP47 reports whether tag is a well-formed BCP 47 language tag.
func IsValidBCP47(tag string) bool { return validation.IsValidBCP47(tag) }

func isValidBCP47(tag string) bool { return IsValidBCP47(tag) }
