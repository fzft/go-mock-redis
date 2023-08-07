package main

// RESP3 is a RESP3 protocol parser and serializer.
// https://github.com/redis/redis-specifications/blob/master/protocol/RESP3.md

type RESP3Type byte

const (
	SimpleString RESP3Type = '+'
	Error        RESP3Type = '-'
	Integer      RESP3Type = ':'
	BlobString   RESP3Type = '$'
	Array        RESP3Type = '*'
	Map          RESP3Type = '%'
	Set          RESP3Type = '~'
)
