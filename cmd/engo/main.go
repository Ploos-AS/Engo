package main

import (
	"fmt"
	"os"

	"github.com/Ploos-AS/Engo/internal/script"
)

func main() {
	if err := script.RunFile("scripts/examples/hello.tengo"); err != nil {
		fmt.Fprintln(os.Stderr, "engo:", err)
		os.Exit(1)
	}
}
