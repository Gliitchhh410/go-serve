# ADR 1: Using sync.Pool for TCP Connection Read Buffers

## Status

Accepted

## Context

Every incoming TCP connection requires a byte buffer to read the raw stream into
before the parser can extract tokens. The naive approach is:

```go
buf := make([]byte, 64*1024) // allocates on every request
```

Under load, this means thousands of 64KB heap allocations per second. Each
allocation pressures the garbage collector, increases GC pause frequency, and
reduces throughput. The standard library's `io.ReadAll` compounds this by
growing the buffer dynamically, meaning each request triggers multiple
allocations of increasing size.

This project targets zero heap allocations in the hot path — the request parsing
loop must not allocate memory once the server is under steady load.

## Decision

We implemented a `sync.Pool` of fixed 64KB byte arrays shared across all
goroutines:

```go
var bufferPool = sync.Pool{
    New: func() any {
        buf := make([]byte, 64*1024)
        return &buf
    },
}
```

A pointer to the slice (`*[]byte`) is stored rather than the slice itself to
prevent the slice header from escaping to the heap on every `Get` call.

**Why 64KB:**

- Covers the vast majority of real-world HTTP requests (headers + small bodies)
  in a single read without needing to grow the buffer
- Matches the typical OS TCP receive buffer size, meaning one `conn.Read` call
  usually drains a full network segment
- Acts as a hard security limit — requests exceeding 64KB are rejected with
  `ErrRequestTooLarge`, bounding memory usage per connection

The buffer is acquired at the start of `RequestFromReader`, attached to the
`*Request` struct via `req.rawBuffer`, and returned to the pool inside
`ReleaseRequest`. This ties the buffer lifetime strictly to the request lifetime.

An append-only cursor (`parseIndex`) advances through the buffer as tokens are
extracted. Bytes are never shifted or copied within the buffer, which means all
parsed slice fields (`Method`, `Path`, `Headers`, `Body`) remain stable pointers
into the same backing array for the entire lifetime of the request.

## Consequences

**Positive:**

- Zero heap allocations per request in the steady state
- Reduced GC pressure under high concurrency
- Predictable memory usage: `workerCount * 64KB` maximum buffer footprint
- Natural DoS protection: requests exceeding 64KB are hard-rejected

**Negative / Constraints:**

- The `*Request` object and all its fields are borrowed memory, not owned memory.
  All slice fields (`Method`, `Path`, `Headers`, `Body`) are only valid until
  `ReleaseRequest` is called. Retaining any of these after release is a silent
  data race — the buffer will be reused for another connection.
- Any handler that needs to retain data asynchronously (e.g. pass a header value
  to a goroutine) must explicitly copy it via `bytes.Clone`. This is an
  uncommon but real constraint that must be enforced by convention and documented
  at every API boundary.
- Pooled buffers are never zeroed between reuses. If a bug causes a field to
  point past its actual token boundary, it may silently read stale data from a
  previous request. Strict index tracking in the parser is the only defense.
- The fixed 64KB ceiling makes this parser unsuitable for APIs that accept large
  request bodies (file uploads, multipart forms). Those use cases require a
  streaming body reader that bypasses the pooled buffer entirely.
