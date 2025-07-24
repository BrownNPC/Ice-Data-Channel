package main

import (
	"context"
	"fmt"
	"github.com/BrownNPC/Ice-Data-Channel/client"
	"github.com/BrownNPC/Ice-Data-Channel/server"
	"log/slog"
	"net"
	"os"
	"time"

	"encoding/json"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // log everything
	}))
	slog.SetDefault(logger)
	l, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	go server.Serve(l, "/ws")

	cfg := client.DefaultConfig("localhost:9090", "/ws")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	owner, err := client.NewOwner(ctx, OnConnect, cfg)
	if err != nil {
		panic(err)
	}

	//6 character room ID. Example: ABC123
	fmt.Println("created room with id", owner.RoomID)

	guest, err := client.NewGuest(ctx, owner.RoomID, cfg)
	if err != nil {
		panic(err)
	}
	for range 5 {
		time.Sleep(time.Second * 1)
		payload, err := json.Marshal(time.Now())
		if err != nil {
			panic("failed to marshal time")
		}
		guest.Conn().Write(payload)
	}
	<-exit
}

var exit = make(chan struct{})

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
