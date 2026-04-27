package response


import (
	"httpServer/internal/headers"
	"net"
	"fmt"
	"strconv"
)

type ResponseWriter struct {
	conn net.Conn
	StatusCode int
	Reason string
	Headers *headers.Headers
	Body []byte
}

func NewResponseWriter(conn net.Conn) *ResponseWriter{
	return &ResponseWriter{
		conn: conn,
		StatusCode: 200,
		Reason: "OK",
		Headers: headers.New(),
		Body: nil,
	}
} 

func (rw *ResponseWriter) SetStatus(code int){
	rw.StatusCode = code 
	rw.Reason = StatusText(code)
}

func (rw *ResponseWriter) SetHeader(name, value string){
	rw.Headers.Set(name, value)
}

func (rw *ResponseWriter) SetBody(body []byte){
	rw.Body = body
	rw.SetHeader("Content-Length", strconv.Itoa(len(body)))
}

func (rw *ResponseWriter) Send() error {
	fmt.Fprintf(rw.conn, "HTTP/1.1 %d %s\r\n", rw.StatusCode, rw.Reason)
	rw.Headers.ForEach(func(name, value string){
		fmt.Fprintf(rw.conn, "%s: %s\r\n", name, value)
	})
	fmt.Fprintf(rw.conn, "\r\n")

	if len(rw.Body) == 0 {
		return nil
	} 
	_, err := rw.conn.Write(rw.Body)
	return err
}