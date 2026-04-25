# Copilot Instructions for `go-serve`

## Build, test, and lint commands

Use the repository root (`D:\DS\httpServer`) as the working directory.

```bash
# Run the TCP listener server
go run ./cmd/tcplistener

# Build all packages
go build ./...

# Run full test suite
go test ./...

# Run request parser tests only
go test ./internal/request

# Run headers package tests only
go test ./internal/headers

# Run a single test by name
go test ./internal/request -run '^TestRequestFromReader_Body$'
go test ./internal/headers -run '^TestHeaders_ParseBlock$'
go test ./internal/request -run '^TestRequest_StateTransitions$'
```

No dedicated lint task/config (Makefile, Taskfile, golangci config) is currently present. Prefer standard Go formatting and vetting commands when needed:

```bash
gofmt -w .
go vet ./...
```

## High-level architecture

This codebase is a small TCP-based HTTP/1.1 request parser server split into three core layers:

1. `cmd/tcplistener/main.go`  
   Owns the TCP listener (`:42069`), accepts connections, parses one request per connection, logs parsed fields, writes a fixed `200 OK` response, then closes the connection.
2. `internal/request/request.go`  
   Implements streaming request parsing from `io.Reader` (`RequestFromReader`). It maintains an internal state machine:
   - `stateInit`: parse request line
   - `stateHeaders`: parse header block
   - `stateBody`: read body according to `Content-Length`
   - `stateDone` / `stateError`: terminal states  
   Parsing is incremental and returns consumed-byte counts so unconsumed bytes remain buffered. Request objects are reused via `sync.Pool` (`AcquireRequest` / `ReleaseRequest`) to reduce allocations.
3. `internal/headers/{headers.go,parse.go}`  
   Encapsulates header storage and parsing. Header keys are normalized to lowercase, duplicate keys are merged, and block parsing is CRLF-driven until an empty line (`\r\n`) marks header completion.

## Key conventions in this repository

- **Strict protocol line endings:** parsing expects `\r\n` (CRLF), not bare `\n`.
- **HTTP version is enforced:** request line must be exactly `HTTP/1.1`.
- **Header key normalization:** always lowercase keys for get/set consistency.
- **Duplicate header behavior:** duplicate names are merged as `existing + ", " + value`.
- **Streaming-first parser contract:** parse methods return exact consumed bytes; caller shifts remaining bytes and continues reading.
- **Request lifecycle contract:** requests are pooled; any request acquired from `RequestFromReader`/`AcquireRequest` should be released with `ReleaseRequest` when done.
- **Body contract:** body parsing is controlled by `Content-Length`; missing header means zero-length body; invalid/negative values are treated as malformed request.
- **Error terminal behavior:** once parser enters `stateError`, it is treated as done (no further parsing progress expected).
- **Header syntax strictness:** header field-name rejects trailing space before `:` and validates token characters; malformed lines fail parsing.
- **Tests model fragmented I/O:** `internal/request/request_test.go` uses a custom chunked reader to simulate partial TCP reads; preserve this style when adding parser tests.
