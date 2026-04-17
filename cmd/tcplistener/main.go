package main

import (
	"fmt"
	"httpServer/internal/request"
	"log"
	"net"
)
func main(){
	listener, err := net.Listen("tcp", ":42069")

	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	log.Println("Listening on :42069")

	for {
		conn, err := listener.Accept()
		
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}
		
		log.Printf("Accepted connection from %s\n", conn.RemoteAddr().String())

		req ,err := request.RequestFromReader(conn)
		if err != nil {
			log.Printf("Error parsing request: %v\n", err)
			conn.Close()
			continue
		}
		fmt.Printf("read: %s", req.Line.Method)
		fmt.Printf("read: %s", req.Line.Target)
		fmt.Printf("read: %s", req.Line.Version)

		conn.Close()
	}
}