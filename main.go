package main

import (
	"github.com/fzft/go-mock-redis/log"
	"github.com/fzft/go-mock-redis/node"
)

func main() {
	log.InitLogger()
	s := node.NewServer(":8080")
	s.Run()
}
