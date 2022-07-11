package chaintester

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
)

func TestPrints(t *testing.T) {

	var protocolFactory thrift.TProtocolFactory
	protocolFactory = thrift.NewTBinaryProtocolFactoryConf(nil)

	var transportFactory thrift.TTransportFactory
	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	transportFactory = thrift.NewTBufferedTransportFactory(8192)

	if err := runClient(transportFactory, protocolFactory, "127.0.0.1:9090", false, cfg); err != nil {
		fmt.Println("error running client:", err)
	}
}

func native_apply(receiver uint64, firstReceiver uint64, action uint64) {
	apiClient, err := NewVMAPIClient("127.0.0.1:9092")
	if err != nil {
		panic(err)
	}

	var ctx = context.Background()

	_, err = apiClient.Prints(ctx, "hello, world")
	if err != nil {
		panic(err)
	}

	_, err = apiClient.EndApply(ctx)
	if err != nil {
		panic(err)
	}

}
