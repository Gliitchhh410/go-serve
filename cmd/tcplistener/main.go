package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	// log.Println("Listening on :42069")
	pool := NewWorkerPool(runtime.NumCPU(), 1000) // optimal workers, 1000 queue

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// log.Printf("Accept loop stopping...")
				break
			}

			select {
			case pool.conns <- conn:
			default:
				// log.Printf("Server saturated, dropping connection from %s\n", conn.RemoteAddr())
				conn.Write(response503)
				conn.Close()
			}
		}
		close(pool.conns)
	}()
	<-quit
	// log.Println("Shutting down server gracefully.....")
	listener.Close()
	pool.wg.Wait() // Wait for all workers to finish their active connections
	// log.Println("Server gracefully shutted down")
}
