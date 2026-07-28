package types

import "context"

// DIDOwnerResolver resolves the corporation owning a DID in another module (pp),
// injected into ec post-construction to avoid an import cycle.
type DIDOwnerResolver interface {
	ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error)
}
