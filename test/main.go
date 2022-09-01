package main

import (
	"github.com/uuosio/chain"
)

//contract test
type Contract struct {
	receiver      chain.Name
	firstReceiver chain.Name
	action        chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	return &Contract{
		receiver,
		firstReceiver,
		action,
	}
}

//action inc
func (c *Contract) Inc(name string) {
	db := NewCounterTable(c.receiver)
	it := db.Find(1)
	payer := c.receiver
	if it.IsOk() {
		value := db.GetByIterator(it)
		value.count += 1
		db.Update(it, value, payer)
		chain.Println("count: ", value.count)
	} else {
		value := &Counter{
			key:   1,
			count: 1,
		}
		db.Store(value, payer)
		chain.Println("count: ", value.count)
	}
}

//action test
func (c *Contract) Test() {
	chain.Println("+++++++current_time:", chain.CurrentTime().Elapsed)
}

//action assert
func (c *Contract) Assert() {
	chain.Println("should panic!")
	chain.Check(false, "oops!")
}
