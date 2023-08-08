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
