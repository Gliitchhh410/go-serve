package main

import (
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
			log.Printf("Method: %s\n", req.Line.Method)
			log.Printf("Target: %s\n", req.Line.Target)
			log.Printf("Version: %s\n", req.Line.Version)
			req.Headers.ForEach(func(name, value string) {
				log.Printf("Header: %s: %s\n", name, value)
			})
			log.Printf("Body: %s\n", string(req.Body))
		}
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
		conn.Close()



	}
}