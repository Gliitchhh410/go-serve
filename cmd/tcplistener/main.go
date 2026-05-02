package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Println("Listening on :42069")
	pool := NewWorkerPool(4, 100)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}

		select {
		case pool.conns <- conn:
		default:
			log.Printf("Server saturated, dropping connection from %s\n", conn.RemoteAddr())
			conn.Write(response503)
			conn.Close()
		}
	}
}
