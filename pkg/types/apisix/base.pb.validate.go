// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: base.proto

package apisix

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = ptypes.DynamicAny{}
)

// define the regex for a UUID once up-front
var _base_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on Var with the rules defined in the proto
// definition for this message. If any rules are violated, an error is returned.
func (m *Var) Validate() error {
	if m == nil {
		return nil
	}

	if l := len(m.GetVars()); l < 2 || l > 4 {
		return VarValidationError{
			field:  "Vars",
			reason: "value must contain between 2 and 4 items, inclusive",
		}
	}

	return nil
}

// VarValidationError is the validation error returned by Var.Validate if the
// designated constraints aren't met.
type VarValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e VarValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e VarValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e VarValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e VarValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e VarValidationError) ErrorName() string { return "VarValidationError" }

// Error satisfies the builtin error interface
func (e VarValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sVar.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = VarValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = VarValidationError{}
