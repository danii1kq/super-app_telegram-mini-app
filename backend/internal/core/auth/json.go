package auth

import "encoding/json"

// jsonUnmarshal is a tiny indirection so initdata.go stays free of imports that
// only exist for parsing.
func jsonUnmarshal(data []byte, dst any) error {
	return json.Unmarshal(data, dst)
}
