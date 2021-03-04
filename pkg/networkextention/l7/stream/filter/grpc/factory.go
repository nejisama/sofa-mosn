package grpc

import (
	"context"
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/networkextention/pkg/module/grpc"
)

func init() {
	api.RegisterStream("grpc", CreateGRPCFactory)
}

var once sync.Once
var f *GRPCFactory

type GRPCFactory struct {
	buffers map[uint64]*mBuffer
	server  *grpc.Server
}

func (f *GRPCFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewFilter(f)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

func CreateGRPCFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	once.Do(func() {
		server := grpc.NewServer()
		server.RegisterService(&_Greeter_serviceDesc, &pb{}) // pb 的内容，怎么获取?
		server.Serve()
		f = &GRPCFactory{
			buffers: make(map[uint64]*mBuffer),
			server:  server,
		}
	})
	return f, nil
}

type pb struct {
	UnimplementedGreeterServer
}

func (a *pb) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}
