package control

import (
	"fmt"
	"net"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

func SetInterfaceUp(iface string, up bool) error {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return fmt.Errorf("interface name cannot be empty")
	}
	if _, err := net.InterfaceByName(iface); err != nil {
		return fmt.Errorf("find network interface %q: %w", iface, err)
	}
	state := "down"
	if up {
		state = "up"
	}
	output, err := system.Run("ip", "link", "set", "dev", iface, state)
	if err != nil {
		return fmt.Errorf("set interface %q %s: %w: %s", iface, state, err, strings.TrimSpace(output))
	}
	return nil
}
