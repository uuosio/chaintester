package chaintester

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"

	"github.com/uuosio/chaintester/interfaces"

	_ "unsafe"

	"github.com/apache/thrift/lib/go/thrift"
)

var g_VMAPI *interfaces.ApplyClient
var g_InApply = false

func SetInApply(inApply bool) {
	g_InApply = inApply
}

func IsInApply() bool {
	return g_InApply
}

func InitVMAPI() {
	if g_VMAPI != nil {
		return
	}

	var err error
	address := fmt.Sprintf("%s:%s", GetDebuggerConfig().VMAPIServerAddress, GetDebuggerConfig().VMAPIServerPort)
	g_VMAPI, err = NewVMAPIClient(address)
	if err != nil {
		panic(err)
	}
}

func GetVMAPI() *interfaces.ApplyClient {
	if g_VMAPI != nil {
		if !IsInApply() {
			panic("error: call vm api function out of apply context!")
		}
		return g_VMAPI
	}

	var err error
	address := fmt.Sprintf("%s:%s", GetDebuggerConfig().VMAPIServerAddress, GetDebuggerConfig().VMAPIServerPort)
	g_VMAPI, err = NewVMAPIClient(address)
	if err != nil {
		panic(err)
	}
	return g_VMAPI
}

var g_VMAPITransport thrift.TTransport

func CloseVMAPI() {
	g_VMAPITransport.Close()
}

func NewProtocol(addr string) (thrift.TProtocol, thrift.TProtocol, error) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(nil)
	// protocolFactory := thrift.NewTCompactProtocolFactoryConf(nil)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)

	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	g_VMAPITransport = thrift.NewTSocketConf(addr, cfg)
	g_VMAPITransport, err := transportFactory.GetTransport(g_VMAPITransport)
	if err != nil {
		return nil, nil, err
	}
	// defer transport.Close()
	if err := g_VMAPITransport.Open(); err != nil {
		return nil, nil, err
	}
	iprot := protocolFactory.GetProtocol(g_VMAPITransport)
	oprot := protocolFactory.GetProtocol(g_VMAPITransport)
	return iprot, oprot, nil
}

func NewVMAPIClient(addr string) (*interfaces.ApplyClient, error) {
	iprot, oprot, err := NewProtocol(addr)
	if err != nil {
		return nil, err
	}
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
	return binary.LittleEndian.Uint64(value.RawValue)
}

var g_ChainTesterApplyMap = make(map[int32]map[string]func(uint64, uint64, uint64))

func (p *ApplyRequestHandler) ApplyRequest(ctx context.Context, receiver *interfaces.Uint64, firstReceiver *interfaces.Uint64, action *interfaces.Uint64, chainTesterId int32) (_r int32, _err error) {
	// fmt.Println("+++++++ApplyRequest called!")
	defer func() {
		if err := recover(); err != nil {
			_, ok := err.(*AssertError)
			if ok {
				// fmt.Printf("recovered from assertion error: %v", _r)
			} else {
				// fmt.Printf("recovered error: %s", err)
				_err = fmt.Errorf("%v", err)
				_r = -1
			}
			GetVMAPI().EndApply(ctx)
			SetInApply(false)
		}
	}()

	_receiver := getUint64(receiver)
	_firstReceiver := getUint64(firstReceiver)
	_action := getUint64(action)

	SetInApply(true)
	if applyMap, ok := g_ChainTesterApplyMap[chainTesterId]; ok {
		if apply, ok := applyMap[N2S(_receiver)]; ok {
			apply(_receiver, _firstReceiver, _action)
		}
	}
	GetVMAPI().EndApply(ctx)
	SetInApply(false)

	_r = -1
	_err = nil
	return
}

func (p *ApplyRequestHandler) ApplyEnd(ctx context.Context, chainTesterId int32) (_r int32, _err error) {
	// fmt.Println("+++++++ApplyEnd")
	GetApplyRequestServer().server.EndProcessRequests()
	return 1, nil
}

type ApplyRequestServer struct {
	server *SimpleIPCServer
}

func NewApplyRequestServer() *ApplyRequestServer {
	var transport thrift.TServerTransport
	var err error
	addr := fmt.Sprintf("%s:%s", g_DebuggerConfig.ApplyRequestServerAddress, g_DebuggerConfig.ApplyRequestServerPort)
	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		return nil
	}

	handler := NewApplyRequestHandler()
	processor := interfaces.NewApplyRequestProcessor(handler)

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(nil)
	// protocolFactory := thrift.NewTCompactProtocolFactoryConf(nil)

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

func (server *ApplyRequestServer) AcceptOnce() (int32, error) {
	return server.server.AcceptOnce()
}

func (server *ApplyRequestServer) Serve() (int32, error) {
	// fmt.Println("+++++++ApplyRequestServer:ProcessRequests")
	return server.server.ProcessRequests()
}

func (server *ApplyRequestServer) Stop() error {
	fmt.Println("+++++++ApplyRequestServer:Stop")
	return server.server.Stop()
}
