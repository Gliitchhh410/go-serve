// Package request implements a zero-allocation HTTP/1.1 stream parser.
//
// # Memory Model
//
// All parsed fields (Method, Path, Headers, Body) are slice windows into a
// pooled 64KB buffer. They are only valid until ReleaseRequest is called.
// See the Request type for full lifetime documentation.
//
// # Security Posture & Threat Model
//
// This parser enforces strict boundaries to defend against common HTTP attacks:
//
// Slowloris (Connection Starvation):
// The parser itself has no opinion on timing. The caller is responsible for
// wrapping the net.Conn in an idleTimeoutReader (see cmd/tcplistener) which
// calls SetReadDeadline before every Read. A client that stops sending data
// mid-request will have its connection dropped after the idle timeout elapses,
// freeing the goroutine and the pooled buffer.
//
// Heap Exhaustion (Header Flooding):
// The parser operates on a single fixed-capacity pooled buffer (64KB). If the
// cumulative size of the request line and headers exceeds this limit, the parser
// returns ErrRequestTooLarge and releases the buffer immediately. No unbounded
// allocation is possible regardless of how many headers the client sends.
//
// HTTP Request Smuggling (Header Injection):
// Each header line is validated by parseHeader, which rejects any name
// containing characters outside the RFC 7230 token alphabet. Obsolete line
// folding (a header value continuation starting with whitespace on the next
// line) is rejected as a malformed header rather than silently concatenated,
// preventing desync attacks between a proxy and this server.
//
// Buffer Overflows:
// The read buffer is sourced from a sync.Pool of fixed 64KB arrays. The parser
// tracks a parseIndex cursor that only advances forward and never writes past
// bufferedBytes. There is no dynamic resizing or shifting of the buffer, so
// no out-of-bounds write is possible regardless of input shape.

package request
