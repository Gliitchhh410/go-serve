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
		

		if req != nil && req.Line != nil {
			fmt.Printf("Method: %s\n", req.Line.Method)
			fmt.Printf("Target: %s\n", req.Line.Target)
			fmt.Printf("Version: %s\n", req.Line.Version)
			req.Headers.ForEach(func(name, value string) {
				fmt.Printf("Header: %s: %s\n", name, value)
			})
			fmt.Printf("Body: %s\n", string(req.Body))
		}
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
		conn.Close()



	}
}