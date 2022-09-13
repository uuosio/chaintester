package chaintester

import (
	"context"
	"fmt"
	"testing"
)

var ctx = context.Background()

func TestChainTester(t *testing.T) {
	tester := NewChainTester()
	info, _ := tester.GetInfo()
	t.Logf("+++++++++info: %v", info)

	key, err := tester.CreateKey()
	if err != nil {
		panic(err)
	}

	t.Logf("+++++++++key: %v", key)

	privKey, _ := key.GetString("private")
	t.Logf("+++++++++private key: %v", privKey)

	pubKey, _ := key.GetString("public")
	t.Logf("+++++++++public key: %v", pubKey)

	_, err = tester.CreateAccount("hello", "helloworld33", pubKey, pubKey, 10*1024*1024, 10*10000, 10*10000)
	if err != nil {
		panic(err)
	}

	ret, _ := tester.GetAccount("helloworld33")
	t.Logf("+++++++++account info: %v", ret)

	// tester.EnableDebugContract("hello", true)
	err = tester.DeployContract("hello", "test/test.wasm", "test/test.abi")
	if err != nil {
		panic(err)
	}
	tester.ProduceBlock()

	permissions := `
	{
		"hello": "active"
	}
	`
	args := `
	{
		"name": "go"
	}
	`
	ret, err = tester.PushAction("hello", "inc", args, permissions)
	if err != nil {
		panic(err)
	}
	// t.Logf("%v", ret.ToString())
	ret, err = tester.GetTableRows(true, "hello", "", "counter", "", "", 10)
	if err != nil {
		panic(fmt.Errorf("++++++++error:%v", err))
	}
	t.Logf("%v", ret.ToString())

	ret, err = tester.GetTableRows(false, "eosio.token", "hello", "accounts", "EOS", "", 1)
	if err != nil {
		panic(fmt.Errorf("++++++++error:%v", err))
	}

	balance, err := ret.GetString("rows", 0)
	if err != nil {
		panic(err)
	}
	t.Logf("++++++++++=raw balance: %s", balance)

	ret, err = tester.PushAction("hello", "test", "", permissions)
	if err != nil {
		panic(err)
	}
	tester.ProduceBlock(10)

	ret, err = tester.PushAction("hello", "test", "", permissions)
	if err != nil {
		panic(err)
	}
	tester.ProduceBlock()
}

func OnApply(receiver, firstReceiver, action uint64) {
	native_apply(receiver, firstReceiver, action)
}

func init() {
	SetApplyFunc(OnApply)
}

func native_apply(receiver uint64, firstReceiver uint64, action uint64) {
	for i := 0; i < 10; i++ {
		GetVMAPI().Prints(ctx, "hello, world!\n")
	}
}
