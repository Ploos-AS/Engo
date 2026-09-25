package script

import "fmt"

type Capabilities struct {
	HTTP bool
}

func (c Capabilities) Require(name string) error {
	switch name {
	case "http":
		if c.HTTP { return nil }
	default:
		return fmt.Errorf("unknown capability %q", name)
	}
	return fmt.Errorf("capability %q is not granted", name)
}
