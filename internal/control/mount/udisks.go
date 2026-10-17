package mount

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/username/sandsysm/internal/system"
)

const (
	udisksName      = "org.freedesktop.UDisks2"
	udisksRoot      = dbus.ObjectPath("/org/freedesktop/UDisks2")
	blockInterface  = "org.freedesktop.UDisks2.Block"
	filesystemIface = "org.freedesktop.UDisks2.Filesystem"
	driveInterface  = "org.freedesktop.UDisks2.Drive"
)

type Drive struct {
	ObjectPath        string
	Path              string
	Device            string
	MountPoint        string
	Size              uint64
	Removable         bool
	MediaRemovable    bool
	ConnectionBus     string
	HintSystem        bool
	HintAuto          bool
	IsSystemPartition bool
	SafeToUnmount     bool
	HasFilesystem     bool
}

func ListDrives() ([]Drive, error) {
	conn, err := system.GetDBusConn()
	if err != nil {
		return nil, err
	}
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	call := conn.Object(udisksName, udisksRoot).Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0,
	)
	if err := call.Store(&objects); err != nil {
		return nil, fmt.Errorf("get UDisks2 objects: %w", err)
	}

	type driveProperties struct {
		removable      bool
		mediaRemovable bool
		connectionBus  string
	}
	drivePropertiesByPath := make(map[dbus.ObjectPath]driveProperties)
	for path, interfaces := range objects {
		if drive, ok := interfaces[driveInterface]; ok {
			drivePropertiesByPath[path] = driveProperties{
				removable:      variantBool(drive["Removable"]),
				mediaRemovable: variantBool(drive["MediaRemovable"]),
				connectionBus:  variantString(drive["ConnectionBus"]),
			}
		}
	}
	drives := make([]Drive, 0)
	for path, interfaces := range objects {
		block, ok := interfaces[blockInterface]
		if !ok {
			continue
		}
		device := variantBytes(block["Device"])
		if device == "" {
			continue
		}
		parentDrive := drivePropertiesByPath[variantObjectPath(block["Drive"])]
		entry := Drive{
			ObjectPath:     string(path),
			Path:           string(path),
			Device:         device,
			Size:           variantUint64(block["Size"]),
			Removable:      parentDrive.removable,
			MediaRemovable: parentDrive.mediaRemovable,
			ConnectionBus:  parentDrive.connectionBus,
			HintSystem:     variantBool(block["HintSystem"]),
			HintAuto:       variantBool(block["HintAuto"]),
		}
		if filesystem, ok := interfaces[filesystemIface]; ok {
			entry.HasFilesystem = true
			entry.MountPoint = firstMountPoint(filesystem["MountPoints"])
		}
		entry.IsSystemPartition = isSystemPartition(entry)
		entry.SafeToUnmount = isSafeToUnmount(entry)
		drives = append(drives, entry)
	}
	sort.Slice(drives, func(i, j int) bool { return drives[i].Device < drives[j].Device })
	return drives, nil
}

func Mount(blockPath string) (string, error) {
	drive, err := findSafeDrive(blockPath)
	if err != nil {
		return "", err
	}
	if drive.MountPoint != "" {
		return drive.MountPoint, fmt.Errorf("device %s is already mounted at %s", drive.Device, drive.MountPoint)
	}
	conn, err := system.GetDBusConn()
	if err != nil {
		return "", err
	}
	var mountPoint string
	call := conn.Object(udisksName, dbus.ObjectPath(blockPath)).Call(
		filesystemIface+".Mount", 0, map[string]dbus.Variant{},
	)
	if err := call.Store(&mountPoint); err != nil {
		return "", fmt.Errorf("mount %s: %w", blockPath, err)
	}
	return mountPoint, nil
}

func Unmount(blockPath string) error {
	drive, err := findSafeDrive(blockPath)
	if err != nil {
		return err
	}
	if drive.MountPoint == "" {
		return fmt.Errorf("device %s is not mounted", drive.Device)
	}
	conn, err := system.GetDBusConn()
	if err != nil {
		return err
	}
	if err := conn.Object(udisksName, dbus.ObjectPath(blockPath)).Call(
		filesystemIface+".Unmount", 0, map[string]dbus.Variant{},
	).Err; err != nil {
		return fmt.Errorf("unmount %s: %w", blockPath, err)
	}
	return nil
}

func findSafeDrive(blockPath string) (Drive, error) {
	drives, err := ListDrives()
	if err != nil {
		return Drive{}, err
	}
	for _, drive := range drives {
		if drive.ObjectPath != blockPath && drive.Path != blockPath {
			continue
		}
		if !drive.SafeToUnmount || !drive.HasFilesystem {
			return Drive{}, fmt.Errorf("refusing operation on protected or non-removable device %s", drive.Device)
		}
		return drive, nil
	}
	return Drive{}, fmt.Errorf("UDisks device %s was not found", blockPath)
}

func isSystemPartition(drive Drive) bool {
	if drive.HintSystem {
		return true
	}
	mountPoint := filepath.Clean(drive.MountPoint)
	for _, protected := range []string{"/", "/boot", "/boot/efi", "/efi", "/home", "/var", "/usr", "/windows"} {
		if mountPoint == protected || strings.HasPrefix(mountPoint, protected+string(filepath.Separator)) {
			return true
		}
	}
	if strings.HasPrefix(drive.Device, "/dev/nvme0n1") && !strings.EqualFold(drive.ConnectionBus, "usb") {
		return true
	}
	return strings.HasPrefix(drive.Device, "/dev/nvme") && !drive.Removable
}

func isSafeToUnmount(drive Drive) bool {
	removableBus := strings.EqualFold(drive.ConnectionBus, "usb") || strings.EqualFold(drive.ConnectionBus, "sdio")
	removable := drive.Removable || drive.MediaRemovable || removableBus
	devicePathAllowed := strings.HasPrefix(drive.Device, "/dev/sd") ||
		strings.HasPrefix(drive.Device, "/dev/mmcblk") ||
		strings.HasPrefix(drive.Device, "/dev/bus/usb")
	return removable && devicePathAllowed && !drive.IsSystemPartition && !drive.HintSystem && drive.HasFilesystem
}

func variantBytes(value dbus.Variant) string {
	bytes, ok := value.Value().([]byte)
	if !ok {
		return ""
	}
	return strings.TrimRight(string(bytes), "\x00")
}

func variantUint64(value dbus.Variant) uint64 {
	size, _ := value.Value().(uint64)
	return size
}

func variantBool(value dbus.Variant) bool {
	flag, _ := value.Value().(bool)
	return flag
}

func variantString(value dbus.Variant) string {
	text, _ := value.Value().(string)
	return text
}

func variantObjectPath(value dbus.Variant) dbus.ObjectPath {
	path, _ := value.Value().(dbus.ObjectPath)
	return path
}

func firstMountPoint(value dbus.Variant) string {
	points, ok := value.Value().([][]byte)
	if !ok || len(points) == 0 {
		return ""
	}
	return strings.TrimRight(string(points[0]), "\x00")
}
