package grpc

import (
	"math"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// encode serializes msg and returns a buffer containing the message, or an
// error if it is too large to be transmitted by grpc.  If msg is nil, it
// generates an empty message.
func encode(c baseCodec, msg interface{}) ([]byte, error) {
	if msg == nil { // NOTE: typed nils will not be caught by this check
		return nil, nil
	}
	b, err := c.Marshal(msg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "grpc: error while marshaling: %v", err.Error())
	}
	if uint(len(b)) > math.MaxUint32 {
		return nil, status.Errorf(codes.ResourceExhausted, "grpc: message too large (%d bytes)", len(b))
	}
	return b, nil
}
