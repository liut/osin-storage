package sqlstore

import (
	"database/sql/driver"
	"encoding/json"
)

type JsonKV map[string]interface{}

func ToJsonKV(src interface{}) (JsonKV, error) {
	switch s := src.(type) {
	case JsonKV:
		return s, nil
	case map[string]interface{}:
		return JsonKV(s), nil
	}
	return nil, ErrInvalidJSON
}

func (m JsonKV) WithKey(key string) (v interface{}) {
	var ok bool
	if v, ok = m[key]; ok {
		return
	}
	return
}

// Scan implements the sql.Scanner interface.
func (m *JsonKV) Scan(value interface{}) (err error) {
	switch data := value.(type) {
	case JsonKV:
		*m = data
	case map[string]interface{}:
		*m = JsonKV(data)
	case []byte:
		err = json.Unmarshal(data, m)
	case string:
		err = json.Unmarshal([]byte(data), m)
	}
	return
}

// Value implements the driver.Valuer interface.
func (m JsonKV) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// ClientMeta ...
type ClientMeta struct {
	Name          string   `json:"name"`
	GrantTypes    []string `json:"grant_types,omitempty"`    // AllowedGrantTypes
	ResponseTypes []string `json:"response_types,omitempty"` // AllowedResponseTypes
	Scopes        []string `json:"scopes,omitempty"`         // AllowedScopes
}

// Scan implements the sql.Scanner interface.
func (m *ClientMeta) Scan(value interface{}) (err error) {
	switch data := value.(type) {
	case ClientMeta:
		*m = data
	case []byte:
		err = json.Unmarshal(data, m)
	case string:
		err = json.Unmarshal([]byte(data), m)
	}
	return
}

// Value implements the driver.Valuer interface.
func (m ClientMeta) Value() (driver.Value, error) {
	return json.Marshal(m)
}
