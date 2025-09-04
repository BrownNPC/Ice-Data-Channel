
# Ice Data Channel

**Ice Data Channel** is a lightweight signaling server and client library for establishing **UDP peer-to-peer connections** between clients behind NAT, using the ICE protocol.

This library enables NAT traversal via ICE and simplifies the setup process with a built-in signaling server over WebSockets.


## Features

* Built-in signaling server (WebSocket-based)
* ICE support for NAT traversal
* Peer-to-peer UDP communication
* Simple and clean API for clients
* Minimal dependencies

---

## Example

[Full Example](https://github.com/BrownNPC/Ice-Data-Channel/blob/master/example/pingPong/main.go)

```go
func main() {
	l, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}

	// Start signaling server
	go server.Serve(l, "/ws")

	cfg := client.DefaultConfig("localhost:9090", "/ws")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Room owner listens for guests
	owner, err := client.NewOwner(ctx, OnConnect, cfg)
	if err != nil {
		panic(err)
	}

	//6 character room ID. Example: ABC123
	fmt.Println("created room with id", owner.RoomID)
	
	// Guest joins using the owner's room ID
	guest, err := client.NewGuest(ctx, owner.RoomID, cfg)
	if err != nil {
		panic(err)
	}

	// Peer-to-peer connection established!
	for range 5 {
		time.Sleep(time.Second)
		payload, err := json.Marshal(time.Now())
		if err != nil {
			panic("failed to marshal time")
		}
		guest.Conn().Write(payload)
	}

	select {} // keep alive
}

func OnConnect(conn client.Conn) {
	var buf [1500]byte
	fmt.Println("new connection!")
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Println(err)
			close(exit)
			return
		}
		t := time.Time{}
		err = json.Unmarshal(buf[:n], &t)
		if err != nil {
			panic("failed to unmarshal time " + err.Error())
		}
		fmt.Println(time.Since(t))
	}
}
```

---

## How It Works


[Here's a great presentation that explains how ice works and why we need it](https://youtu.be/FExZvpVvYxA?t=677)


1. **Start a Signaling Server**
   The signaling server uses WebSockets to allow clients to discover and exchange ICE candidates.

2. **Client Connection**

   * One peer becomes the **owner**, opening a room.
   * Other peers join as a **guest** using the owner's room ID.
   * Both exchange ICE candidates through the signaling server.

3. **P2P Connection**
   Once ICE negotiation is complete, a **direct UDP connection** is established between peers.
   The signaling server is no longer involved for sending traffic.
   It only listens for more peers that want to connect, and kicks inactive peers


---

## Usage

1. Run a signaling server (publicly accessible).
2. Use `client.NewOwner()` to create a host.
3. Use `client.NewGuest()` to connect to the host with the provided room ID.
4. Send and receive data using `*ice.Conn`.

---

## TODO

* [ ] Figure out what to do


---

## 📄 License

MIT © [BrownNPC](https://github.com/BrownNPC)

---
