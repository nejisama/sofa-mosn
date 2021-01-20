package grpc

import "mosn.io/mosn/pkg/module/grpc"

type mBuffer struct {
	req  chan *grpc.MStream
	resp chan *grpc.MStream
}

// TODO: 需要完善
func (b *mBuffer) Read() (*grpc.MStream, error) {
	m := <-b.req
	return m, nil
}

func (b *mBuffer) Write(m *grpc.MStream) error {
	b.resp <- m
	return nil
}

func (b *mBuffer) Send(m *grpc.MStream) {
	b.req <- m
}

func (b *mBuffer) Recv() *grpc.MStream {
	m := <-b.resp
	return m
}

func newBuffer() *mBuffer {
	return &mBuffer{
		req:  make(chan *grpc.MStream, 10),
		resp: make(chan *grpc.MStream, 10),
	}
}
