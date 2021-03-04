package grpc

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
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
	// TODO: 支持流式
	f.buf.Send(mstream)
	resp := f.buf.Recv()
	log.DefaultLogger.Infof("响应头:", resp.Header)
	log.DefaultLogger.Infof("响应体:", resp.Body.Bytes())
	log.DefaultLogger.Infof("响应尾:", resp.Trailer)
	//f.handler.SendDirectResponse(resp.Header, resp.Body, resp.Trailer)
	resp.Trailer.Range(func(k, v string) bool {
		resp.Header.Set(k, v)
		return true
	})
	resp.Header.Del(":status")
	log.DefaultLogger.Infof("修改后的头", resp.Header)
	f.handler.SendHijackReplyWithBody(200, resp.Header, resp.Body.String())
	return api.StreamFilterStop
}

func (f *grpcFilter) OnDestroy() {}
