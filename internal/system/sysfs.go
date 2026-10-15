package system

import (
	"os"
	"strings"
)

func ReadSysfs(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteSysfs(path, value string) error {
	return os.WriteFile(path, []byte(value), 0o644)
}
