package main

import (
	"context"
	"dc/client"
	"dc/server"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"encoding/json"
	"github.com/google/uuid"
	"github.com/pion/ice/v4"
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
	select {}
}
func OnConnect(id uuid.UUID, conn *ice.Conn) {
	var buf [1500]byte
	fmt.Println("new connection!")
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Println(err)
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
