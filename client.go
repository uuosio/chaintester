package chaintester

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"

	"github.com/learnforpractice/chaintester/interfaces"

	"github.com/apache/thrift/lib/go/thrift"
)

type IPCClient struct {
	seqId        int32
	iprot, oprot thrift.TProtocol
}

// IPCClient implements TClient, and uses the standard message format for Thrift.
// It is not safe for concurrent use.
func NewIPCClient(inputProtocol, outputProtocol thrift.TProtocol) *IPCClient {
	return &IPCClient{
		iprot: inputProtocol,
		oprot: outputProtocol,
	}
}

func (p *IPCClient) Send(ctx context.Context, oprot thrift.TProtocol, seqId int32, method string, args thrift.TStruct) error {
	// Set headers from context object on THeaderProtocol
	if headerProt, ok := oprot.(*thrift.THeaderProtocol); ok {
		headerProt.ClearWriteHeaders()
		for _, key := range thrift.GetWriteHeaderList(ctx) {
			if value, ok := thrift.GetHeader(ctx, key); ok {
				headerProt.SetWriteHeader(key, value)
			}
		}
	}

	if err := oprot.WriteMessageBegin(ctx, method, thrift.CALL, seqId); err != nil {
		return err
	}
	if err := args.Write(ctx, oprot); err != nil {
		return err
	}
	if err := oprot.WriteMessageEnd(ctx); err != nil {
		return err
	}
	return oprot.Flush(ctx)
}

func (p *IPCClient) Recv(ctx context.Context, iprot thrift.TProtocol, seqId int32, method string, result thrift.TStruct) error {
	rMethod, rTypeId, rSeqId, err := iprot.ReadMessageBegin(ctx)
	if err != nil {
		return err
	}

	if method != rMethod {
		return thrift.NewTApplicationException(thrift.WRONG_METHOD_NAME, fmt.Sprintf("%s: wrong method name", method))
	} else if seqId != rSeqId {
		return thrift.NewTApplicationException(thrift.BAD_SEQUENCE_ID, fmt.Sprintf("%s: out of order sequence response", method))
	} else if rTypeId == thrift.EXCEPTION {
		exception := thrift.NewTApplicationException(0, "")
		if err := exception.Read(ctx, iprot); err != nil {
			return err
		}

		if err := iprot.ReadMessageEnd(ctx); err != nil {
			return err
		}

		return exception
	} else if rTypeId != thrift.REPLY {
		return thrift.NewTApplicationException(thrift.INVALID_MESSAGE_TYPE_EXCEPTION, fmt.Sprintf("%s: invalid message type", method))
	}

	if err := result.Read(ctx, iprot); err != nil {
		return err
	}

	return iprot.ReadMessageEnd(ctx)
}

func (p *IPCClient) Call(ctx context.Context, method string, args, result thrift.TStruct) (thrift.ResponseMeta, error) {
	p.seqId++
	seqId := p.seqId

	if err := p.Send(ctx, p.oprot, seqId, method, args); err != nil {
		return thrift.ResponseMeta{}, err
	}

	// method is oneway
	if result == nil {
		return thrift.ResponseMeta{}, nil
	}

	err := p.Recv(ctx, p.iprot, seqId, method, result)
	var headers thrift.THeaderMap
	if hp, ok := p.iprot.(*thrift.THeaderProtocol); ok {
		transport := reflect.ValueOf(hp).Elem().FieldByName("transport")
		readHeaders := reflect.ValueOf(transport).Elem().FieldByName("readHeaders")
		headers = readHeaders.Interface().(thrift.THeaderMap)
	}
	return thrift.ResponseMeta{
		Headers: headers,
	}, err
}

var defaultCtx = context.Background()

type ChainTester struct {
	interfaces.IPCChainTesterClient
	client             *IPCClient
	applyRequestServer *ApplyRequestServer
}

// cannot use c (variable of type *IPCClient) as thrift.TClient value in argument to interfaces.NewIPCChainTesterClient: wrong type for method Call (have
// 	func(ctx context.Context, method string, args github.com/apache/thrift/lib/go/thrift.TStruct, result github.com/apache/thrift/lib/go/thrift.TStruct) (chaintester.ResponseMeta, error), want
// 	func(ctx context.Context, method string, args github.com/apache/thrift/lib/go/thrift.TStruct, result github.com/apache/thrift/lib/go/thrift.TStruct) (github.com/apache/thrift/lib/go/thrift.ResponseMeta, error))compilerInvalidIfaceAssign

func NewChainTester() *ChainTester {
	var protocolFactory thrift.TProtocolFactory
	protocolFactory = thrift.NewTBinaryProtocolFactoryConf(nil)

	var transportFactory thrift.TTransportFactory
	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	transportFactory = thrift.NewTBufferedTransportFactory(8192)

	var transport thrift.TTransport

	transport = thrift.NewTSocketConf(addr, cfg)
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return nil, err
	}
	defer transport.Close()
	if err := transport.Open(); err != nil {
		return nil, err
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	c := NewIPCClient(iprot, oprot)

	return &ChainTester{
		IPCChainTesterClient: *interfaces.NewIPCChainTesterClient(c),
		client:               c,
		applyRequestServer:   NewApplyRequestServer(),
	}
}

func (p *ChainTester) Call(ctx context.Context, method string, args, result thrift.TStruct) (thrift.ResponseMeta, error) {
	p.client.seqId++
	seqId := p.client.seqId

	if err := p.client.Send(ctx, p.client.oprot, seqId, method, args); err != nil {
		return thrift.ResponseMeta{}, err
	}

	// method is oneway
	if result == nil {
		return thrift.ResponseMeta{}, nil
	}

	//runApplyRequestServer

	//start apply request server
	p.applyRequestServer.Serve()

	err := p.client.Recv(ctx, p.client.iprot, seqId, method, result)
	var headers thrift.THeaderMap
	if hp, ok := p.client.iprot.(*thrift.THeaderProtocol); ok {
		transport := reflect.ValueOf(hp).Elem().FieldByName("transport")
		readHeaders := reflect.ValueOf(transport).Elem().FieldByName("readHeaders")
		headers = readHeaders.Interface().(thrift.THeaderMap)
	}
	return thrift.ResponseMeta{
		Headers: headers,
	}, err
}

func (p *ChainTester) PushAction(ctx context.Context, id int32, account string, action string, arguments string, permissions string) (_r int32, _err error) {
	var _args6 interfaces.IPCChainTesterPushActionArgs
	_args6.ID = id
	_args6.Account = account
	_args6.Action = action
	_args6.Arguments = arguments
	_args6.Permissions = permissions
	var _result8 interfaces.IPCChainTesterPushActionResult
	var _meta7 thrift.ResponseMeta
	_meta7, _err = p.Call(ctx, "push_action", &_args6, &_result8)
	p.SetLastResponseMeta_(_meta7)
	if _err != nil {
		return
	}
	return _result8.GetSuccess(), nil
}

func handleClient(client *ChainTester) (err error) {
	args := `
	{
		"name": "go"
	}
	`
	permissions := `
	{
		"hello": "active"
	}
	`

	// id, err := client.NewChain_(defaultCtx)
	// if err != nil {
	// 	return err
	// }
	id := int32(0)
	client.PushAction(defaultCtx, id, "hello", "sayhello", args, permissions)
	return nil
}
