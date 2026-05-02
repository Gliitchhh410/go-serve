package main

import (
	"errors"
	"httpServer/internal/request"
	"httpServer/internal/response"
	"log"
	"net"
	"time"
)

const readTimeout = 5 * time.Second

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
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	rw := response.NewResponseWriter(conn)

	log.Printf("Accepted connection from %s\n", conn.RemoteAddr())

	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("Failed to set read deadline: %v\n", err)
		return
	}

	req, err := request.RequestFromReader(conn)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			log.Printf("Connection timed out (Slowloris?): %s\n", conn.RemoteAddr())
			rw.SetStatus(408)
			rw.Send()
			return
		}
		if errors.Is(err, request.ErrUnexpectedEOF) {
			log.Printf("Client disconnected: %s\n", conn.RemoteAddr())
			return
		}
		if errors.Is(err, request.ErrMethodNotAllowed) {
			rw.SetStatus(405)
			rw.SetHeader("Allow", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
			rw.Send()
			return
		}
		if errors.Is(err, request.ErrUnsupportedTransferEncoding) {
			rw.SetStatus(501)
			rw.Send()
			return
		}
		rw.SetStatus(400)
		rw.Send()
		return
	}

	defer request.ReleaseRequest(req)

	log.Printf("Method: %s\n", req.Line.Method)
	log.Printf("Target: %s\n", req.Line.Target.Path)
	log.Printf("Version: %s\n", req.Line.Version)
	req.Headers.ForEach(func(name, value []byte) {
		log.Printf("Header: %s: %s\n", name, value)
	})
	log.Printf("Body: %s\n", req.Body)

	rw.SetStatus(200)
	rw.SetBody([]byte("OK"))
	rw.Send()
}
