package main

import (
	"crypto/tls"
	"fmt"

	"github.com/apache/thrift/lib/go/thrift"
)

func main() {

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
