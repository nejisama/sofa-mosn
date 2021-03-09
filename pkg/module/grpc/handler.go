package grpc

import (
	"fmt"
	"strconv"

	"google.golang.org/grpc/status"
	"mosn.io/pkg/buffer"
)

// 代替原来的Transport的简化实现
type BufferHandler interface {
	WriteStatus(*GRPCStream, *status.Status)
	Write(*GRPCStream, []byte)
	HandleStreams(func(*GRPCStream))
	Close()
}

func NewBufferHandler(buf Buffer) BufferHandler {
	return &bufferHandler{
		buf: buf,
	}
}

type bufferHandler struct {
	buf Buffer // 代替原来的Connection
}

func (t *bufferHandler) HandleStreams(handle func(*GRPCStream)) {
	for {
		mstream, err := t.buf.Read()
		if err != nil {
			// 这里本来还有一些异常判断可以Continue的，暂时简化掉
			t.Close()
			return
		}
		// 这里暂时也简化掉,认为Read完成了全部的封装
		if t.handleStream(mstream, handle) {
			t.Close()
		}
	}
}

func (t *bufferHandler) Close() {
	// 暂时先不实现，应该包括清理streams 一类的
}

func (t *bufferHandler) WriteStatus(s *GRPCStream, st *status.Status) {
	// 构造一个Trailer
	header := CommonHeader(map[string]string{
		":status":      "200",
		"content-type": contentType(s.subtype),
	})
	trailer := CommonHeader(map[string]string{
		"grpc-status":  strconv.Itoa(int(st.Code())),
		"grpc-message": encodeGrpcMessage(st.Message()),
	})
	ms := &MStream{
		Header:  header,
		Trailer: trailer,
	}
	fmt.Println("测试异常Write")
	t.buf.Write(ms)
}

func (t *bufferHandler) Write(s *GRPCStream, data []byte) {
	header := CommonHeader(map[string]string{
		":status":      "200",
		"content-type": contentType(s.subtype),
	})
	trailer := CommonHeader(map[string]string{
		"grpc-status": "0",
	})
	ms := &MStream{
		Header:  header,
		Trailer: trailer,
		Body:    buffer.NewIoBufferBytes(data), // TODO:内存复用
	}
	fmt.Println("测试正常Write")
	t.buf.Write(ms)
}

// 构造grpc要用的结构体
func (t *bufferHandler) handleStream(stream *MStream, handle func(*GRPCStream)) (fatal bool) {
	// 理论上做流式这里需要一些进一步实现，这里都先简化掉了
	// 包括EndStream什么的
	s := &GRPCStream{
		buf:  t.buf,
		data: stream.Body.Bytes(), // 细节忽略
	}
	// 解析Header, 在mosn的模式下有些内容会不会在Trailer里？不确定
	stream.Header.Range(func(key, value string) bool {
		s.processHeader(key, value)
		return true
	})
	// :path 被mosn特殊处理了
	if m, ok := stream.Header.Get(":path"); ok {
		fmt.Println("测试PATH:", m)
		s.method = m
	}
	handle(s)
	return false
}
