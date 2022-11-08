package chaintester

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/uuosio/chaintester/interfaces"

	"github.com/apache/thrift/lib/go/thrift"
)

type ActionArguments interfaces.ActionArguments

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

type DebuggerConfig struct {
	DebuggerServerAddress     string
	DebuggerServerPort        string
	VMAPIServerAddress        string
	VMAPIServerPort           string
	ApplyRequestServerAddress string
	ApplyRequestServerPort    string
}

var g_DebuggerConfig = DebuggerConfig{
	DebuggerServerAddress:     "127.0.0.1",
	DebuggerServerPort:        "9090",
	VMAPIServerAddress:        "127.0.0.1",
	VMAPIServerPort:           "9092",
	ApplyRequestServerAddress: "127.0.0.1",
	ApplyRequestServerPort:    "9091",
}

func GetDebuggerConfig() *DebuggerConfig {
	return &g_DebuggerConfig
}

func SetDebuggerServerAddress(addr string) {
	g_DebuggerConfig.DebuggerServerAddress = addr
}

func SetDebuggerServerPort(port string) {
	g_DebuggerConfig.DebuggerServerPort = port
}

func SetApplyRequestServerAddress(addr string) {
	g_DebuggerConfig.ApplyRequestServerAddress = addr
}

func SetApplyRequestServerPort(port string) {
	g_DebuggerConfig.ApplyRequestServerPort = port
}

func SetVMAPIServerAddress(addr string) {
	g_DebuggerConfig.VMAPIServerAddress = addr
}

func SetVMAPIServerPort(port string) {
	g_DebuggerConfig.VMAPIServerPort = port
}

