package chaintester

import (
	"fmt"

	"github.com/uuosio/chaintester/interfaces"
)

type ActionSender struct {
	tester  *ChainTester
	actions []*interfaces.Action
}

func NewActionSender(tester *ChainTester) *ActionSender {
	return &ActionSender{tester, nil}
}

func (sender *ActionSender) AddAction(account string, action string, args string, permissions string) {
	_args := interfaces.NewActionArguments()
	_args.JSONArgs_ = &args
	_action := interfaces.Action{
		Account:     account,
		Action:      action,
		Arguments:   _args,
		Permissions: permissions,
	}
	sender.actions = append(sender.actions, &_action)
}

func (sender *ActionSender) AddActionEx(account string, action string, rawArgs []byte, permissions string) {
	_args := interfaces.NewActionArguments()
	_rawArgs := make([]byte, len(rawArgs))
	copy(_rawArgs, rawArgs)
	_args.RawArgs_ = _rawArgs
	_action := interfaces.Action{
		Account:     account,
		Action:      action,
		Arguments:   _args,
		Permissions: permissions,
	}
	sender.actions = append(sender.actions, &_action)
}

func (sender *ActionSender) AddActionWithSigner(account string, action string, jsonArgs string, signer string) {
	permissions := fmt.Sprintf(`{"%s": "active"}`, signer)
	sender.AddAction(account, action, jsonArgs, permissions)
}

func (sender *ActionSender) AddActionWithSignerEx(account string, action string, rawArgs []byte, signer string) {
	permissions := fmt.Sprintf(`{"%s": "active"}`, signer)
	sender.AddActionEx(account, action, rawArgs, permissions)
}

func (sender *ActionSender) Send() (*JsonValue, error) {
	return sender.tester.PushActions(sender.actions)
}
