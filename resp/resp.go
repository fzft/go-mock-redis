package resp

const CRLF string = "\r\n"

// Types equivalent to RESP version 2
const (
	TypeArray   byte = '*'
	TypeBlob    byte = '$'
	TypeSimple  byte = '+'
	TypeError   byte = '-'
	TypeInteger byte = ':'
)

type Node interface {
}

type BlobString struct {
	Value string
}

type SimpleString struct {
	Value string
}

type Error struct {
	Message string
}

type Integer struct {
	Value int
}

type Null struct {
}
