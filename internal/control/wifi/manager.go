package wifi

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

const (
	nmSettingsPath    = dbus.ObjectPath("/org/freedesktop/NetworkManager/Settings")
	nmSettingsIface   = "org.freedesktop.NetworkManager.Settings"
	nmConnectionIface = "org.freedesktop.NetworkManager.Settings.Connection"
)

func Connect(ssid, password string) error {
	if ssid == "" {
		return fmt.Errorf("SSID cannot be empty")
	}
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	device, err := findWirelessDevice(conn, "")
	if err != nil {
		return err
	}

	connection := map[string]map[string]dbus.Variant{
		"connection": {
			"id":   dbus.MakeVariant(ssid),
			"type": dbus.MakeVariant("802-11-wireless"),
		},
		"802-11-wireless": {
			"ssid": dbus.MakeVariant([]byte(ssid)),
			"mode": dbus.MakeVariant("infrastructure"),
		},
		"ipv4": {"method": dbus.MakeVariant("auto")},
		"ipv6": {"method": dbus.MakeVariant("auto")},
	}
	if password != "" {
		connection["802-11-wireless-security"] = map[string]dbus.Variant{
			"key-mgmt": dbus.MakeVariant("wpa-psk"),
			"psk":      dbus.MakeVariant(password),
		}
		wireless := connection["802-11-wireless"]
		wireless["security"] = dbus.MakeVariant("802-11-wireless-security")
	}

	var connectionPath, activePath dbus.ObjectPath
	call := conn.Object(nmName, nmPath).Call(
		"org.freedesktop.NetworkManager.AddAndActivateConnection", 0,
		connection, device, dbus.ObjectPath("/"),
	)
	if err := call.Store(&connectionPath, &activePath); err != nil {
		return fmt.Errorf("connect to Wi-Fi network %q: %w", ssid, err)
	}
	return nil
}

func Disconnect(iface string) error {
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	device, err := findWirelessDevice(conn, iface)
	if err != nil {
		return err
	}
	if err := conn.Object(nmName, device).Call(nmDeviceInterface+".Disconnect", 0).Err; err != nil {
		return fmt.Errorf("disconnect Wi-Fi interface %q: %w", iface, err)
	}
	return nil
}

func Forget(ssid string) error {
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	var paths []dbus.ObjectPath
	if err := conn.Object(nmName, nmSettingsPath).Call(nmSettingsIface+".ListConnections", 0).Store(&paths); err != nil {
		return fmt.Errorf("list saved NetworkManager connections: %w", err)
	}
	for _, path := range paths {
		var settings map[string]map[string]dbus.Variant
		if err := conn.Object(nmName, path).Call(nmConnectionIface+".GetSettings", 0).Store(&settings); err != nil {
			continue
		}
		wifiSettings := settings["802-11-wireless"]
		networkSSID, ok := wifiSettings["ssid"].Value().([]byte)
		if !ok || string(networkSSID) != ssid {
			continue
		}
		if err := conn.Object(nmName, path).Call(nmConnectionIface+".Delete", 0).Err; err != nil {
			return fmt.Errorf("forget Wi-Fi network %q: %w", ssid, err)
		}
	}
	return nil
}

func GetSecret(ssid string) (string, error) {
	conn, err := system.GetDBusConn()
	if err != nil {
		return "", err
	}
	var paths []dbus.ObjectPath
	if err := conn.Object(nmName, nmSettingsPath).Call(nmSettingsIface+".ListConnections", 0).Store(&paths); err != nil {
		return "", fmt.Errorf("list saved NetworkManager connections: %w", err)
	}
	for _, path := range paths {
		object := conn.Object(nmName, path)
		var settings map[string]map[string]dbus.Variant
		if err := object.Call(nmConnectionIface+".GetSettings", 0).Store(&settings); err != nil {
			continue
		}
		networkSSID, ok := settings["802-11-wireless"]["ssid"].Value().([]byte)
		if !ok || string(networkSSID) != ssid {
			continue
		}
		var secrets map[string]map[string]dbus.Variant
		if err := object.Call(nmConnectionIface+".GetSecrets", 0, "802-11-wireless-security").Store(&secrets); err != nil {
			return "", fmt.Errorf("read Wi-Fi secret for %q: %w", ssid, err)
		}
		psk, ok := secrets["802-11-wireless-security"]["psk"].Value().(string)
		if !ok {
			return "", fmt.Errorf("no saved PSK for Wi-Fi network %q", ssid)
		}
		return psk, nil
	}
	return "", fmt.Errorf("saved Wi-Fi network %q not found", ssid)
}

func findWirelessDevice(conn *dbus.Conn, iface string) (dbus.ObjectPath, error) {
	var devices []dbus.ObjectPath
	if err := conn.Object(nmName, nmPath).Call("org.freedesktop.NetworkManager.GetDevices", 0).Store(&devices); err != nil {
		return "", fmt.Errorf("get NetworkManager devices: %w", err)
	}
	for _, path := range devices {
		object := conn.Object(nmName, path)
		deviceType, err := object.GetProperty(nmDeviceInterface + ".DeviceType")
		if err != nil || variantUint32(deviceType) != 2 {
			continue
		}
		if iface == "" {
			return path, nil
		}
		name, err := object.GetProperty(nmDeviceInterface + ".Interface")
		if err == nil && strings.EqualFold(variantString(name), iface) {
			return path, nil
		}
	}
	if iface == "" {
		return "", fmt.Errorf("no Wi-Fi interface found")
	}
	return "", fmt.Errorf("Wi-Fi interface %q not found", iface)
}
