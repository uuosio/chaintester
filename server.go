package chaintester

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/learnforpractice/chaintester/interfaces"

	_ "unsafe"

	"github.com/apache/thrift/lib/go/thrift"
)

var g_VMAPI *interfaces.ApplyClient

func GetVMAPI() *interfaces.ApplyClient {
	if g_VMAPI != nil {
		return g_VMAPI
	}

	var err error
	g_VMAPI, err = NewVMAPIClient("127.0.0.1:9092")
	if err != nil {
		panic(err)
	}
	return g_VMAPI
}

func NewVMAPIClient(addr string) (*interfaces.ApplyClient, error) {
	var transport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(nil)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)

	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	transport = thrift.NewTSocketConf(addr, cfg)
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return nil, err
	}
	// defer transport.Close()
	if err := transport.Open(); err != nil {
		return nil, err
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	// transport.Close()
	// oprot.Transport().Close()
	return interfaces.NewApplyClient(NewIPCClient(iprot, oprot)), nil
}

type ApplyRequestHandler struct {
}

func NewApplyRequestHandler() *ApplyRequestHandler {
	return &ApplyRequestHandler{}
}

//go:linkname callNativeApply main.native_apply
func callNativeApply(receiver uint64, firstReceiver uint64, action uint64)

func getUint64(value *interfaces.Uint64) uint64 {
	return (uint64(value.Hi) << 32) | uint64(value.Lo)
}

var g_apply_func func(uint64, uint64, uint64)

func SetApplyFunc(apply func(uint64, uint64, uint64)) {
	g_apply_func = apply
}

func (p *ApplyRequestHandler) ApplyRequest(ctx context.Context, receiver *interfaces.Uint64, firstReceiver *interfaces.Uint64, action *interfaces.Uint64) (_r int32, _err error) {
	fmt.Println("+++++++ApplyRequest called!")

	_receiver := getUint64(receiver)
	_firstReceiver := getUint64(firstReceiver)
	_action := getUint64(action)
	if g_apply_func == nil {
		panic("apply function not set!")
	}
	g_apply_func(_receiver, _firstReceiver, _action)
	GetVMAPI().EndApply(ctx)
	return 1, nil
}

func (p *ApplyRequestHandler) ApplyEnd(ctx context.Context) (_r int32, _err error) {
	return 1, nil
}

type ApplyRequestServer struct {
	server *SimpleIPCServer
}

func NewApplyRequestServer() *ApplyRequestServer {
	var transport thrift.TServerTransport
	var err error
	transport, err = thrift.NewTServerSocket("127.0.0.1:9091")
	if err != nil {
		return nil
	}

	handler := NewApplyRequestHandler()
	processor := interfaces.NewApplyRequestProcessor(handler)

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(nil)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)

	server := &ApplyRequestServer{
		server: NewSimpleIPCServer4(processor, transport, transportFactory, protocolFactory),
	}

	err = server.server.Listen()
	if err != nil {
		panic(err)
	}
	return server
}

func (server *ApplyRequestServer) Serve() (int32, error) {
	return server.server.AcceptOnce()
}
