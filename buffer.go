package main

type Buffer interface {
	// DataToWrite closes the connection.
	DataToWrite() []byte

	Next(n int)

	Len() int
}
