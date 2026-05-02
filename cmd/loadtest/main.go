// cmd/loadtest/main.go
package main

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var success, overloaded, failed int32
	var wg sync.WaitGroup

	req := []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	concurrency := 500

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", "localhost:42069", 5*time.Second)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				return
			}
			defer conn.Close()

			_, err = conn.Write(req)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				return
			}

			buf := make([]byte, 512)
			n, err := conn.Read(buf)
			if err != nil && n == 0 {
				atomic.AddInt32(&failed, 1)
				return
			}

			resp := buf[:n]
			if bytes.Contains(resp, []byte("200 OK")) {
				atomic.AddInt32(&success, 1)
			} else if bytes.Contains(resp, []byte("503")) {
				atomic.AddInt32(&overloaded, 1)
			} else {
				atomic.AddInt32(&failed, 1)
			}
		}()
	}

	wg.Wait()

	fmt.Printf("success:    %d\n", success)
	fmt.Printf("overloaded: %d\n", overloaded)
	fmt.Printf("failed:     %d\n", failed)
	fmt.Printf("total:      %d\n", success+overloaded+failed)
}
