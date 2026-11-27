package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/gf/types"
)

const (
	tCorp   = "cosmos14wcc52lpsxwuxxhqjxrhvuumhm0xr6z247un93"
	tOp     = "cosmos1fvz0kp4jfseea3zyduu78dd5yqcwrarwtxthjn"
	tAuth   = "cosmos1lyfknrsmxhlr7rflvuz6x7jjjpnx4s5uywj78f"
	tURL    = "https://example.com/gf.html"
	tDigest = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	t.Run("missing authority", func(t *testing.T) {
		m := &types.MsgUpdateParams{Signer: "", Params: types.DefaultParams()}
		require.Error(t, m.ValidateBasic())
	})
	t.Run("happy path", func(t *testing.T) {
		m := &types.MsgUpdateParams{Signer: tAuth, Params: types.DefaultParams()}
		require.NoError(t, m.ValidateBasic())
	})
}

func TestMsgAddGovernanceFrameworkDocument_ValidateBasic(t *testing.T) {
	valid := types.MsgAddGovernanceFrameworkDocument{
		Corporation:  tCorp,
		Signer:       tOp,
		DocLanguage:  "en",
		DocUrl:       tURL,
		DocDigestSri: tDigest,
		Version:      1,
	}

	t.Run("happy path (corporation-targeted)", func(t *testing.T) {
		m := valid
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("happy path (ecosystem-targeted)", func(t *testing.T) {
		m := valid
		m.EcosystemId = 42
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("missing corporation", func(t *testing.T) {
		m := valid
		m.Corporation = ""
		require.Error(t, m.ValidateBasic())
	})
	t.Run("missing delegate", func(t *testing.T) {
		m := valid
		m.Signer = ""
		require.Error(t, m.ValidateBasic())
	})
	t.Run("missing doc_language", func(t *testing.T) {
		m := valid
		m.DocLanguage = ""
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidLanguage)
	})
	t.Run("invalid doc_language BCP47", func(t *testing.T) {
		m := valid
		m.DocLanguage = "1bad"
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidLanguage)
	})
	t.Run("missing doc_url", func(t *testing.T) {
		m := valid
		m.DocUrl = ""
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidURL)
	})
	t.Run("invalid doc_url", func(t *testing.T) {
		m := valid
		m.DocUrl = "not a url"
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidURL)
	})
	t.Run("missing doc_digest_sri", func(t *testing.T) {
		m := valid
		m.DocDigestSri = ""
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidDigestSRI)
	})
	t.Run("invalid doc_digest_sri", func(t *testing.T) {
		m := valid
		m.DocDigestSri = "md5-deadbeef"
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidDigestSRI)
	})
	t.Run("version below 1", func(t *testing.T) {
		m := valid
		m.Version = 0
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidVersion)
	})
	// Negative version is unrepresentable now that Version is uint32 — the
	// type system enforces it at compile time, so no runtime test is needed.

	// [MOD-GF-MSG-1-1] network path: corporation and ecosystem_id absent.
	network := valid
	network.Corporation, network.Signer, network.Network = "", tAuth, true
	t.Run("network path", func(t *testing.T) {
		m := network
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("network path with corporation", func(t *testing.T) {
		m := network
		m.Corporation = tCorp
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidSubject)
	})
	t.Run("network path with ecosystem_id", func(t *testing.T) {
		m := network
		m.EcosystemId = 42
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidSubject)
	})
	t.Run("invalid signer", func(t *testing.T) {
		m := network
		m.Signer = "bad"
		require.Error(t, m.ValidateBasic())
	})
}

func TestMsgIncreaseActiveGovernanceFrameworkVersion_ValidateBasic(t *testing.T) {
	valid := types.MsgIncreaseActiveGovernanceFrameworkVersion{
		Corporation: tCorp,
		Signer:      tOp,
	}

	t.Run("happy path (corporation-targeted)", func(t *testing.T) {
		m := valid
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("happy path (ecosystem-targeted)", func(t *testing.T) {
		m := valid
		m.EcosystemId = 7
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("missing corporation", func(t *testing.T) {
		m := valid
		m.Corporation = ""
		require.Error(t, m.ValidateBasic())
	})
	t.Run("missing signer", func(t *testing.T) {
		m := valid
		m.Signer = ""
		require.Error(t, m.ValidateBasic())
	})

	// [MOD-GF-MSG-2-1] network path.
	network := types.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: tAuth, Network: true}
	t.Run("network path", func(t *testing.T) {
		m := network
		require.NoError(t, m.ValidateBasic())
	})
	t.Run("network path with corporation", func(t *testing.T) {
		m := network
		m.Corporation = tCorp
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidSubject)
	})
	t.Run("network path without signer", func(t *testing.T) {
		m := network
		m.Signer = ""
		require.Error(t, m.ValidateBasic())
	})
	t.Run("network path with ecosystem_id", func(t *testing.T) {
		m := network
		m.EcosystemId = 7
		require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidSubject)
	})
}
