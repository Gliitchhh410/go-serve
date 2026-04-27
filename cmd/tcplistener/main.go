package main

import (
	"errors"
	"httpServer/internal/request"
	"httpServer/internal/response"
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

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}
		rw := response.NewResponseWriter(conn)

		log.Printf("Accepted connection from %s\n", conn.RemoteAddr().String())

		req, err := request.RequestFromReader(conn)
		if errors.Is(err, request.ErrMethodNotAllowed) {
			rw.SetStatus(405)
			rw.SetHeader("Allow", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
			rw.Send()
			request.ReleaseRequest(req)
			conn.Close()
			continue
		}
		if err != nil {
			rw.SetStatus(400)
			rw.Send()
			request.ReleaseRequest(req)
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
		rw.SetStatus(200)
		rw.SetBody([]byte("OK"))
		rw.Send()
		request.ReleaseRequest(req)
		conn.Close()

	}
}
