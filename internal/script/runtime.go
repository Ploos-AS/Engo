package script

import (
	"fmt"
	"os"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// RunFile executes a Tengo script in the initial Engo sandbox.
func RunFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	s := tengo.NewScript(src)
	s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	if _, err := s.Run(); err != nil {
		return fmt.Errorf("run script: %w", err)
	}
	return nil
}
