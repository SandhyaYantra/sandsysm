package bluetooth

import (
	"fmt"
	"sort"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

const (
	bluezName       = "org.bluez"
	bluezRoot       = dbus.ObjectPath("/")
	adapterIface    = "org.bluez.Adapter1"
	deviceIface     = "org.bluez.Device1"
	objectManagerIF = "org.freedesktop.DBus.ObjectManager"
)

type BTDevice struct {
	Name      string
	Address   string
	RSSI      int16
	Paired    bool
	Connected bool
}

func StartDiscovery() error {
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	path, err := firstAdapter(conn)
	if err != nil {
		return err
	}
	if err := conn.Object(bluezName, path).Call(adapterIface+".StartDiscovery", 0).Err; err != nil {
		return fmt.Errorf("start Bluetooth discovery: %w", err)
	}
	return nil
}

func StopDiscovery() error {
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	path, err := firstAdapter(conn)
	if err != nil {
		return err
	}
	if err := conn.Object(bluezName, path).Call(adapterIface+".StopDiscovery", 0).Err; err != nil {
		return fmt.Errorf("stop Bluetooth discovery: %w", err)
	}
	return nil
}

func ListDevices() ([]BTDevice, error) {
	conn, err := system.GetDBusConn()
	if err != nil {
		return nil, err
	}
	objects, err := managedObjects(conn)
	if err != nil {
		return nil, err
	}
	devices := make([]BTDevice, 0)
	for _, interfaces := range objects {
		properties, ok := interfaces[deviceIface]
		if !ok {
			continue
		}
		name := propertyString(properties["Name"])
		if name == "" {
			name = propertyString(properties["Alias"])
		}
		devices = append(devices, BTDevice{
			Name:      name,
			Address:   propertyString(properties["Address"]),
			RSSI:      propertyInt16(properties["RSSI"]),
			Paired:    propertyBool(properties["Paired"]),
			Connected: propertyBool(properties["Connected"]),
		})
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })
	return devices, nil
}

func managedObjects(conn *dbus.Conn) (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, error) {
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := conn.Object(bluezName, bluezRoot).Call(objectManagerIF+".GetManagedObjects", 0).Store(&objects); err != nil {
		return nil, fmt.Errorf("get BlueZ objects: %w", err)
	}
	return objects, nil
}

func firstAdapter(conn *dbus.Conn) (dbus.ObjectPath, error) {
	objects, err := managedObjects(conn)
	if err != nil {
		return "", err
	}
	for path, interfaces := range objects {
		if _, ok := interfaces[adapterIface]; ok {
			return path, nil
		}
	}
	return "", fmt.Errorf("no Bluetooth adapter found")
}

func propertyString(value dbus.Variant) string {
	text, _ := value.Value().(string)
	return text
}

func propertyBool(value dbus.Variant) bool {
	flag, _ := value.Value().(bool)
	return flag
}

func propertyInt16(value dbus.Variant) int16 {
	switch number := value.Value().(type) {
	case int16:
		return number
	case int32:
		return int16(number)
	}
	return 0
}
