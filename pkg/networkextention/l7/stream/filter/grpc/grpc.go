package grpc

import (
	"context"
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/networkextention/pkg/module/grpc"
	"mosn.io/pkg/buffer"
)

type grpcFilter struct {
	handler api.StreamReceiverFilterHandler
	fc      *GRPCFactory
	buf     *mBuffer
}

func NewFilter(fc *GRPCFactory) *grpcFilter {
	return &grpcFilter{
		fc: fc,
	}
}

func (f *grpcFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
	id := handler.Connection().ID()
	buf, ok := f.fc.buffers[id]
	if !ok {
		buf = newBuffer()
		f.fc.server.HandleBuffer(buf)
	}
	f.buf = buf
}

func (f *grpcFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	mstream := &grpc.MStream{
		Ctx:       ctx,
		Header:    headers,
		Body:      buf,
		Trailer:   trailers,
		EndStream: true, // TODO: 应该有一个办法判断
	}
	fmt.Println("收到的请求:", headers, buf.Bytes(), trailers)
	// TODO: 支持流式
	f.buf.Send(mstream)
	resp := f.buf.Recv()
	fmt.Println("响应的内容:", resp)
	f.handler.SendDirectResponse(resp.Header, resp.Body, resp.Trailer)
	return api.StreamFilterStop
}

func (f *grpcFilter) OnDestroy() {}
