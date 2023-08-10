package resp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRESPSimpleString(t *testing.T) {
	resp := []byte("+OK\r\n")
	node, _ := parseRESP(resp)
	expected := SimpleString{Value: "OK"}
	assert.Equal(t, expected, node)
}

func TestParseRESPError(t *testing.T) {
	resp := []byte("-Error message\r\n")
	node, _ := parseRESP(resp)
	expected := Error{Message: "Error message"}
	assert.Equal(t, expected, node)
}

func TestParseRESPBoolean(t *testing.T) {
	resp := []byte("#t\r\n")
	node, _ := parseRESP(resp)
	expected := Boolean{Value: true}
	assert.Equal(t, expected, node)
}

func TestParseRESPSet(t *testing.T) {
	resp := []byte("~2\r\n+First\r\n+Second\r\n")
	node, _ := parseRESP(resp)
	expected := Set{Elements: []Node{SimpleString{Value: "First"}, SimpleString{Value: "Second"}}}
	assert.Equal(t, expected, node)
}

func TestParseRESPMap(t *testing.T) {
	resp := []byte("%2\r\n+Key1\r\n+Value1\r\n+Key2\r\n+Value2\r\n")
	node, _ := parseRESP(resp)
	expected := Map{Elements: map[Node]Node{
		SimpleString{Value: "Key1"}: SimpleString{Value: "Value1"},
		SimpleString{Value: "Key2"}: SimpleString{Value: "Value2"},
	}}

	assert.Equal(t, expected, node)
}

func TestParseRESPVerbatimString(t *testing.T) {
	resp := []byte("=15\r\nfmt:Hello World\r\n")
	node, _ := parseRESP(resp)
	expected := VerbatimString{Format: "fmt", Value: "Hello World"}
	assert.Equal(t, expected, node)
}

func TestParseRESPNull(t *testing.T) {
	resp := []byte("_\r\n")
	node, _ := parseRESP(resp)
	expected := Null{}
	assert.Equal(t, expected, node)
}

func TestParseRESPDouble(t *testing.T) {
	resp := []byte(",3.14159\r\n")
	node, _ := parseRESP(resp)
	expected := Double{Value: 3.14159}
	assert.Equal(t, expected, node)
}

func TestParseRESPBignum(t *testing.T) {
	resp := []byte("(123456789012345678901234567890\r\n") // an arbitrary big number
	node, _ := parseRESP(resp)
	expected := BigNum{Value: "123456789012345678901234567890"}
	assert.Equal(t, expected, node)
}

/**
*5\r\n
+Hello\r\n
-ErrorText\r\n
:1234\r\n
,3.14159\r\n
%2\r\n
+Key1\r\n
+Value1\r\n
+Key2\r\n
~3\r\n
+SetValue1\r\n
+SetValue2\r\n
+SetValue3\r\n

*/

func TestParseRESPComplex(t *testing.T) {
	resp := []byte("*5\r\n+Hello\r\n-ErrorText\r\n:1234\r\n,3.14159\r\n%2\r\n+Key1\r\n+Value1\r\n+Key2\r\n~3\r\n+SetValue1\r\n+SetValue2\r\n+SetValue3\r\n")
	node, _ := parseRESP(resp)

	expectedSetValue := []Node{
		SimpleString{Value: "SetValue1"},
		SimpleString{Value: "SetValue2"},
		SimpleString{Value: "SetValue3"},
	}

	expectedMap := Map{
		Elements: map[Node]Node{
			SimpleString{Value: "Key1"}: SimpleString{Value: "Value1"},
			SimpleString{Value: "Key2"}: Set{Elements: expectedSetValue},
		},
	}

	expected := Array{
		Elements: []Node{
			SimpleString{Value: "Hello"},
			Error{Message: "ErrorText"},
			Integer{Value: 1234},
			Double{Value: 3.14159},
			expectedMap,
		},
	}

	assert.Equal(t, expected, node)
}

