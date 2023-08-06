package main

func main() {
	InitLogger()
	s := NewServer(":8080")
	s.Run()
}
