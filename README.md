# chill-http: Zero-Allocation TCP Protocol Parser & Connection Pool

A high-performance HTTP/1.1 server built from scratch in Go, focused strictly on
protocol token extraction, zero-allocation memory pooling, and bounded concurrency.
This is not a general-purpose framework. Routing, TLS, and middleware are
deliberately out of scope.

---

## What It Does

- Parses raw TCP streams into structured `*Request` objects with zero heap allocations in the hot path
- Bounds concurrency to a fixed worker pool, preventing goroutine exhaustion under load
- Actively sheds excess traffic with instant 503 responses when the pool is saturated
- Drops stalled connections via per-read idle timeouts, defending against Slowloris

## What It Deliberately Does Not Do

- No router or URL pattern matching
- No TLS termination
- No middleware chain
- No HTTP/2

---

## Architecture

### Zero-Allocation Stream Parsing

The parser operates on a single 64KB buffer sourced from a `sync.Pool`. Rather than
shifting bytes or cloning slices, it advances an append-only `parseIndex` cursor
through the buffer as tokens are extracted. All parsed fields (`Method`, `Path`,
`Headers`, `Body`) are lightweight slice headers pointing directly into the pooled
buffer — no heap allocation occurs during parsing under steady load. chill-http focuses strictly on the parsing hot path. The zero-allocation constraint applies to request parsing only. The response path and connection lifecycle are unoptimized. For a full keep-alive comparison, see the benchmarks directory.

```
TCP stream → idleTimeoutReader → RequestFromReader → *Request
                                         ↑
                              64KB buffer from sync.Pool
                              (cursor-based, no copy, no clone)
```

### Memory Model

Two `sync.Pool` instances manage all allocations:

| Pool          | Type             | Lifecycle                                                                              |
| ------------- | ---------------- | -------------------------------------------------------------------------------------- |
| `requestPool` | `*Request`       | Acquired at connection start, released after response                                  |
| `bufferPool`  | `*[]byte` (64KB) | Acquired inside `RequestFromReader`, owned by `*Request`, released by `ReleaseRequest` |

**Lifetime constraint:** All fields on `*Request` are slice windows into the pooled
buffer. They are only valid until `ReleaseRequest` is called. Any field that must
outlive the handler (e.g. passed to a goroutine) must be explicitly copied:

```go
body := bytes.Clone(req.Body) // caller owns this copy
go process(body)
```

Retaining a slice from `*Request` after `ReleaseRequest` is a data race.

### Concurrency Model

```
listener.Accept()
       │
       ▼
   select {
   case pool.conns <- conn:   ← queued (up to 1000 slots)
   default:                   ← instant 503, conn closed
   }
       │
       ▼
  WorkerPool (runtime.NumCPU() goroutines)
  each ranging over pool.conns
       │
       ▼
  handleConn → idleTimeoutReader → RequestFromReader → response
```

- **`runtime.NumCPU()` workers**, **1000-slot queue** = maximum `NumCPU + 1000` concurrent connections
- Worker count scales automatically to the host machine's logical core count
- Connections beyond capacity receive `HTTP/1.1 503 Service Unavailable` immediately
- The main goroutine never blocks, keeping the OS TCP backlog clear

### Graceful Shutdown

On `SIGTERM` or `Ctrl+C`:

1. `listener.Close()` stops accepting new connections
2. The accept goroutine exits its loop and calls `close(pool.conns)`
3. Workers drain the queue, finishing active connections
4. `pool.wg.Wait()` blocks until all workers exit
5. Process terminates cleanly

No active connection is dropped mid-request during a controlled shutdown.

---

## Security Posture

| Threat                                  | Mitigation                                                                                                                                        |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Slowloris**                           | `idleTimeoutReader` calls `SetReadDeadline` before every `Read`. A client that stops sending for 5 seconds is dropped with 408.                   |
| **Heap exhaustion via header flooding** | Parser operates on a fixed 64KB buffer. Requests exceeding this limit return `ErrRequestTooLarge` immediately, no unbounded allocation possible.  |
| **Buffer overflow**                     | Append-only cursor model. `parseIndex` only advances forward, never writes past `bufferedBytes`. No dynamic resizing.                             |
| **Request smuggling**                   | Header names are validated against the RFC 7230 token alphabet. Obsolete line folding is rejected as malformed rather than silently concatenated. |
| **Goroutine exhaustion**                | Fixed worker pool bounds maximum concurrent goroutines regardless of connection rate.                                                             |

---

## Benchmarks

Measured on Intel Core i7-10750H, Windows, Go 1.22:

```
BenchmarkRequestFromReader-12    0 B/op    0 allocs/op
```

The hot path (single-read request fitting in one `r.Read` call) allocates nothing.
Fragmented requests trigger one `bytes.Clone` per field only when the parse buffer
must be refilled mid-header — unavoidable given the streaming constraint.

---

## Running

```bash
go build -o server.exe ./cmd/tcplistener
./server.exe
```

```bash
# Load test (requires server running)
go run ./cmd/loadtest/main.go
```
