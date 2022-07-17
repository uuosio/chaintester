package chaintester

import "encoding/json"

type AssertError struct {
	Err error
}

func (err *AssertError) Error() string {
	return err.Err.Error()
}

func NewAssertError(err error) *AssertError {
	return &AssertError{err}
}

type TransactionError struct {
	Err []byte
}

func (t *TransactionError) Error() string {
	return string(t.Err)
}

func (t *TransactionError) Json() (*JsonValue, error) {
	value := &JsonValue{}
	// fmt.Printf("++++++push_action return: %v", string(ret))
	err := json.Unmarshal(t.Err, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func NewTransactionError(value []byte) *TransactionError {
	return &TransactionError{value}
}
