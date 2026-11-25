package types

import (
	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/x/tx/signing"
	"github.com/cosmos/gogoproto/proto"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// CustomGetSigners returns the signer of the two framework messages: the
// council account in `authority` on the network path, else `delegate`.
func CustomGetSigners(ac address.Codec) []signing.CustomGetSigner {
	fn := func(msg protov2.Message) ([][]byte, error) {
		m := msg.ProtoReflect()
		fields := m.Descriptor().Fields()
		signer := m.Get(fields.ByName("authority")).String()
		if signer == "" {
			signer = m.Get(fields.ByName("delegate")).String()
		}
		bz, err := ac.StringToBytes(signer)
		if err != nil {
			return nil, err
		}
		return [][]byte{bz}, nil
	}
	return []signing.CustomGetSigner{
		{MsgType: protoreflect.FullName(proto.MessageName(&MsgAddGovernanceFrameworkDocument{})), Fn: fn},
		{MsgType: protoreflect.FullName(proto.MessageName(&MsgIncreaseActiveGovernanceFrameworkVersion{})), Fn: fn},
	}
}
