package main

import (
	"fmt"
	"os"

	"github.com/Ploos-AS/Engo/internal/pbmp"
)

func main() {
	socket := os.Getenv("ENGO_PBMP_SOCKET")
	if socket == "" {
		fmt.Fprintln(os.Stderr, "ENGO_PBMP_SOCKET is required")
		os.Exit(2)
	}
	state := pbmp.NewState("engo-qualification", "irc.example.invalid")
	if err := pbmp.Serve(socket, state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
