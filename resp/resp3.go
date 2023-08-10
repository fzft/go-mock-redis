package resp

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// RESP3 is a RESP3 protocol parser and serializer.
// https://github.com/redis/redis-specifications/blob/master/protocol/RESP3.md

// Types introduced by RESP3
const (
	TypeNull      byte = '_'
	TypeDouble    byte = ','
	TypeBoolean   byte = '#'
	TypeBlobError byte = '!'
	TypeVerbatim  byte = '='
	TypeMap       byte = '%'
	TypeSet       byte = '~'
	TypeAttribute byte = '|'
	TypePush      byte = '>'
	TypeBignum    byte = '('
)

type Double struct {
	Value float64
}

type Boolean struct {
	Value bool
}

type BlobError struct {
	Message string
}

type VerbatimString struct {
	Format string
	Value  string
}

type BigNum struct {
	Value string
}

// Array represents an array in RESP
type Array struct {
	Elements []Node
}

type Map struct {
	Elements map[Node]Node
}

type Set struct {
	Elements []Node
}

type Push struct {
	Elements []Node
}

type Attribute struct {
	Key   Node
	Value Node
}

func parseRESP(data []byte) (Node, []byte) {
	switch data[0] {
	case TypeArray: // Array
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		count, _ := strconv.Atoi(string(parts[0][1:]))
		remaining := parts[1]

		array := Array{Elements: make([]Node, count)}
		for i := 0; i < count; i++ {
			array.Elements[i], remaining = parseRESP(remaining)
		}
		return array, remaining

	case TypeBlob: // Bulk string
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		length, _ := strconv.Atoi(string(parts[0][1:]))
		end := length + 2 // Add 2 for trailing \r\n
		return BlobString{Value: string(parts[1][:length])}, parts[1][end:]

	case TypeInteger: // Number
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		num, _ := strconv.Atoi(string(parts[0][1:]))
		return Integer{Value: num}, parts[1]
	case TypeSimple: // SimpleString
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		return SimpleString{Value: string(parts[0][1:])}, parts[1]

	case TypeError: // Error
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		return Error{Message: string(parts[0][1:])}, parts[1]

	case TypeBlobError: // BlobError
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		return BlobError{Message: string(parts[0][1:])}, parts[1]
	case TypeBoolean: // Boolean
		val := data[1] == 't' || data[1] == 'T' // you can check for 'f'/'F' for false but just checking for 't'/'T' should be sufficient
		return Boolean{Value: val}, data[3:]

	case TypeVerbatim: // VerbatimString
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		totalLength, _ := strconv.Atoi(string(parts[0][1:])) // Extracting length

		// We know that the format is always 3 characters, so let's split the remaining data accordingly
		formatContentPart := parts[1]
		format := string(formatContentPart[:3])
		value := string(formatContentPart[4:totalLength]) // Subtracting 4 for "txt:" length

		// Returning remaining data after consuming the VerbatimString
		remainingData := formatContentPart[totalLength:]
		return VerbatimString{Format: format, Value: value}, remainingData
	case TypeNull: // Null value
		return Null{}, data[2:] // Consume the "_\r\n"

	case TypeDouble: // Double
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		value, _ := strconv.ParseFloat(string(parts[0][1:]), 64) // Parse the float value
		return Double{Value: value}, parts[1]

	case TypeBignum: // Bignum
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		return BigNum{Value: string(parts[0][1:])}, parts[1]

	case TypeSet: // Set
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		count, _ := strconv.Atoi(string(parts[0][1:]))
		remaining := parts[1]

		set := Set{Elements: make([]Node, count)}
		for i := 0; i < count; i++ {
			set.Elements[i], remaining = parseRESP(remaining)
		}
		return set, remaining

	case TypeMap: // Map
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		count, _ := strconv.Atoi(string(parts[0][1:]))
		remaining := parts[1]

		m := Map{Elements: make(map[Node]Node)}
		for i := 0; i < count*2; i += 2 { // Notice the change in loop condition
			key, newRemaining := parseRESP(remaining)
			value, nextRemaining := parseRESP(newRemaining)
			m.Elements[key] = value

			remaining = nextRemaining
		}
		return m, remaining

	case TypePush: // Push
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		count, _ := strconv.Atoi(string(parts[0][1:]))
		remaining := parts[1]

		array := Push{Elements: make([]Node, count)}
		for i := 0; i < count; i++ {
			array.Elements[i], remaining = parseRESP(remaining)
		}
		return array, remaining

	case TypeAttribute: // Attribute
		parts := bytes.SplitN(data, []byte(CRLF), 2)
		count, _ := strconv.Atoi(string(parts[0][1:]))
		remaining := parts[1]

		attributes := make(map[Node]Node)
		for i := 0; i < count*2; i += 2 {
			key, newRemaining := parseRESP(remaining)
			value, nextRemaining := parseRESP(newRemaining)
			attributes[key] = value
			remaining = nextRemaining
		}

		// Attributes are followed by another RESP type (the actual payload)
		payload, remaining := parseRESP(remaining)

		// You can decide how to use the attributes. Maybe you want to add them to the payload Node, or create a new Node type
		// that holds both the attributes and the payload. For this example, I'll just ignore the attributes and return the payload.
		return payload, remaining
	default:
		return nil, data
	}
}

func ConvertToRESP(command string, arguments ...string) []byte {
	totalArgs := len(arguments) + 1 // +1 for the command itself

	var builder strings.Builder

	// Array with totalArgs elements
	builder.WriteString(fmt.Sprintf("*%d%s", totalArgs, CRLF))

	// Add the command
	builder.WriteString(fmt.Sprintf("$%d%s%s%s", len(command), CRLF, command, CRLF))

	// Add the arguments
	for _, arg := range arguments {
		builder.WriteString(fmt.Sprintf("$%d%s%s%s", len(arg), CRLF, arg, CRLF))
	}

	return []byte(builder.String())
}

func printNode(node Node, indent string) {
	switch n := node.(type) {
	case SimpleString:
		fmt.Println(indent+"SimpleString:", n.Value)
	case Error:
		fmt.Println(indent+"Error:", n.Message)
	case Boolean:
		if n.Value {
			fmt.Println(indent + "Boolean: true")
		} else {
			fmt.Println(indent + "Boolean: false")
		}
	case BlobString:
		fmt.Println(indent+"BulkString:", n.Value)
	case Integer:
		fmt.Println(indent+"Integer:", n.Value)
	case Array:
		fmt.Println(indent + "Array:")
		for _, elem := range n.Elements {
			printNode(elem, indent+"  ")
		}
	case Set:
		fmt.Println(indent + "Set:")
		for _, elem := range n.Elements {
			printNode(elem, indent+"  ")
		}
	case Map:
		fmt.Println(indent + "Map:")
		for key, value := range n.Elements {
			fmt.Println(indent + "  Key:")
			printNode(key, indent+"    ")
			fmt.Println(indent + "  Value:")
			printNode(value, indent+"    ")
		}
	case VerbatimString:
		fmt.Printf("%sVerbatimString (Format: %s): %s\n", indent, n.Format, n.Value)
	case Push:
		fmt.Println(indent + "Push:")
		for _, elem := range n.Elements {
			printNode(elem, indent+"  ")
		}
	default:
		fmt.Println(indent + "Unknown Node Type!")
	}
}
