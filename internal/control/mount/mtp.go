package mount

import (
	"fmt"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

type MTPDevice struct {
	ID          string
	Description string
}

func ListMTP() ([]MTPDevice, error) {
	output, err := system.Run("lsusb")
	if err != nil {
		return []MTPDevice{}, nil
	}
	devices := make([]MTPDevice, 0)
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "mtp") && !strings.Contains(lower, "android") && !strings.Contains(lower, "portable device") {
			continue
		}
		fields := strings.Fields(line)
		id := ""
		for index, field := range fields {
			if field == "Device" && index+1 < len(fields) {
				id = fields[index+1]
				break
			}
		}
		devices = append(devices, MTPDevice{ID: id, Description: strings.TrimSpace(line)})
	}
	return devices, nil
}

func MountMTP(dev, mountpoint string) error {
	if strings.TrimSpace(mountpoint) == "" {
		return fmt.Errorf("mountpoint cannot be empty")
	}
	args := make([]string, 0, 2)
	if strings.TrimSpace(dev) != "" {
		args = append(args, "-device="+dev)
	}
	args = append(args, mountpoint)
	output, err := system.Run("jmtpfs", args...)
	if err != nil {
		return fmt.Errorf("mount MTP device: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}

func UnmountMTP(mountpoint string) error {
	output, err := system.Run("fusermount3", "-u", mountpoint)
	if err == nil {
		return nil
	}
	output, err = system.Run("fusermount", "-u", mountpoint)
	if err != nil {
		return fmt.Errorf("unmount MTP device: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}
