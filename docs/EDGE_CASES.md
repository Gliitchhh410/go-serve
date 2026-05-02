# Protocol Edge Cases & Handling

## TCP Fragmentation (Partial Reads)

TCP is a streaming protocol, not a packet protocol. When a client sends:

```
GET /foo HTTP/1.1\r\nHost: localhost\r\n\r\n
```

The OS may deliver this to your `conn.Read` call in arbitrary fragments:

```
Read 1: "GET /fo"
Read 2: "o HTTP/1"
Read 3: ".1\r\nHost"
Read 4: ": localhost\r\n\r\n"
```

This is not an error — it is normal network behavior, especially under load or
across real network links.

### How chill-http Handles It

`RequestFromReader` maintains two integers over the read buffer:

```
readBuf:       [ G E T   / f o o   H T T P / 1 . 1 \r \n H o s t ... ]
                ↑                                    ↑
            bufferedBytes=0                     bufferedBytes grows
            parseIndex=0                        parseIndex advances
```

On each iteration of the read loop:

1. `r.Read(readBuf[bufferedBytes:])` appends new bytes into the free space
   at the end of the buffer, advancing `bufferedBytes`.
2. `req.parse(readBuf[parseIndex:bufferedBytes])` is called on the window
   of unprocessed bytes.
3. If the parser cannot find a complete token (e.g. no `\r\n` yet), it
   returns `consumed = 0` and the loop reads more data.
4. When `consumed > 0`, `parseIndex += consumed` advances the cursor past
   the processed bytes.

**Critical property:** bytes are never moved or overwritten once written.
`parseIndex` only moves forward. This means slice fields like `req.Line.Method`
remain valid pointers into the buffer across multiple read iterations — no
copying or cloning is required for correctly-sized requests.

**Buffer exhaustion:** If `bufferedBytes == len(readBuf)` (64KB) and the
parser is still not done, the request is rejected with `ErrRequestTooLarge`.
This prevents unbounded memory growth from adversarial inputs.

---

## Unexpected EOF (Client Drops Mid-Request)

If a client connects, sends a partial request, and then closes the connection,
`conn.Read` returns `(0, io.EOF)` or `(n, io.EOF)`.

### What Happens

`RequestFromReader` checks for `io.EOF` immediately after every `Read` call:

```go
bytesRead, err := r.Read(readBuf[bufferedBytes:])
if err == io.EOF {
    ReleaseBuffer(bufPtr)
    req.rawBuffer = nil
    ReleaseRequest(req)
    return nil, ErrUnexpectedEOF
}
```

All pooled resources are released before returning. The connection handler in
`handleConn` receives `ErrUnexpectedEOF`, logs the disconnect, and returns
without writing a response — the connection is already closed by the client.

### Why Not Attempt Recovery

There is no meaningful recovery from a mid-request EOF. The partial data cannot
be reassembled without the missing bytes. Attempting to parse or respond to a
partial request risks sending garbage to the wrong client or reading stale data
from the pooled buffer.

---

## Idle Connection Timeout (Slowloris)

A client may connect and send bytes so slowly that it never completes a request
within a reasonable time — intentionally (Slowloris attack) or unintentionally
(very slow mobile connection).

### What Happens

`idleTimeoutReader` wraps the `net.Conn` and calls `SetReadDeadline` before
every `Read`:

```go
func (itr *idleTimeoutReader) Read(p []byte) (int, error) {
    itr.conn.SetReadDeadline(time.Now().Add(itr.timeout))
    return itr.conn.Read(p)
}
```

If the client sends no data for 5 seconds, `conn.Read` returns a timeout error.
`RequestFromReader` propagates this error, resources are released, and
`handleConn` responds with `HTTP/1.1 408 Request Timeout` before closing.

**Key distinction from an absolute deadline:** The deadline resets on every
successful read. A legitimate client uploading a large body slowly but steadily
will never hit the timeout as long as it keeps sending data. Only a client that
goes completely silent triggers the drop.
