package grpc

import (
	"fmt"
	"strings"
)

type GRPCStream struct {
	method  string
	data    []byte
	subtype string
	buf     Buffer
}

func (s *GRPCStream) Method() string {
	return s.method
}

// 如果要支持流式这里要重新改一改 暂时先不用，简化
func (s *GRPCStream) Data() []byte {
	return s.data
}

func (s *GRPCStream) ContentSubtype() string {
	return s.subtype
}

// 简化实现
func (s *GRPCStream) processHeader(key, value string) {
	name := strings.ToLower(key)
	fmt.Println("测试Header:", name, value)
	switch name {
	case "content-type":
		contentSubtype, validContentType := contentSubtype(value)
		if !validContentType {
			//  错误处略都略
			return
		}
		s.subtype = contentSubtype
	case ":path":
		s.method = value
		//  其他暂时看上去都用不上
	}
}
