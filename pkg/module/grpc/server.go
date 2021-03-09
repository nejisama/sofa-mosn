package grpc

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

// service consists of the information of the server serving this service and
// the methods in this service.
type service struct {
	server interface{} // the server for service methods
	md     map[string]*grpc.MethodDesc
	sd     map[string]*grpc.StreamDesc
	mdata  interface{}
}

// Server is a gRPC server to serve RPC requests.
type Server struct {
	mu    sync.Mutex // guards following
	serve bool
	m     map[string]*service // service name -> service info
}

func NewServer() *Server {
	return &Server{
		m: make(map[string]*service),
	}
}

// 为了对接MOSN 这里改掉了,实际上是不需要的
func (s *Server) Serve() {
	s.mu.Lock()
	s.serve = true
	s.mu.Unlock()
}

type MStream struct {
	Ctx       context.Context
	Header    api.HeaderMap
	Body      buffer.IoBuffer
	Trailer   api.HeaderMap
	EndStream bool
}

type Buffer interface {
	Read() (*MStream, error)
	Write(*MStream) error
}

// 这里是为了由Filter传一个类似Conn的Buffer过来,模拟一个Conn的生成的调用
// BufferHandler 代替原来的Transport
// GRPCStream代替原来的Transport.Stream
func (s *Server) HandleBuffer(buf Buffer) {
	st := NewBufferHandler(buf)
	go func() {
		st.HandleStreams(func(stream *GRPCStream) {
			s.handleStream(st, stream)
		})
	}()
}

func (s *Server) handleStream(t BufferHandler, stream *GRPCStream) {
	sm := stream.Method()
	if sm != "" && sm[0] == '/' {
		sm = sm[1:]
	}
	pos := strings.LastIndex(sm, "/")
	if pos == -1 {
		fmt.Println("不识别的Method:", sm)
		return
	}
	service := sm[:pos]
	method := sm[pos+1:]
	srv, knownService := s.m[service]
	if !knownService {
		fmt.Println("没有注册的service:", service)
		return
	}
	// 暂时只支持一种
	if md, ok := srv.md[method]; ok {
		_ = s.processUnaryRPC(t, stream, srv, md)
		return
	}
	fmt.Println("其他场景")
	return
}

// pf 表示是否压缩 暂时不考虑
func (s *Server) readGRPCMessage(data []byte) (pf uint8, msg []byte) {
	pf = uint8(data[0])
	length := binary.BigEndian.Uint32(data[1:])
	if length == 0 {
		return pf, nil
	}
	return pf, data[5:]
}

func (s *Server) processUnaryRPC(t BufferHandler, stream *GRPCStream, srv *service, md *grpc.MethodDesc) error {
	// 核心是不要Decode部分的逻辑了 认为stream里有足够的信息
	_, d := s.readGRPCMessage(stream.Data())
	df := func(v interface{}) error {
		fmt.Println("测试数据:", d)
		if err := s.getCodec(stream.ContentSubtype()).Unmarshal(d, v); err != nil {
			return status.Errorf(codes.Internal, "grpc: error unmarshalling request: %v", err)
		}
		return nil
	}
	ctx := context.Background()                           // 这里先简化，暂时测试的函数中没有ctx的使用
	reply, appErr := md.Handler(srv.server, ctx, df, nil) // interceptor暂时也没用到
	if appErr != nil {
		fmt.Println("Handler报错:", appErr)
		appStatus := status.New(codes.Unknown, appErr.Error())
		t.WriteStatus(stream, appStatus)
		return appErr
	}
	if err := s.sendResponse(t, stream, reply); err != nil {
		appStatus := status.New(codes.Unknown, err.Error())
		t.WriteStatus(stream, appStatus)
		return err
	}
	return nil
}

func (s *Server) sendResponse(t BufferHandler, stream *GRPCStream, msg interface{}) error {
	// pb encode
	data, err := encode(s.getCodec(stream.ContentSubtype()), msg)
	if err != nil {
		return err
	}
	// 处理data
	hdr := make([]byte, 5) // 这个是grpc的头
	hdr[0] = byte(0)       // hardcode不加密
	binary.BigEndian.PutUint32(hdr[1:], uint32(len(data)))
	hdr = append(hdr, data...)
	t.Write(stream, hdr)
	return nil
}

// 简化实现,没有opts判断
// contentSubtype must be lowercase
// cannot return nil
func (s *Server) getCodec(contentSubtype string) baseCodec {
	if contentSubtype == "" {
		return encoding.GetCodec(proto.Name)
	}
	codec := encoding.GetCodec(contentSubtype)
	if codec == nil {
		return encoding.GetCodec(proto.Name)
	}
	return codec
}

// 这里没有动,是原来的函数实现尽量保持兼容
// RegisterService registers a service and its implementation to the gRPC
// server. It is called from the IDL generated code. This must be called before
// invoking Serve.
func (s *Server) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(ss)
	if !st.Implements(ht) {
		grpclog.Fatalf("grpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
	}
	s.register(sd, ss)
}

func (s *Server) register(sd *grpc.ServiceDesc, ss interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serve {
		grpclog.Fatalf("grpc: Server.RegisterService after Server.Serve for %q", sd.ServiceName)
	}
	if _, ok := s.m[sd.ServiceName]; ok {
		grpclog.Fatalf("grpc: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
	}
	srv := &service{
		server: ss,
		md:     make(map[string]*grpc.MethodDesc),
		sd:     make(map[string]*grpc.StreamDesc),
		mdata:  sd.Metadata,
	}
	for i := range sd.Methods {
		d := &sd.Methods[i]
		srv.md[d.MethodName] = d
	}
	for i := range sd.Streams {
		d := &sd.Streams[i]
		srv.sd[d.StreamName] = d
	}
	s.m[sd.ServiceName] = srv
}
