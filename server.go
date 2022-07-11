package main

import (
	"chaintester/interfaces"
	"context"
	"crypto/tls"
	"fmt"

	"github.com/apache/thrift/lib/go/thrift"
)

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

func (p *ApplyRequestHandler) ApplyRequest(ctx context.Context, receiver *interfaces.Uint64, firstReceiver *interfaces.Uint64, action *interfaces.Uint64) (_r int32, _err error) {
	fmt.Println("+++++++ApplyRequest called!")
	apiClient, err := NewVMAPIClient("127.0.0.1:9092")
	if err != nil {
		panic(err)
	}

	_, err = apiClient.Prints(ctx, "hello, world")
	if err != nil {
		panic(err)
	}

	_, err = apiClient.EndApply(ctx)
	if err != nil {
		panic(err)
	}
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
