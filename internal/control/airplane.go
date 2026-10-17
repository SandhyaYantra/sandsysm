package control

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

const (
	networkManagerName = "org.freedesktop.NetworkManager"
	networkManagerPath = dbus.ObjectPath("/org/freedesktop/NetworkManager")
	networkManagerIF   = "org.freedesktop.NetworkManager"
)

func GetAirplaneMode() (bool, error) {
	output, err := system.Run("rfkill", "list")
	if err == nil {
		var relevant, currentRelevant bool
		blockedStates := make([]bool, 0, 2)
		for _, line := range strings.Split(output, "\n") {
			trimmed := strings.TrimSpace(strings.ToLower(line))
			if strings.Contains(trimmed, "wireless lan") || strings.Contains(trimmed, "wireless wlan") || strings.Contains(trimmed, "bluetooth") {
				relevant = true
				currentRelevant = true
				continue
			}
			if strings.Contains(trimmed, "wireless") || strings.Contains(trimmed, "wireless wan") || strings.Contains(trimmed, "wireless wan") {
				currentRelevant = false
			}
			if strings.HasPrefix(trimmed, "soft blocked:") && currentRelevant {
				blockedStates = append(blockedStates, strings.HasSuffix(trimmed, "yes"))
				currentRelevant = false
			}
		}
		if relevant && len(blockedStates) > 0 {
			allBlocked := true
			for _, blocked := range blockedStates {
				if !blocked {
					allBlocked = false
					break
				}
			}
			return allBlocked, nil
		}
	}

	conn, connErr := system.GetDBusConn()
	if connErr != nil {
		if err != nil {
			return false, fmt.Errorf("read rfkill state: %w; connect to NetworkManager: %v", err, connErr)
		}
		return false, connErr
	}
	value, err := conn.Object(networkManagerName, networkManagerPath).GetProperty(networkManagerIF + ".NetworkingEnabled")
	if err != nil {
		return false, fmt.Errorf("read NetworkManager state: %w", err)
	}
	enabled, ok := value.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected NetworkManager NetworkingEnabled type %T", value.Value())
	}
	return !enabled, nil
}

func SetAirplaneMode(on bool) error {
	operation := "unblock"
	if on {
		operation = "block"
	}
	output, rfkillErr := system.Run("rfkill", operation, "all")
	if rfkillErr != nil {
		rfkillErr = fmt.Errorf("rfkill %s: %w: %s", operation, rfkillErr, strings.TrimSpace(output))
	}

	conn, err := system.GetDBusConn()
	if err != nil {
		if rfkillErr != nil {
			return fmt.Errorf("%v; connect to NetworkManager: %w", rfkillErr, err)
		}
		return err
	}
	if err := conn.Object(networkManagerName, networkManagerPath).Call(
		"org.freedesktop.DBus.Properties.Set", 0,
		networkManagerIF, "NetworkingEnabled", dbus.MakeVariant(!on),
	).Err; err != nil {
		if rfkillErr != nil {
			return fmt.Errorf("%v; set NetworkManager networking: %w", rfkillErr, err)
		}
		return fmt.Errorf("set NetworkManager networking: %w", err)
	}
	return rfkillErr
}
