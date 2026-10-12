package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Strings 自定义类型：string 数组，存为 JSONB
type Strings []string

// Value 实现 driver.Valuer
func (s Strings) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner
func (s *Strings) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("Strings: unsupported type")
	}
	return json.Unmarshal(bytes, s)
}

// JSONB 自定义类型：任意 JSON 对象，存为 JSONB
type JSONB map[string]interface{}

// Value 实现 driver.Valuer
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("JSONB: unsupported type")
	}
	return json.Unmarshal(bytes, j)
}

// JSONList 自定义类型：JSON 数组（[]Object），存为 JSONB
// 用于 list_view.fields 等需要存数组的字段
type JSONList []map[string]interface{}

// Value 实现 driver.Valuer
func (j JSONList) Value() (driver.Value, error) {
	if j == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner
func (j *JSONList) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("JSONList: unsupported type")
	}
	return json.Unmarshal(bytes, j)
}