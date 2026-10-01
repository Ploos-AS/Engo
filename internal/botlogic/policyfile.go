package botlogic

import (
	"fmt"
	"os"
	"strings"
)

const maxConsultSource = 64 << 10

func LoadPolicyFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("policy file is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", fmt.Errorf("policy file is empty")
	}
	if len(b) > maxConsultSource {
		return "", fmt.Errorf("ruleset source exceeds BotLogic 64 KiB limit")
	}
	source := string(b)
	if !strings.Contains(source, "may_execute(") {
		return "", fmt.Errorf("policy must define may_execute/2")
	}
	return source, nil
}
