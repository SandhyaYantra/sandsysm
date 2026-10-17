package ui

import (
	"strings"
	"testing"

	"github.com/username/sandsysm/internal/control/mount"
	"github.com/username/sandsysm/internal/monitor"
	"github.com/username/sandsysm/internal/ui/components"
)

func TestRenderDiskTabSeparatesSystemAndRemovableDrives(t *testing.T) {
	drives := []mount.Drive{
		{
			Device:            "/dev/nvme0n1p1",
			MountPoint:        "/boot/efi",
			Size:              512 * 1024 * 1024,
			IsSystemPartition: true,
			HasFilesystem:     true,
		},
		{
			Device:        "/dev/sdb1",
			MountPoint:    "/run/media/user/USB",
			Size:          32 * 1024 * 1024 * 1024,
			Removable:     true,
			ConnectionBus: "usb",
			SafeToUnmount: true,
			HasFilesystem: true,
		},
	}
	usage := []monitor.DiskStat{
		{Mount: "/boot/efi", Used: 1, Total: 1, Percent: 50},
		{Mount: "/run/media/user/USB", Used: 2, Total: 32, Percent: 6},
	}
	view := RenderDiskTab(usage, drives, components.ListModel{})
	for _, expected := range []string{"[SYSTEM]", "/dev/nvme0n1p1", "[REMOVABLE] /dev/sdb1", "read-only"} {
		if !strings.Contains(view, expected) {
			t.Errorf("Disk panel missing %q", expected)
		}
	}
}

func TestRenderDiskTabDoesNotOfferInternalDriveAsActionable(t *testing.T) {
	drives := []mount.Drive{{
		Device:            "/dev/nvme0n1p3",
		MountPoint:        "/",
		IsSystemPartition: true,
		HasFilesystem:     true,
	}}
	view := RenderDiskTab(nil, drives, components.ListModel{})
	if !strings.Contains(view, "[SYSTEM]") {
		t.Fatal("expected the system partition to be labeled")
	}
	if !strings.Contains(view, "No removable drive") {
		t.Fatal("expected the actionable list to exclude internal storage")
	}
	if strings.Contains(view, "> /dev/nvme0n1p3") {
		t.Fatal("system partition appeared as an actionable selection")
	}
}
