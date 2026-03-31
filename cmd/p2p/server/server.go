package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const ProtocolID = protocol.ID("/sst/1.0.0")

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	// Create a libp2p host with AutoRelay enabled.
	host, err := libp2p.New(libp2p.EnableAutoNATv2(), libp2p.EnableAutoRelay())
	if err != nil {
		fmt.Println("Error creating host:", err)
		os.Exit(1)
	}
	defer host.Close()

	// Set a simple stream handler for our protocol.
	host.SetStreamHandler(ProtocolID, func(s network.Stream) {
		defer s.Close()
		// Read all data sent on this stream.
		data, err := io.ReadAll(s)
		if err != nil {
			fmt.Println("Error reading stream:", err)
			return
		}
		fmt.Printf("Received: %s\n", string(data))
	})

	// Print our peer ID and addresses.
	fmt.Println("Server is running!")
	fmt.Println("Peer ID:", host.ID().String())
	fmt.Println("Listening addresses:")
	for _, addr := range host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host.ID().String())
	}

	// Block forever.
	<-ctx.Done()
}
