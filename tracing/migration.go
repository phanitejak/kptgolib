package tracing

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// The contents of this file is used to convert deprecated Opentracing library types into OpenTelemetry supported types.

// Field represents a deprecated opentracing field structure.
type Field struct {
	key          string
	fieldType    fieldType
	numericVal   int64
	stringVal    string
	interfaceVal any
}

type fieldType int

const (
	stringType fieldType = iota
	boolType
	intType
	errorType
	objectType
)

// String adds a string-valued key:value pair to a Span.LogFields() record.
func String(key, val string) Field {
	return Field{
		key:       key,
		fieldType: stringType,
		stringVal: val,
	}
}

// Bool adds a bool-valued key:value pair to a Span.LogFields() record.
func Bool(key string, val bool) Field {
	var numericVal int64
	if val {
		numericVal = 1
	}
	return Field{
		key:        key,
		fieldType:  boolType,
		numericVal: numericVal,
	}
}

// Int adds an int-valued key:value pair to a Span.LogFields() record.
func Int(key string, val int) Field {
	return Field{
		key:        key,
		fieldType:  intType,
		numericVal: int64(val),
	}
}

// Error adds an error with the key "error.object" to a Span.LogFields() record.
func Error(err error) Field {
	return Field{
		key:          "error.object",
		fieldType:    errorType,
		interfaceVal: err,
	}
}

// Object adds an object-valued key:value pair to a Span.LogFields() record
// Please pass in an immutable object, otherwise there may be concurrency issues.
// Such as passing in the map, log.Object may result in "fatal error: concurrent map iteration and map write".
// Because span is sent asynchronously, it is possible that this map will also be modified.
func Object(key string, obj any) Field {
	return Field{
		key:          key,
		fieldType:    objectType,
		interfaceVal: obj,
	}
}

func legacyLogFieldsToAttributes(fields []Field) []attribute.KeyValue {
	encoder := &bridgeFieldEncoder{}
	for _, field := range fields {
		field.Marshal(encoder)
	}
	return encoder.pairs
}

type bridgeFieldEncoder struct {
	pairs []attribute.KeyValue
}

var _ Encoder = &bridgeFieldEncoder{}

func (e *bridgeFieldEncoder) EmitString(key, value string) {
	e.emitCommon(key, value)
}

func (e *bridgeFieldEncoder) EmitBool(key string, value bool) {
	e.emitCommon(key, value)
}

func (e *bridgeFieldEncoder) EmitInt(key string, value int) {
	e.emitCommon(key, value)
}

func (e *bridgeFieldEncoder) EmitObject(key string, value any) {
	e.emitCommon(key, value)
}

func (e *bridgeFieldEncoder) emitCommon(key string, value any) {
	e.pairs = append(e.pairs, KeyValueToAttribute(key, value))
}

// Encoder allows access to the contents of a Field (via a call to
// Field.Marshal).
//
// Tracer implementations typically provide an implementation of Encoder;
// Tracing callers typically do not need to concern themselves with it.
type Encoder interface {
	EmitString(key, value string)
	EmitBool(key string, value bool)
	EmitInt(key string, value int)
	EmitObject(key string, value any)
}

// Marshal passes a Field instance through to the appropriate
// field-type-specific method of an Encoder.
func (lf Field) Marshal(visitor Encoder) {
	switch lf.fieldType {
	case stringType:
		visitor.EmitString(lf.key, lf.stringVal)
	case boolType:
		visitor.EmitBool(lf.key, lf.numericVal != 0)
	case intType:
		visitor.EmitInt(lf.key, int(lf.numericVal))
	case errorType:
		if err, ok := lf.interfaceVal.(error); ok {
			visitor.EmitString(lf.key, err.Error())
		} else {
			visitor.EmitString(lf.key, "<invalid error value>")
		}
	case objectType:
		visitor.EmitObject(lf.key, lf.interfaceVal)
	default:
		// intentionally left blank
	}
}

// Key returns the field's key.
func (lf Field) Key() string {
	return lf.key
}

// Value returns the field's value as any.
func (lf Field) Value() any {
	switch lf.fieldType {
	case stringType:
		return lf.stringVal
	case boolType:
		return lf.numericVal != 0
	case intType:
		return int(lf.numericVal)
	case errorType, objectType:
		return lf.interfaceVal
	default:
		return nil
	}
}

// String returns a string representation of the key and value.
func (lf Field) String() string {
	return fmt.Sprint(lf.Key(), ":", lf.Value())
}

// KeyValueToAttribute ...
func KeyValueToAttribute(k string, v any) attribute.KeyValue {
	key := stringToAttributeKey(k)
	switch val := v.(type) {
	case bool:
		return key.Bool(val)
	case int64:
		return key.Int64(val)
	case uint64:
		return key.String(fmt.Sprintf("%d", val))
	case float64:
		return key.Float64(val)
	case int32:
		return key.Int64(int64(val))
	case uint32:
		return key.Int64(int64(val))
	case float32:
		return key.Float64(float64(val))
	case int:
		return key.Int(val)
	case uint:
		return key.String(fmt.Sprintf("%d", val))
	case string:
		return key.String(val)
	default:
		return key.String(fmt.Sprint(v))
	}
}

func stringToAttributeKey(k string) attribute.Key {
	return attribute.Key(k)
}
