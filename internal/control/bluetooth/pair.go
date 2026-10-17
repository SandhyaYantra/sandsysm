package bluetooth

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

func Pair(address string) error {
	return callDevice(address, "Pair")
}

func Trust(address string) error {
	conn, path, err := findDevice(address)
	if err != nil {
		return err
	}
	if err := conn.Object(bluezName, path).Call(
		"org.freedesktop.DBus.Properties.Set", 0,
		deviceIface, "Trusted", dbus.MakeVariant(true),
	).Err; err != nil {
		return fmt.Errorf("trust Bluetooth device %q: %w", address, err)
	}
	return nil
}

func ConnectBT(address string) error {
	return callDevice(address, "Connect")
}

func callDevice(address, method string) error {
	conn, path, err := findDevice(address)
	if err != nil {
		return err
	}
	if err := conn.Object(bluezName, path).Call(deviceIface+"."+method, 0).Err; err != nil {
		return fmt.Errorf("%s Bluetooth device %q: %w", strings.ToLower(method), address, err)
	}
	return nil
}

func findDevice(address string) (*dbus.Conn, dbus.ObjectPath, error) {
	conn, err := system.GetDBusConn()
	if err != nil {
		return nil, "", err
	}
	objects, err := managedObjects(conn)
	if err != nil {
		return nil, "", err
	}
	for path, interfaces := range objects {
		properties, ok := interfaces[deviceIface]
		if !ok {
			continue
		}
		if strings.EqualFold(propertyString(properties["Address"]), address) {
			return conn, path, nil
		}
	}
	return nil, "", fmt.Errorf("Bluetooth device %q not found", address)
}
