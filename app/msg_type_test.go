package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	detypes "github.com/verana-labs/verana-node/x/de/types"
)

func TestMsgTypeURLsAreRoutable(t *testing.T) {
	a, _ := feegrantTestApp(t)
	for n := range detypes.MsgType_name {
		for _, url := range detypes.MsgType(n).URLs() {
			require.NotNil(t, a.MsgServiceRouter().HandlerByTypeURL(url), url)
		}
	}
}
