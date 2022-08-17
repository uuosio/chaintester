package chaintester

import (
	"context"
	"fmt"
	"testing"
)

var ctx = context.Background()

func TestPrints(t *testing.T) {
	tester := NewChainTester()
	// tester.EnableDebugContract("hello", true)
	err := tester.DeployContract("hello", "test/test.wasm", "test/test.abi")
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
	ret, err := tester.PushAction("hello", "inc", args, permissions)
	if err != nil {
		panic(err)
	}
	// t.Logf("%v", ret.ToString())
	ret, err = tester.GetTableRows(true, "hello", "", "counter", "", "", 10)
	if err != nil {
		panic(fmt.Errorf("++++++++error:%v", err))
	}
	t.Logf("%v", ret.ToString())
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
