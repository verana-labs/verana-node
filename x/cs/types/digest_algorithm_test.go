package types

import "testing"

// [MOD-CS-MSG-1-1] restricts digest_algorithm to sha384 and sha512. sha256 was
// accepted by earlier builds; credentials issued under a sha256 schema cannot
// produce a spec-conformant digestJCS, and digest_algorithm is immutable once
// the schema is created, so the rejection is guarded here.
func TestValidateDigestAlgorithm(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantErr   bool
	}{
		{"sha384", "sha384", false},
		{"sha512", "sha512", false},
		{"sha256 rejected", "sha256", true},
		{"empty", "", true},
		{"uppercase", "SHA384", true},
		{"unknown", "md5", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDigestAlgorithm(tc.algorithm)
			if tc.wantErr && err == nil {
				t.Errorf("validateDigestAlgorithm(%q) = nil, want error", tc.algorithm)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateDigestAlgorithm(%q) = %v, want nil", tc.algorithm, err)
			}
		})
	}
}
