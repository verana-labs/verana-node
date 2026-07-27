package types

import "context"

// DIDOwnerResolver resolves the corporation owning a DID in another module (ec/pp),
// injected into co post-construction to avoid an import cycle.
type DIDOwnerResolver interface {
	ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error)
}