func GetIPCClient() *IPCClient {
	if g_IPCClient == nil {
		addr := fmt.Sprintf("%s:%s", g_DebuggerConfig.DebuggerServerAddress, g_DebuggerConfig.DebuggerServerPort)
		iprot, oprot, err := NewProtocol(addr)
		if err != nil {
			panic(err)
		}
		g_IPCClient = NewIPCClient(iprot, oprot)
		tester := interfaces.NewIPCChainTesterClient(g_IPCClient)

		tester.InitVMAPI(defaultCtx)
		InitVMAPI() //init vm api client

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
	tester.id, err = tester.NewChain_(defaultCtx, true)
	if err != nil {
		panic(err)
	}
	return tester
}

func (p *ChainTester) SetNativeApply(apply func(uint64, uint64, uint64)) {
	g_ChainTesterApplyMap[p.id] = apply
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
	if "push_action" == method || "push_actions" == method || "produce_block" == method {
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

func (p *ChainTester) pushAction(account string, action string, arguments *interfaces.ActionArguments, permissions string) (*JsonValue, error) {
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
	ret := _result22.GetSuccess()

	value := &JsonValue{}
	// fmt.Printf("++++++push_action return: %v", string(ret))
	err := json.Unmarshal(ret, value)
	if err != nil {
		return nil, err
	}

	_, err = value.Get("except")
	if err == nil {
		return nil, NewTransactionError(ret)
	} else {
		return value, nil
	}
}

func (p *ChainTester) PushAction(account string, action string, arguments string, permissions string) (*JsonValue, error) {
	_arguments := interfaces.NewActionArguments()
	_arguments.JSONArgs_ = &arguments
	return p.pushAction(account, action, _arguments, permissions)
}

func (p *ChainTester) PushActionEx(account string, action string, arguments []byte, permissions string) (*JsonValue, error) {
	_arguments := interfaces.NewActionArguments()
	_arguments.RawArgs_ = arguments
	return p.pushAction(account, action, _arguments, permissions)
}

func (p *ChainTester) PushActions(actions []*interfaces.Action) (*JsonValue, error) {
	var _args29 interfaces.IPCChainTesterPushActionsArgs
	_args29.ID = p.id
	_args29.Actions = actions
	var _result31 interfaces.IPCChainTesterPushActionsResult
	var _meta30 thrift.ResponseMeta

	var _err error
	_meta30, _err = p.Call(defaultCtx, "push_actions", &_args29, &_result31)
	p.SetLastResponseMeta_(_meta30)
	if _err != nil {
		return nil, _err
	}
	ret := _result31.GetSuccess()
	value := &JsonValue{}
	// fmt.Printf("++++++push_action return: %v", string(ret))
	err := json.Unmarshal(ret, value)
	if err != nil {
		return nil, err
	}

	_, err = value.Get("except")
	if err == nil {
		return nil, NewTransactionError(ret)
	} else {
		return value, nil
	}
}

func (p *ChainTester) DeployContract(account string, wasmFile string, abiFile string) (err error) {
	wasm, err := os.ReadFile(wasmFile)
	if err != nil {
		return err
	}

	hexWasm := make([]byte, len(wasm)*2)
	hex.Encode(hexWasm, wasm)

	jsonArgs := fmt.Sprintf(`
		{
			"account": "%s",
			"vmtype": 0,
			"vmversion": 0,
			"code": "%s"
		}
		`,
		account,
		string(hexWasm))

	permissions := fmt.Sprintf(`
	{
		"%s": "active"
	}
	`, account)

	setCodeArgs := interfaces.NewActionArguments()
	setCodeArgs.JSONArgs_ = &jsonArgs
	setCodeAction := &interfaces.Action{
		Account:     "eosio",
		Action:      "setcode",
		Arguments:   setCodeArgs,
		Permissions: permissions,
	}
	actions := make([]*interfaces.Action, 0, 2)
	actions = append(actions, setCodeAction)

	if abiFile != "" {
		abi, err := os.ReadFile(abiFile)
		if err != nil {
			return err
		}
		rawAbi, _ := p.PackAbi(string(abi))
		hexRawAbi := make([]byte, len(rawAbi)*2)
		hex.Encode(hexRawAbi, rawAbi)
		jsonArgs := fmt.Sprintf(
			`
			{
				"account": "%s",
				"abi": "%s"
			 }
			`,
			account,
			string(hexRawAbi),
		)
		setAbiArgs := interfaces.NewActionArguments()
		setAbiArgs.JSONArgs_ = &jsonArgs
		setAbiAction := &interfaces.Action{
			Account:     "eosio",
			Action:      "setabi",
			Arguments:   setAbiArgs,
			Permissions: permissions,
		}
		actions = append(actions, setAbiAction)
	}

	_, err = p.PushActions(actions)
	if err != nil {
		return err
	}
	return nil
}

func (p *ChainTester) produceBlock(next_block_skip_seconds int64) error {
	var _args41 interfaces.IPCChainTesterProduceBlockArgs
	_args41.ID = p.id
	_args41.NextBlockSkipSeconds = next_block_skip_seconds
	var _result43 interfaces.IPCChainTesterProduceBlockResult
	var _meta42 thrift.ResponseMeta

	var _err error
	_meta42, _err = p.Call(defaultCtx, "produce_block", &_args41, &_result43)
	p.IPCChainTesterClient.SetLastResponseMeta_(_meta42)
	if _err != nil {
		return _err
	}
	return nil
}

func (p *ChainTester) ProduceBlock(next_block_skip_time ...int64) error {
	if len(next_block_skip_time) == 0 {
		return p.produceBlock(0)
	}

	if len(next_block_skip_time) != 1 {
		panic("invalid arguments")
	}
	return p.produceBlock(next_block_skip_time[0])
}

func (p *ChainTester) GetInfo() (*JsonValue, error) {
	ret, err := p.IPCChainTesterClient.GetInfo(defaultCtx, p.id)
	value := &JsonValue{}
	err = json.Unmarshal([]byte(ret), value)
	if err != nil {
		return nil, fmt.Errorf("%v", string(ret))
	}
	return value, nil
}

func (p *ChainTester) CreateKey(keyType ...string) (*JsonValue, error) {
	value := &JsonValue{}
	if len(keyType) == 0 {
		ret, err := p.IPCChainTesterClient.CreateKey(defaultCtx, "K1")
		err = json.Unmarshal([]byte(ret), value)
		if err != nil {
			return nil, fmt.Errorf("%v", string(ret))
		}
	} else {
		if len(keyType) != 1 {
			panic("Invalid keyType")
		}
		ret, err := p.IPCChainTesterClient.CreateKey(defaultCtx, keyType[0])
		err = json.Unmarshal([]byte(ret), value)
		if err != nil {
			return nil, fmt.Errorf("%v", string(ret))
		}
	}

	return value, nil
}

func (p *ChainTester) GetAccount(account string) (*JsonValue, error) {
	ret, err := p.IPCChainTesterClient.GetAccount(defaultCtx, p.id, account)
	value := &JsonValue{}
	err = json.Unmarshal([]byte(ret), value)
	if err != nil {
		return nil, fmt.Errorf("%v", string(ret))
	}
	return value, nil
}

func (p *ChainTester) CreateAccount(creator string, account string, owner_key string, active_key string, ram_bytes int64, res ...int64) (*JsonValue, error) {
	stake_net := int64(0)
	stake_cpu := int64(0)

	if len(res) != 0 {
		if len(res) != 2 {
			panic("invalid argugments")
		}
		stake_net = res[0]
		stake_cpu = res[1]
	}

	ret, err := p.IPCChainTesterClient.CreateAccount(defaultCtx, p.id, creator, account, owner_key, active_key, ram_bytes, stake_net, stake_cpu)
	value := &JsonValue{}
	err = json.Unmarshal([]byte(ret), value)
	if err != nil {
		return nil, fmt.Errorf("%v", string(ret))
	}
	return value, nil
}

func (p *ChainTester) GetTableRows(_json bool, code string, scope string, table string, lower_bound string, upper_bound string, limit int64) (*JsonValue, error) {
	return p.GetTableRowsEx(
		_json,
		code,
		scope,
		table,
		lower_bound,
		upper_bound,
		limit,
		"",
		"",
		false,
		false)
}

func (p *ChainTester) GetTableRowsEx(_json bool, code string, scope string, table string, lower_bound string, upper_bound string, limit int64, key_type string, index_position string, reverse bool, show_payer bool) (*JsonValue, error) {
	ret, err := p.IPCChainTesterClient.GetTableRows(defaultCtx,
		p.id,
		_json,
		code,
		scope,
		table,
		lower_bound,
		upper_bound,
		limit,
		key_type,
		index_position,
		reverse,
		show_payer)
	value := &JsonValue{}
	err = json.Unmarshal([]byte(ret), value)
	if err != nil {
		return nil, fmt.Errorf("%v", string(ret))
	}
	return value, nil
}

func (tester *ChainTester) GetBalance(account string, extras ...string) uint64 {
	tokenAccount := "eosio.token"
	symbol := "EOS"

	if len(extras) == 1 {
		tokenAccount = extras[0]
	}
	if len(extras) == 2 {
		symbol = extras[1]
	}

	rows, err := tester.GetTableRows(false, tokenAccount, account, "accounts", symbol, "", 1)
	if err != nil {
		panic(err)
	}
	hexString, err := rows.GetString("rows", 0)
	if err != nil {
		panic(err)
	}

	rawBalance, err := hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(rawBalance[:8])
}

func (p *ChainTester) EnableDebugContract(contract string, enable bool) error {
	err := p.IPCChainTesterClient.EnableDebugContract(defaultCtx, p.id, contract, enable)
	return err
}

func (p *ChainTester) PackAbi(abi string) ([]byte, error) {
	return p.IPCChainTesterClient.PackAbi(defaultCtx, abi)
}

func (p *ChainTester) FreeChain() (int32, error) {
	delete(g_ChainTesterApplyMap, p.id)
	return p.IPCChainTesterClient.FreeChain(defaultCtx, p.id)
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