func TestConvertToRESPSet(t *testing.T) {
	command := ConvertToRESP("SET", "key", "value")
	expected := []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n")
	assert.Equal(t, expected, command)
}

func TestConvertToRESPGet(t *testing.T) {
	command := ConvertToRESP("GET", "key")
	expected := []byte("*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n")
	assert.Equal(t, expected, command)
}

func TestConvertToRESPZAdd(t *testing.T) {
	command := ConvertToRESP("ZADD", "myzset", "1", "one")
	expected := []byte("*4\r\n$4\r\nZADD\r\n$6\r\nmyzset\r\n$1\r\n1\r\n$3\r\none\r\n")
	assert.Equal(t, expected, command)
}

func TestConvertToRESPZAddMultiple(t *testing.T) {
	command := ConvertToRESP("ZADD", "myzset", "1", "one", "2", "two")
	expected := []byte("*6\r\n$4\r\nZADD\r\n$6\r\nmyzset\r\n$1\r\n1\r\n$3\r\none\r\n$1\r\n2\r\n$3\r\ntwo\r\n")
	assert.Equal(t, expected, command)
}

//func TestParseRESPAttribute(t *testing.T) {
//	// An example that includes an attribute followed by a simple string.
//	resp := []byte("|1\r\n+key\r\n+value\r\n+Hello world\r\n")
//	node, _ := parseRESP(resp)
//
//	// The expected output is an attribute followed by a simple string.
//	expected := []Node{
//		Attribute{Key: SimpleString{Value: "key"}, Value: SimpleString{Value: "value"}},
//		SimpleString{Value: "Hello world"},
//	}
//	assert.Equal(t, expected, node)
//}
//
//func TestParseRESPMultipleAttributes(t *testing.T) {
//	// An example with multiple attributes followed by a simple string.
//	resp := []byte("|2\r\n+key1\r\n+value1\r\n+key2\r\n+value2\r\n+Payload\r\n")
//	node, _ := parseRESP(resp)
//
//	expected := []Node{
//		Attribute{Key: SimpleString{Value: "key1"}, Value: SimpleString{Value: "value1"}},
//		Attribute{Key: SimpleString{Value: "key2"}, Value: SimpleString{Value: "value2"}},
//		SimpleString{Value: "Payload"},
//	}
//	assert.Equal(t, expected, node)
//}
//
//func TestParseRESPAttributeWithArrayPayload(t *testing.T) {
//	// An example that includes an attribute followed by an array.
//	resp := []byte("|1\r\n+annotation\r\n+This is an attribute\r\n*2\r\n+Element1\r\n+Element2\r\n")
//	node, _ := parseRESP(resp)
//
//	expected := []Node{
//		Attribute{Key: SimpleString{Value: "annotation"}, Value: SimpleString{Value: "This is an attribute"}},
//		Array{
//			Elements: []Node{
//				SimpleString{Value: "Element1"},
//				SimpleString{Value: "Element2"},
//			},
//		},
//	}
//	assert.Equal(t, expected, node)
//}
//
//func TestParseRESPComplexAttribute(t *testing.T) {
//	resp := []byte("|1\r\n+key-popularity\r\n%2\r\n$1\r\na\r\n,0.1923\r\n$1\r\nb\r\n,0.0012\r\n*2\r\n:2039123\r\n:9543892\r\n")
//	node, _ := parseRESP(resp)
//
//	expected := []Node{
//		Attribute{
//			Key: SimpleString{Value: "key-popularity"},
//			Value: Map{
//				Elements: map[Node]Node{
//					BlobString{Value: "a"}: Double{Value: 0.1923},
//					BlobString{Value: "b"}: Double{Value: 0.0012},
//				},
//			},
//		},
//		Array{
//			Elements: []Node{
//				Integer{Value: 2039123},
//				Integer{Value: 9543892},
//			},
//		},
//	}
//	assert.Equal(t, expected, node)
//}
