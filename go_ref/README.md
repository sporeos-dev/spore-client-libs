# Spore Go Client

A Go library for building nodes that connect to the [Spore OS](https://github.com/sporeos-dev/spore-os) daemon via IPC.

**Module path:** `github.com/sporeos-dev/spore-client-libs/go`  
**Protocol version:** SPORE/v1d0  
**Transport:** Unix domain socket (path is platform-specific, see `DefaultSocketPath`)  
**Go version:** 1.25+

---

## Overview

Spore OS is a local IPC daemon that lets processes discover and call each other without knowing anything about each other's implementation. It uses a hub-and-spoke architecture: a central hub daemon routes messages between nodes based on a manifest registry.

This library handles everything needed to be a **node** (spoke):

1. Connecting to the hub and completing the handshake
2. Registering handlers for the subjects your node exposes
3. Listening for incoming calls and dispatching them to your handlers
4. Sending correctly-formatted responses (or errors) back through the hub

---

## Installation

### Remote (once the repository is public)

```sh
go get github.com/sporeos-dev/spore-client-libs/go
```

### Local path (during development)

In the `go.mod` of the project consuming this library:

```
require github.com/sporeos-dev/spore-client-libs/go v0.0.0

replace github.com/sporeos-dev/spore-client-libs/go => /path/to/spore-client-libs/go
```

---

## Concepts

Before using the library, three Spore concepts are important:

### Node ID

Every node has a **reverse-domain ID** declared in its manifest YAML, e.g. `com.example.clock`. This ID is sent to the hub during the handshake. The hub looks it up in its manifest registry — **the node must be installed with the hub before `Connect()` will succeed**.

Installation is done out-of-band via the Spore CLI:
```sh
SPORE.node.install path=/path/to/node.manifest.spore.yaml
```

### Subjects

A subject is the name of an API endpoint. Subjects use dot-notation, e.g. `clock.get_time`. They can be:

- **Short form:** `clock.get_time` — derived from the last segment of the node ID + the API name
- **Full form:** `com.example.clock.get_time` — fully qualified, always unambiguous

When you register a handler with this library using just the short name (`"get_time"`), it will match both short-form and full-form incoming calls.

### Handles

A handle (`~h1`, `~w3`, etc.) is a caller-supplied token used to correlate a response to its originating call. The hub echoes it back as `~handle:subject` on the response. Your handler receives it in `call.Handle` — `Reply()` and `Error()` use it automatically to format the correct response prefix.

---

## Minimal Working Example

```go
package main

import (
	"fmt"
	"log"

	spore "github.com/sporeos-dev/spore-client-libs/go"
)

func main() {
	// Create a client for this node. The ID must match the manifest.
	client := spore.NewClient("com.example.clock")

	// Register a handler for the "get_time" API subject.
	// This matches incoming calls like "clock.get_time" or "com.example.clock.get_time".
	client.HandleRequest("get_time", func(call *spore.Call) {
		timezone := call.ArgOr("timezone", "UTC")
		fmt.Println("get_time called, timezone:", timezone)

		err := call.Reply(map[string]string{
			"time": "2026-03-13T14:32:00Z",
		})
		if err != nil {
			log.Println("reply error:", err)
		}
	})

	// Connect to the daemon and complete the handshake.
	if err := client.Connect(); err != nil {
		log.Fatal("connect:", err)
	}
	defer client.Close()

	fmt.Println("Connected. Listening for calls...")

	// Block and handle incoming calls until the connection drops.
	if err := client.Listen(); err != nil {
		log.Println("listen ended:", err)
	}
}
```

---

## Common Patterns

### Handling an incoming call

```go
client.HandleRequest("get_time", func(call *spore.Call) {
    tz := call.ArgOr("timezone", "UTC")
    call.Reply(map[string]string{"time": "2026-03-13T14:32:00Z", "timezone": tz})
})
```

### Returning errors

```go
client.HandleRequest("get_time", func(call *spore.Call) {
    if !call.HasArg("timezone") {
        call.Error(spore.ErrorCodeArgumentMissing, "timezone is required")
        return
    }
    // node-defined error code declared in the manifest
    call.ErrorCustom("clock.err.invalid_timezone", "Unknown timezone: Fakezone")
})
```

### Cancel (no result, not an error)

```go
client.HandleRequest("open_dialog", func(call *spore.Call) {
    // user dismissed the dialog — clean non-result
    call.Cancel()
})
```

### Calling another node and waiting for the response

```go
resp, err := client.SendAndWait("clock.get_time timezone=UTC ~req1", 5000)
if err != nil {
    log.Fatal(err)
}
if !resp.OK {
    log.Fatalf("error: %s — %s", resp.ErrCode, resp.ErrWhat)
}
fmt.Println("time:", resp.Arg("time"))
```

### Calling another node from inside a handler (goroutine required)

```go
client.HandleRequest("what_time", func(call *spore.Call) {
    go func() {
        resp, err := client.SendAndWait("clock.get_time timezone=UTC ~inner1", 5000)
        if err != nil || !resp.OK {
            call.Error(spore.ErrorCodeUpstream, "clock unavailable")
            return
        }
        call.Reply(map[string]string{"time": resp.Arg("time")})
    }()
})
```

### Fire-and-forget with an async response handler

```go
client.HandleResponse("get_time", func(resp *spore.Response) {
    if resp.OK {
        fmt.Println("time:", resp.Arg("time"))
    }
})

client.Send("clock.get_time timezone=UTC ~fire1")
// response is delivered to HandleResponse when Listen() reads it
```

### Subscribe to a topic

```go
if err := client.Subscribe("sensors.temperature", func(msg *spore.PublishMessage) {
    fmt.Println("temp:", msg.Arg("value"), msg.ArgOr("unit", "?"))
}, 5000); err != nil {
    log.Fatal(err)
}
// later:
client.Unsubscribe("sensors.temperature", 5000)
```

### Observe all traffic (witness node)

```go
client.HandleWitness(func(msg *spore.WitnessMessage) {
    if msg.Kind == spore.WitnessKindOutgoing {
        fmt.Printf("[%d] → %s\n", msg.SporeTime, msg.Subject)
    }
})
```

---

## API Reference

### Creating a Client

```go
// Connect to the default socket (platform-specific path)
client := spore.NewClient("com.example.mynode")

// Connect to a custom socket path (useful for testing)
client := spore.NewClientWithSocket("com.example.mynode", "/custom/path.sock")
```

`DefaultSocketPath` is an exported variable with the correct socket path for the current platform:
- **Linux:** `/var/lib/spore-os/spore.sock`
- **macOS:** `/Library/Application Support/spore-os/spore.sock`
- **Windows:** `%LOCALAPPDATA%\spore-os\spore.sock`

---

### Registering Request Handlers

```go
client.HandleRequest(subject string, handler spore.HandlerFunc)
```

`subject` can be:

| Registered as | Matches incoming subject |
|---|---|
| `"get_time"` | `get_time`, `clock.get_time`, `com.example.clock.get_time` |
| `"clock.get_time"` | `clock.get_time` exactly |

Matching priority: exact match first, then short-name suffix match (last segment after the final dot).

`HandlerFunc` signature:
```go
type HandlerFunc func(call *spore.Call)
```

Handlers are safe to call concurrently. The library holds a read lock while dispatching, so you can call `HandleRequest()` from any goroutine.

---

### Connecting

```go
err := client.Connect()
```

- Dials the Unix socket
- Sends the node ID to the hub (`nodeID + "\n"`)
- Expects `OK\n` back from the hub
- Returns an error if the node is not installed or the daemon is not running
- Handshake has a 5-second timeout

---

### Listening for Calls

```go
err := client.Listen()
```

- Blocks until the connection is closed or a read error occurs
- Reads newline-delimited messages from the hub
- Dispatches inbound calls to handlers registered with `HandleRequest()`
- Routes responses (`~handle:subject ...`) to active `Wait`/`WaitFor` waiters, then to a matching `HandleResponse(subject)` handler, then discards
- Routes witness copies (`witness ...`) to the `HandleWitness` handler if registered
- Returns a non-nil error when the connection ends — check the error to distinguish a clean close from a network error

Typical usage:
```go
if err := client.Listen(); err != nil {
	if strings.Contains(err.Error(), "use of closed network connection") {
		// clean shutdown from client.Close()
	} else {
		log.Println("unexpected disconnect:", err)
	}
}
```

---

### Registering a Response Handler

```go
client.HandleResponse("clock.get_time", func(resp *spore.Response) {
	// called for responses to outgoing clock.get_time calls,
	// when no Wait/WaitFor waiter is active
	if resp.OK {
		fmt.Println("time:", resp.Arg("time"))
	} else {
		fmt.Println("error:", resp.ErrCode, resp.ErrWhat)
	}
})
```

`HandleResponse` registers a subject-keyed callback for responses received during `Listen()`. The subject is matched against the echoed subject in the response (`~handle:subject`) using the same two-step logic as `HandleRequest`: exact match first, then short-name suffix. For example, `HandleResponse("get_time", ...)` matches responses for `clock.get_time` or `com.example.clock.get_time`.

Delivery priority when a response arrives in `Listen()`:

| Priority | Who gets it |
|---|---|
| 1 | A `SendAndWait` or `WaitFor(handle, timeoutMs)` waiter registered for that exact handle |
| 2 | A `Wait(timeoutMs)` wildcard waiter (next response of any handle) |
| 3 | The `HandleResponse(subject)` handler whose subject matches |

Only one destination receives each response. Calling `HandleResponse` with the same subject again replaces that subject's handler.

---

### Registering a Witness Handler

```go
client.HandleWitness(func(msg *spore.WitnessMessage) {
    // called for every witness copy the hub sends to this node
    fmt.Printf("[witness %s] %s\n", msg.Kind, msg.Subject)
})
```

Witness nodes receive a copy of every message that passes through the hub — before and after routing, inline expansions, hub events, and node-emitted diagnostics. A node opts in by declaring `witness: true` in its manifest YAML. The client library routes all `witness `-prefixed lines to this handler rather than to any request or response handler.

`HandleWitness` replaces any previously registered handler. Pass `nil` to clear it.

`WitnessHandlerFunc` signature:
```go
type WitnessHandlerFunc func(msg *spore.WitnessMessage)
```

#### The `WitnessMessage` Object

```go
msg.Kind       // WitnessKindIncoming | Outgoing | Expanded | Event | Node
msg.SporeTime  // hub-stamped Unix timestamp in milliseconds
msg.IsResponse // true when the body is a ~handle:subject response; false for subject-first calls
msg.Subject    // subject of the inner message
msg.Handle     // correlation token, if present
msg.Cast       // originating caller node ID (call-type bodies, from cast=)

// Response-type body fields (IsResponse == true):
msg.OK           // true when inner response carries "ok"
msg.Cancelled    // true when inner response carries "cancelled"
msg.CustomError  // true when inner response carries "custom_error"
msg.ErrCode      // error code on error responses
msg.ErrWhat      // human-readable error description on error responses
msg.Capture      // responding node ID (from capture=)
msg.ErrorOrigin  // ErrorOriginSpore | Node | Cast | Capture

msg.Args         // data key=value pairs (protocol + spore_ fields excluded)
msg.Flags        // bare flag tokens (protocol + spore_ flags excluded)
```

#### Witness Kinds

| Constant | Wire flag | What it represents |
|---|---|---|
| `WitnessKindIncoming` | `spore_incoming` | Raw message received from a node, before hub injection |
| `WitnessKindOutgoing` | `spore_outgoing` | Message delivered to a node, after hub injection (`cast=`, `ok`, etc.) |
| `WitnessKindExpanded` | `spore_expanded` | Inline call expansion (`~~`-handle inner call) generated by the hub |
| `WitnessKindEvent` | `spore_event` | Hub lifecycle or internal error event (e.g. handshake failures) |
| `WitnessKindNode` | `spore_node` | Node-emitted observability message via `SPORE.witness.emit` |

#### Filtering Example

You can inspect `Kind` inside the handler to filter what you care about:

```go
client.HandleWitness(func(msg *spore.WitnessMessage) {
    // Only log messages going out to nodes, ignore hub-internal copies.
    if msg.Kind != spore.WitnessKindOutgoing {
        return
    }
    t := time.UnixMilli(msg.SporeTime)
    fmt.Printf("%s → %s\n", t.Format(time.RFC3339), msg.Subject)
})
```

#### Node-Emitted Witness Messages

Any connected node can push its own observability messages to all witnesses by calling the hub subject `SPORE.witness.emit`:

```go
client.Send("SPORE.witness.emit message=\"loaded config\" ~w1")
```

The hub forwards the content to all witness nodes tagged with `spore_node`, and returns a standard `ok` response to the emitting node.

---

### Sending Calls to Other Nodes

```go
err := client.Send("clock.get_time timezone=UTC ~h1")
```

`Send` writes a raw Spore message to the hub, terminated with `\n`. Use this to call subjects on other nodes. The `~handle` token at the end lets you correlate the response — use `WaitFor(handle, timeoutMs)` or `Wait(timeoutMs)` immediately after `Send` to receive the reply, or use `SendAndWait` to do both in one call.

---

### SendAndWait

`SendAndWait` is a convenience that atomically sends a message and blocks until the response for its handle arrives, or the timeout elapses. It is race-free in listener mode because the waiter is registered before the message is written to the socket.

The message **must** contain a handle token. If none is present, `SendAndWait` returns an error immediately.

```go
resp, err := client.SendAndWait("clock.get_time timezone=UTC ~req1", 5000)
if err != nil {
    log.Fatal(err) // includes "send timed out waiting for response" on timeout
}
if !resp.OK {
    if resp.FromSpore() {
        log.Fatalf("hub error: %s — %s", resp.ErrCode, resp.ErrWhat)
    }
    log.Fatalf("node error: %s — %s", resp.ErrCode, resp.ErrWhat)
}
fmt.Println("time:", resp.Arg("time"))
```

Under the hood, `SendAndWait` uses the same handle-keyed waiter slot as `WaitFor`, so delivery priority is identical — it takes precedence over any `HandleResponse` handler registered for the same subject. The same deadlock warning applies: do not call `SendAndWait` from inside a `HandleRequest` handler while `Listen()` is running.

---

### Waiting for Responses

`Wait(timeoutMs)` and `WaitFor(handle, timeoutMs)` both block until a response arrives or the timeout elapses. Both return an error on timeout. They work in two modes depending on whether `Listen()` is running:

| Mode | How it works |
|---|---|
| **Standalone** (`Listen()` not running) | Reads directly from the socket. Use for procedural scripts and CLIs. |
| **Listener** (`Listen()` running in a goroutine) | Registers a channel; `Listen()` delivers the response to it. Safe to call from any goroutine. |

In both modes, `Wait`/`WaitFor` take precedence over `HandleResponse` — the handler is not called for a response consumed by a waiter.

> **Deadlock warning:** Do not call `Wait` or `WaitFor` from inside a `HandleRequest()` call handler while `Listen()` is running. `Listen()` dispatches handlers synchronously, so it cannot deliver the response while it is blocked inside your handler. If you need to make an outgoing call and wait for its reply from within a handler, launch a goroutine:
> ```go
> client.HandleRequest("my_command", func(call *spore.Call) {
> 	go func() {
> 		client.Send("other.thing ~out1")
> 		resp, _ := client.WaitFor("out1", 5000)
> 		call.Reply(map[string]string{"result": resp.Arg("result")})
> 	}()
> })
> ```

#### `Wait(timeoutMs)`

Blocks until the **next response** of any handle arrives, or until `timeoutMs` milliseconds have elapsed.

```go
client.Send("clock.get_time timezone=UTC ~req1")
resp, err := client.Wait(5000)
if err != nil {
	log.Fatal(err) // includes "wait timed out" on timeout
}
if resp.OK {
	fmt.Println("time:", resp.Arg("time"))
} else {
	fmt.Println("error:", resp.ErrCode, resp.ErrWhat)
}
```

#### `WaitFor(handle, timeoutMs)`

Blocks until the response with a **specific handle** arrives, discarding any others, or until `timeoutMs` milliseconds have elapsed. Use this when you have multiple in-flight calls and want to match a reply to a particular one.

```go
client.Send("clock.get_time timezone=UTC ~req1")
resp, err := client.WaitFor("req1", 5000)
if err != nil {
	log.Fatal(err) // includes "wait timed out" on timeout
}
fmt.Println("time:", resp.Arg("time"))
```

#### The `Response` Object

`Wait(timeoutMs)`, `WaitFor(handle, timeoutMs)`, and `SendAndWait` return a `*Response`:

```go
resp.OK        // true on success, false on error
resp.ErrCode   // namespaced error code (e.g. "clock.err.invalid_timezone")
resp.ErrWhat   // human-readable error description
resp.Handle    // the handle echoed back by the hub (e.g. "req1")
resp.Subject   // the subject echoed back (e.g. "clock.get_time")
resp.Capture   // node ID of whoever handled the call (always present)
               // "SPORE.hub" when the hub itself produced the response

resp.FromSpore() // convenience: true when Capture == "SPORE.hub"

// Read returned key=value arguments.
// Reserved fields (capture, code, what) are promoted to their own fields
// and are never present in Args.
resp.Arg("time")              // returns "" if missing
resp.ArgOr("time", "unknown") // returns the default if missing
```

Typical error handling pattern:

```go
if !resp.OK {
    if resp.FromSpore() {
        // Hub-produced: node unavailable, malformed call, hub timeout, etc.
        log.Fatalf("hub error: %s — %s", resp.ErrCode, resp.ErrWhat)
    }
    // Node-produced domain error
    log.Fatalf("node error: %s — %s", resp.ErrCode, resp.ErrWhat)
}
```

---

### Closing

```go
err := client.Close()
```

Closes the connection. The hub marks the node as installed-but-not-connected. Safe to call multiple times (no-op if already closed).

---

## The `Call` Object

Every handler receives a `*spore.Call`. It provides:

### Reading Incoming Arguments

```go
// Get a required argument. Returns "" if not present.
timezone := call.Arg("timezone")

// Get an argument with a fallback default.
timezone := call.ArgOr("timezone", "UTC")

// Check if an argument is present before reading.
if call.HasArg("timezone") {
	// ...
}

// Check for a bare flag token (e.g., "recursive" in "filesystem.read path=/tmp recursive")
if call.HasFlag("recursive") {
	// ...
}
```

### Metadata Fields

```go
call.Subject  // Full subject string, e.g. "clock.get_time"
call.Handle   // Correlation handle, e.g. "h1" (from ~h1). Empty if no handle was sent.
call.Cast     // Node ID of the caller, injected by the hub. May be empty.
```

### Sending a Response

**Success response:**
```go
// With data
call.Reply(map[string]string{
	"time": "2026-03-13T14:32:00Z",
})

// Void (no return data — sends just the prefix, hub adds "ok capture=...")
call.Reply(nil)
call.Reply(map[string]string{})
```

Produces on the wire (with handle `h1`):
```
~h1:clock.get_time time=2026-03-13T14:32:00Z
```
Without a handle:
```
clock.get_time time=2026-03-13T14:32:00Z
```

Values containing spaces are automatically quoted:
```go
call.Reply(map[string]string{"message": "hello world"})
// → ~h1:subject.name message="hello world"
```

**Error response:**
```go
call.Error("clock.err.invalid_timezone", "Unknown timezone: Fakezone")
```

Produces on the wire:
```
~h1:clock.get_time error code=clock.err.invalid_timezone what="Unknown timezone: Fakezone"
```

Error code convention: `nodename.err.error_name` (e.g., `clock.err.invalid_timezone`, `dialog.err.user_cancelled`). Hub-level errors use the `SPORE.*` namespace and are produced by the hub itself, never by your node.

---

## Wire Protocol Cheat Sheet

The library handles all wire formatting, but this is what moves over the socket:

```
# Caller → Hub → Your node (hub injects cast=)
clock.get_time timezone=UTC ~h1 cast=dev.sporeos.cli

# Your node → Hub → Caller (hub injects ok and capture=)
~h1:clock.get_time time=2026-03-13T14:32:00Z

# Error from your node → Hub → Caller (hub injects capture=)
~h1:clock.get_time error code=clock.err.invalid_timezone what="Unknown timezone"
```

All messages are newline-delimited plaintext. No binary framing.

---

## Writing a Complete Node

Here is a complete example that mirrors what the existing `node-a` does, rewritten to use this library:

```go
package main

import (
	"fmt"
	"log"

	spore "github.com/sporeos-dev/spore-client-libs/go"
)

func main() {
	client := spore.NewClient("dev.sporeos.a")

	client.HandleRequest("aardvark", func(call *spore.Call) {
		fmt.Println("[node-a] received aardvark")
		call.Reply(map[string]string{"result": "ants"})
	})

	client.HandleRequest("aardvark.burrow", func(call *spore.Call) {
		fmt.Println("[node-a] received aardvark.burrow")
		call.Reply(map[string]string{"result": "underground"})
	})

	if err := client.Connect(); err != nil {
		log.Fatal("connect:", err)
	}
	defer client.Close()

	log.Println("node-a connected")
	if err := client.Listen(); err != nil {
		log.Println("disconnected:", err)
	}
}
```

---

## Using a Local Path (`replace` directive)

During development, before the library is published to GitHub, reference it by local path in the consuming project's `go.mod`:

```
module com.example.mynode

go 1.25.0

require github.com/sporeos-dev/spore-client-libs/go v0.0.0

replace github.com/sporeos-dev/spore-client-libs/go => ../spore-client-libs/go
```

Then in your source:
```go
import spore "github.com/sporeos-dev/spore-client-libs/go"
```

The `replace` directive tells Go to use the local directory instead of fetching from GitHub. Remove it when you're ready to publish and tag a release.

---

## File Structure

```
go/
├── go.mod          — Module declaration
├── client.go       — Client struct, Connect/Listen/Handle/Send/SendAndWait/Wait/WaitFor/Close
├── message.go      — Call/Response/WitnessMessage structs, parseCall/parseResponse/parseWitness, Reply/Error, splitFields
└── client_test.go  — Tests for parsing, dispatch, response formatting, wait, SendAndWait, and witness behaviour
```

The library has **no external dependencies** — only the Go standard library.

---

## Known Limitations (v1d0)

- **No reconnection logic.** If the daemon restarts, `Listen()` returns an error and the application must reconnect manually.
- **Single connection per client.** Each `Client` manages one connection. Running the same node ID from two processes simultaneously is undefined behaviour at the hub level.
- **No manifest validation.** The library does not read or validate the node's manifest YAML. Ensure your registered subjects match what is declared in the manifest.

---

## License

MIT License — see [LICENSE](../LICENSE) for details.
