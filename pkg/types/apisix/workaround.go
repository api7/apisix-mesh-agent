package apisix

import "encoding/json"

// MarshalJSON implements the json.Marshaler interface.
func (v *Var) MarshalJSON() ([]byte, error) {
	if v.Vars == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(v.Vars)
}
