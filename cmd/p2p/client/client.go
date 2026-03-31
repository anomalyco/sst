package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"log/slog"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const RendezvousProtocol = protocol.ID("/rendezvous/1.0.0")

// sendCommand opens a new stream to the given peer, sends a command, and prints the response.
func sendCommand(ctx context.Context, h host.Host, peerID peer.ID, cmd string) error {
	s, err := h.NewStream(ctx, peerID, RendezvousProtocol)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer s.Close()

	// Send the command (e.g. "REGISTER <rendezvous>\n" or "DISCOVER <rendezvous>\n")
	_, err = s.Write([]byte(cmd))
	if err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	// Read and print each line of response.
	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			fmt.Println("Response:", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	return nil
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	// Command-line flags:
	//   -server: the multiaddress of the rendezvous server (including /p2p/<peerID>).
	//   -rendezvous: the rendezvous point name.
	var serverAddrStr string
	var rendezvousPoint string
	flag.StringVar(&serverAddrStr, "server", "", "Multiaddress of the rendezvous server (e.g., /ip4/127.0.0.1/tcp/12345/p2p/<peerID>)")
	flag.StringVar(&rendezvousPoint, "rendezvous", "example", "Rendezvous point name")
	flag.Parse()

	if serverAddrStr == "" {
		slog.Error("Please provide the server multiaddress using -server")
		os.Exit(1)
	}

	// Convert the string to a peer.AddrInfo.
	serverAddr, err := peer.AddrInfoFromString(serverAddrStr)
	if err != nil {
		slog.Error("Error parsing server multiaddress", "error", err)
		os.Exit(1)
	}

	// Create a new libp2p host for the client.
	h, err := libp2p.New()
	if err != nil {
		slog.Error("Failed to create client host", "error", err)
		os.Exit(1)
	}
	defer h.Close()

	fmt.Println("Client running with Peer ID:", h.ID().String())
	for _, addr := range h.Addrs() {
		fmt.Printf("Listening on: %s/p2p/%s\n", addr, h.ID().String())
	}

	// Connect to the rendezvous server.
	if err := h.Connect(ctx, *serverAddr); err != nil {
		slog.Error("Failed to connect to server", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to rendezvous server")

	// Register at the rendezvous point.
	slog.Info("Registering at rendezvous point", "rendezvous", rendezvousPoint)
	if err := sendCommand(ctx, h, serverAddr.ID, fmt.Sprintf("REGISTER %s\n", rendezvousPoint)); err != nil {
		slog.Error("Registration failed", "error", err)
		os.Exit(1)
	}

	// Wait a moment (so other peers might also register).
	time.Sleep(2 * time.Second)

	// Discover peers registered under the same rendezvous point.
	slog.Info("Discovering peers at rendezvous point", "rendezvous", rendezvousPoint)
	if err := sendCommand(ctx, h, serverAddr.ID, fmt.Sprintf("DISCOVER %s\n", rendezvousPoint)); err != nil {
		slog.Error("Discovery failed", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
}

