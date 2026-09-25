package script

import "fmt"

type Capabilities struct {
	HTTP bool
}

func (c Capabilities) Require(name string) error {
	switch name {
	case "http":
		if c.HTTP {
			return nil
		}
		return fmt.Errorf("capability %q is not granted", name)
	default:
		return fmt.Errorf("unknown capability %q", name)
	}
}
