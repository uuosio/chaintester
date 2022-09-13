package chaintester

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	traceable_errors "github.com/go-errors/errors"
)

var DEBUG = true

func newError(err error) error {
	if DEBUG {
		if _, ok := err.(*traceable_errors.Error); ok {
			return err
		}
		return traceable_errors.New(err.Error())
	} else {
		return err
	}
}

func newErrorf(format string, args ...interface{}) error {
	errMsg := fmt.Sprintf(format, args...)
	if DEBUG {
		return traceable_errors.New(errMsg)
	} else {
		return errors.New(errMsg)
	}
}

type JsonValue struct {
	value interface{}
}

func NewJsonValue(value []byte) *JsonValue {
	ret := &JsonValue{}
	err := json.Unmarshal(value, ret)
	if err == nil {
		return ret
	}
	return nil
}

func (b *JsonValue) GetValue() interface{} {
	return b.value
}

func (b *JsonValue) SetValue(value interface{}) error {
	switch value.(type) {
	case string, []JsonValue, map[string]JsonValue:
		b.value = value
		return nil
	default:
		panic("value must be a string, slice, or map")
	}
	return nil
}

func parseSubValue(subValue JsonValue) interface{} {
	switch v := subValue.value.(type) {
	case string:
		return strings.Trim(v, "\"")
	case []JsonValue, map[string]JsonValue:
		return subValue.value
	}
	return subValue
}

func (b *JsonValue) GetStringValue() (string, bool) {
	switch v := b.value.(type) {
	case string:
		return strings.Trim(v, "\""), true
	default:
		return "", false
	}
}

//return string, []JsonValue, or map[string]JsonValue
func (b *JsonValue) Get(keys ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return JsonValue{}, newErrorf("no key specified")
	}

	value := b.value
	for _, key := range keys {
		switch v := key.(type) {
		case string:
			switch v2 := value.(type) {
			case map[string]JsonValue:
				subValue, ok := v2[v]
				if !ok {
					return nil, newErrorf("key not found")
				}
				value = parseSubValue(subValue)
			case JsonValue:
				v3, ok := v2.value.(map[string]JsonValue)
				if !ok {
					return JsonValue{}, newErrorf("1:JsonValue is not a map")
				}
				subValue, ok := v3[v]
				if !ok {
					return nil, newErrorf("key not found")
				}
				value = parseSubValue(subValue)
			default:
				return JsonValue{}, newErrorf("2:JsonValue is not a map")
			}
		case int:
			var arr []JsonValue
			var ok bool
			switch v := value.(type) {
			case []JsonValue:
				arr = v
			case JsonValue:
				arr, ok = v.value.([]JsonValue)
				if !ok {
					return JsonValue{}, newErrorf("JsonValue is not an array")
				}
			default:
				return JsonValue{}, newErrorf("JsonValue is not an array")
			}
			if v < 0 || v >= len(arr) {
				return JsonValue{}, newErrorf("index out of range")
			}
			value = arr[v]
		default:
			return JsonValue{}, newErrorf("invalid key type")
		}
	}
	return value, nil
}

func (b *JsonValue) GetString(keys ...interface{}) (string, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return "", err
	}

	switch _v := v.(type) {
	case string:
		return _v, nil
	case map[string]JsonValue:
		__v, err := json.Marshal(_v)
		if err != nil {
			return "", err
		}
		return string(__v), nil
	case []JsonValue:
		__v, err := json.Marshal(_v)
		if err != nil {
			return "", err
		}
		return string(__v), nil
	default:
		return "", newErrorf("invalid json value")
	}
	return "", newErrorf("invalid json value!")
}

func (b *JsonValue) GetTime(keys ...interface{}) (*time.Time, error) {
	v, err := b.GetString(keys...)
	if err != nil {
		return nil, err
	}

	t, err := time.Parse("2006-01-02T15:04:05", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (b JsonValue) MarshalJSON() ([]byte, error) {
	switch v := b.value.(type) {
	case string:
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			return []byte(v), nil
		} else {
			return []byte(v), nil //[]byte(strconv.Quote(v)), nil
		}
	case JsonValue:
		if s, ok := v.value.(string); ok {
			return []byte(s), nil
		}
		return json.Marshal(v.value)
	case map[string]JsonValue:
		return json.Marshal(v)
	case []JsonValue:
		return json.Marshal(v)
	}
	return nil, newErrorf("bad JsonValue")
}

func (b *JsonValue) UnmarshalJSON(data []byte) error {
	// fmt.Println("+++++:UnmarshaJSON", string(data))
	if data[0] == '{' {
		m := make(map[string]JsonValue)
		err := json.Unmarshal(data, &m)
		if err != nil {
			return newError(err)
		}
		b.value = m
	} else if data[0] == '[' {
		m := make([]JsonValue, 0, 1)
		err := json.Unmarshal(data, &m)
		if err != nil {
			return newError(err)
		}
		b.value = m
	} else {
		b.value = string(data)
	}
	return nil
}

func (b *JsonValue) ToString() string {
	value, _ := json.Marshal(b.value)
	return string(value)
}
