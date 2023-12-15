// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha2

import (
	"bytes"
	"encoding/json"
)

// Values represent parts of the config.
type Values map[string]interface{}

// DeepCopy indicate how to do a deep copy of Values type.
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

// Config encapsulates collector config.
type Config struct {
	cfg Values `json:"-"`
}

var _ json.Marshaler = &Config{}
var _ json.Unmarshaler = &Config{}

// UnmarshalJSON implements an alternative parser for this field.
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

// MarshalJSON specifies how to convert this object into JSON.
func (c *Config) MarshalJSON() ([]byte, error) {
	if len(c.cfg) == 0 {
		return []byte("{}"), nil
	}

	return json.Marshal(c.cfg)
}
