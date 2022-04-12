/*
This file contains manually added methods for our proto that are useful to the server.
*/

package agent

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

// Validate is used to validate an InstallReq.
func (i *InstallReq) Validate() error {
	i.Name = strings.TrimSpace(i.Name)
	i.Binary = strings.TrimSpace(i.Binary)
	switch "" {
	case i.Name:
		return fmt.Errorf("Name must be set")
	case i.Binary:
		return fmt.Errorf("Binary must be set")
	}
	if len(i.Package) == 0 {
		return fmt.Errorf("Package must be set")
	}
	switch {
	case !validName(i.Name):
		return fmt.Errorf("Name(%s) must only contain 0-9, A-Z, a-z", i.Name)
	case !validName(i.Binary):
		return fmt.Errorf("Binary(%s) must only contain 0-9, A-Z, a-z", i.Binary)
	}
	return nil
}

func validName(s string) bool {
	for i := 0; i < len(s); i++ {
		switch {
		// 0-9
		case s[i] >= 48 && s[i] <= 57:
		// A-Z
		case s[i] >= 65 && s[i] <= 90:
		// a-z
		case s[i] >= 97 && s[i] <= 122:
		default:
			return false
		}
	}
	return true
}

// MarshalJSON implement json.Marshaller for CPUPerfs so that we use the
// protojson.Marshal() instead of the standard marshaller.
func (x *CPUPerfs) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(x)
}

// UnmarshalJSON implement json.Unmarshaller for CPUPerfs so that we use the
// protojson.Unmarshal() instead of the standard unmarshaller.
func (x *CPUPerfs) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, x)
}

// MarshalJSON implement json.Marshaller for MemPerf so that we use the
// protojson.Marshal() instead of the standard marshaller.
func (x *MemPerf) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(x)
}

// UnmarshalJSON implement json.Unmarshaller for MemPerf so that we use the
// protojson.Unmarshal() instead of the standard unmarshaller.
func (x *MemPerf) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, x)
}
