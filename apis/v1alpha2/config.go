package v1alpha2

import (
	"bytes"
	"encoding/json"
)

// Values hold a map, with string as the key and either a string or a slice of strings as the value
type Values map[string]interface{}

// DeepCopy indicate how to do a deep copy of Values type
func (v *Values) DeepCopy() *Values {
	out := make(Values, len(*v))
	for key, val := range *v {
		switch val := val.(type) {
		case string:
			out[key] = val

		case []string:
			out[key] = append([]string(nil), val...)
		default:
			out[key] = val
		}
	}
	return &out
}

type Config struct {
	cfg Values `json:"-"`
}

var _ json.Marshaler = &Config{}
var _ json.Unmarshaler = &Config{}

// UnmarshalJSON implements an alternative parser for this field
func (c *Config) UnmarshalJSON(b []byte) error {
	var entries Values
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := d.Decode(&entries); err != nil {
		return err
	}
	c.cfg = entries
	return nil
}

// MarshalJSON specifies how to convert this object into JSON
func (c *Config) MarshalJSON() ([]byte, error) {
	if len(c.cfg) == 0 {
		return []byte("{}"), nil
	}

	return json.Marshal(c.cfg)
}
