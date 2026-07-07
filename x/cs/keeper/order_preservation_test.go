package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/cs/keeper"
)

// [MOD-CS-MSG-1-3] json_schema MUST be saved canonized (JCS, RFC 8785), so
// object keys are sorted on storage.
func TestCreateCredentialSchema_CanonicalizesJSON(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	authority := sdk.AccAddress([]byte("order_authority_____")).String()
	operator := sdk.AccAddress([]byte("order_operator______")).String()
	trID := mockTrk.CreateMockEcosystem(authority, "did:example:order")

	// Properties in deliberately NON-alphabetical order.
	schema := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "OrderCredential",
  "description": "order test",
  "type": "object",
  "properties": {
    "zebra": { "type": "string" },
    "apple": { "type": "string" },
    "mango": { "type": "string" }
  }
}`
	msg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, schema, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	require.NoError(t, msg.ValidateBasic())

	resp, err := ms.CreateCredentialSchema(ctx, msg)
	require.NoError(t, err)

	stored, err := k.CredentialSchema.Get(sdkCtx, resp.Id)
	require.NoError(t, err)

	// JCS sorts object keys: apple < mango < zebra regardless of submit order.
	ai := strings.Index(stored.JsonSchema, `"apple"`)
	mi := strings.Index(stored.JsonSchema, `"mango"`)
	zi := strings.Index(stored.JsonSchema, `"zebra"`)
	require.True(t, ai >= 0 && mi >= 0 && zi >= 0, "all properties present")
	require.Less(t, ai, mi, "apple must precede mango (JCS-sorted)")
	require.Less(t, mi, zi, "mango must precede zebra (JCS-sorted)")

	// Canonical $id is colon-form and injected first.
	require.Contains(t, stored.JsonSchema, ":cs:")
}
