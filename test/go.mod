module test

go 1.17

replace github.com/uuosio/chaintester => ../

require (
	github.com/uuosio/chain v0.2.3
	github.com/uuosio/chaintester v0.0.0-20221108030052-a405ff36b294
)

require (
	github.com/apache/thrift v0.16.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
)
