package wifi

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

const (
	nmName              = "org.freedesktop.NetworkManager"
	nmPath              = dbus.ObjectPath("/org/freedesktop/NetworkManager")
	nmDeviceInterface   = "org.freedesktop.NetworkManager.Device"
	nmWirelessInterface = "org.freedesktop.NetworkManager.Device.Wireless"
	nmAPInterface       = "org.freedesktop.NetworkManager.AccessPoint"
)

type AP struct {
	SSID      string
	Signal    int
	BSSID     string
	Freq      uint32
	Security  string
	Connected bool
}

func Scan() ([]AP, error) {
	conn, err := system.GetDBusConn()
	if err != nil {
		return nil, err
	}
	var devices []dbus.ObjectPath
	if err := conn.Object(nmName, nmPath).Call("org.freedesktop.NetworkManager.GetDevices", 0).Store(&devices); err != nil {
		return nil, fmt.Errorf("get NetworkManager devices: %w", err)
	}

	type deviceAccessPoints struct {
		paths  []dbus.ObjectPath
		active dbus.ObjectPath
	}
	deviceResults := make([]deviceAccessPoints, 0)
	activeAccessPoints := make(map[dbus.ObjectPath]bool)
	for _, path := range devices {
		device := conn.Object(nmName, path)
		deviceType, err := device.GetProperty(nmDeviceInterface + ".DeviceType")
		if err != nil || variantUint32(deviceType) != 2 {
			continue
		}
		wireless := conn.Object(nmName, path)
		_ = wireless.Call(nmWirelessInterface+".RequestScan", 0, map[string]dbus.Variant{}).Err
		activeVariant, _ := wireless.GetProperty(nmWirelessInterface + ".ActiveAccessPoint")
		activePath := variantObjectPath(activeVariant)
		if activePath != "/" {
			activeAccessPoints[activePath] = true
		}
		var deviceAPs []dbus.ObjectPath
		if err := wireless.Call(nmWirelessInterface+".GetAccessPoints", 0).Store(&deviceAPs); err != nil {
			continue
		}
		deviceResults = append(deviceResults, deviceAccessPoints{paths: deviceAPs, active: activePath})
	}

	results := make([]AP, 0)
	for _, deviceResult := range deviceResults {
		for _, path := range deviceResult.paths {
			object := conn.Object(nmName, path)
			ssid, _ := object.GetProperty(nmAPInterface + ".Ssid")
			strength, _ := object.GetProperty(nmAPInterface + ".Strength")
			bssid, _ := object.GetProperty(nmAPInterface + ".HwAddress")
			frequency, _ := object.GetProperty(nmAPInterface + ".Frequency")
			flags, _ := object.GetProperty(nmAPInterface + ".Flags")
			wpaFlags, _ := object.GetProperty(nmAPInterface + ".WpaFlags")
			rsnFlags, _ := object.GetProperty(nmAPInterface + ".RsnFlags")
			results = append(results, AP{
				SSID:      variantSSID(ssid),
				Signal:    variantInt(strength),
				BSSID:     variantString(bssid),
				Freq:      variantUint32(frequency),
				Security:  accessPointSecurity(flags, wpaFlags, rsnFlags),
				Connected: activeAccessPoints[path],
			})
		}
	}
	return results, nil
}

func variantSSID(value dbus.Variant) string {
	ssid, _ := value.Value().([]byte)
	return string(ssid)
}

func variantString(value dbus.Variant) string {
	text, _ := value.Value().(string)
	return text
}

func variantUint32(value dbus.Variant) uint32 {
	switch number := value.Value().(type) {
	case uint32:
		return number
	case int32:
		if number > 0 {
			return uint32(number)
		}
	}
	return 0
}

func variantObjectPath(value dbus.Variant) dbus.ObjectPath {
	path, _ := value.Value().(dbus.ObjectPath)
	return path
}

func variantInt(value dbus.Variant) int {
	switch number := value.Value().(type) {
	case byte:
		return int(number)
	case uint32:
		return int(number)
	case int32:
		return int(number)
	}
	return 0
}

func accessPointSecurity(flags, wpaFlags, rsnFlags dbus.Variant) string {
	legacy := variantUint32(flags)
	wpa := variantUint32(wpaFlags)
	rsn := variantUint32(rsnFlags)
	if legacy == 0 && wpa == 0 && rsn == 0 {
		return "Open"
	}
	if rsn != 0 && wpa != 0 {
		return "WPA/WPA2"
	}
	if rsn != 0 {
		return "WPA2/WPA3"
	}
	if wpa != 0 {
		return "WPA"
	}
	return "WEP"
}
