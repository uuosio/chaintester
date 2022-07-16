package chaintester

import (
	"context"
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
	client *IPCClient
	id     int32
}

var g_ApplyRequestServer *ApplyRequestServer

func GetApplyRequestServer() *ApplyRequestServer {
	if g_ApplyRequestServer == nil {
		g_ApplyRequestServer = NewApplyRequestServer()
		g_ApplyRequestServer.AcceptOnce()
	}
	return g_ApplyRequestServer
}

var g_IPCClient *IPCClient = nil

func GetIPCClient() *IPCClient {
	if g_IPCClient == nil {
		iprot, oprot, err := NewProtocol("127.0.0.1:9090")
		if err != nil {
			return nil
		}
		g_IPCClient = NewIPCClient(iprot, oprot)
		tester := interfaces.NewIPCChainTesterClient(g_IPCClient)

		tester.InitVMAPI(defaultCtx)
		GetVMAPI() //init vm api client

		tester.InitApplyRequest(defaultCtx)
		GetApplyRequestServer() // init apply request server

	}
	return g_IPCClient
}

// cannot use c (variable of type *IPCClient) as thrift.TClient value in argument to interfaces.NewIPCChainTesterClient: wrong type for method Call (have
// 	func(ctx context.Context, method string, args github.com/apache/thrift/lib/go/thrift.TStruct, result github.com/apache/thrift/lib/go/thrift.TStruct) (chaintester.ResponseMeta, error), want
// 	func(ctx context.Context, method string, args github.com/apache/thrift/lib/go/thrift.TStruct, result github.com/apache/thrift/lib/go/thrift.TStruct) (github.com/apache/thrift/lib/go/thrift.ResponseMeta, error))compilerInvalidIfaceAssign

func NewChainTester() *ChainTester {
	c := GetIPCClient()

	tester := &ChainTester{
		IPCChainTesterClient: *interfaces.NewIPCChainTesterClient(c),
		client:               c,
	}

	var err error
	tester.id, err = tester.NewChain_(defaultCtx)
	if err != nil {
		panic(err)
	}
	return tester
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
	if "push_action" == method {
		GetApplyRequestServer().Serve()
	}

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

func (p *ChainTester) PushAction(account string, action string, arguments string, permissions string) []byte {
	var _args20 interfaces.IPCChainTesterPushActionArgs
	_args20.ID = p.id
	_args20.Account = account
	_args20.Action = action
	_args20.Arguments = arguments
	_args20.Permissions = permissions
	var _result22 interfaces.IPCChainTesterPushActionResult
	var _meta21 thrift.ResponseMeta
	var _err error
	_meta21, _err = p.Call(defaultCtx, "push_action", &_args20, &_result22)
	if _err != nil {
		panic(_err)
	}
	p.IPCChainTesterClient.SetLastResponseMeta_(_meta21)
	return _result22.GetSuccess()
}

func (p *ChainTester) EnableDebugContract(contract string, enable bool) error {
	err := p.IPCChainTesterClient.EnableDebugContract(defaultCtx, p.id, contract, enable)
	return err
}

func (p *ChainTester) PackAbi(abi string) ([]byte, error) {
	return p.IPCChainTesterClient.PackAbi(defaultCtx, abi)
}

func (p *ChainTester) FreeChain() (int32, error) {
	return p.IPCChainTesterClient.FreeChain(defaultCtx, p.id)
}

func (p *ChainTester) ProduceBlock() error {
	return p.IPCChainTesterClient.ProduceBlock(defaultCtx, p.id)
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
	client.PushAction("hello", "sayhello", args, permissions)
	return nil
}
