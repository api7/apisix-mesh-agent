package types

import json "google.golang.org/protobuf/encoding/protojson"

var (
	JsonOpts = json.MarshalOptions{UseEnumNumbers: true}
)
