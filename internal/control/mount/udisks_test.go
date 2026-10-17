package mount

import "testing"

func TestDriveSafetyClassification(t *testing.T) {
	tests := []struct {
		name            string
		drive           Drive
		wantSystem      bool
		wantSafeUnmount bool
	}{
		{
			name: "NVMe EFI system partition",
			drive: Drive{
				Device:        "/dev/nvme0n1p1",
				MountPoint:    "/boot/efi",
				ConnectionBus: "nvme",
				HintSystem:    true,
				HasFilesystem: true,
			},
			wantSystem: true,
		},
		{
			name: "NVMe root partition",
			drive: Drive{
				Device:        "/dev/nvme0n1p3",
				MountPoint:    "/",
				ConnectionBus: "nvme",
				HasFilesystem: true,
			},
			wantSystem: true,
		},
		{
			name: "USB flash partition",
			drive: Drive{
				Device:        "/dev/sdb1",
				MountPoint:    "/run/media/user/USB",
				ConnectionBus: "usb",
				Removable:     true,
				HasFilesystem: true,
			},
			wantSafeUnmount: true,
		},
		{
			name: "SD card partition",
			drive: Drive{
				Device:        "/dev/mmcblk0p1",
				ConnectionBus: "sdio",
				Removable:     true,
				HasFilesystem: true,
			},
			wantSafeUnmount: true,
		},
		{
			name: "USB device flagged as system",
			drive: Drive{
				Device:        "/dev/sdc1",
				ConnectionBus: "usb",
				Removable:     true,
				HintSystem:    true,
				HasFilesystem: true,
			},
			wantSystem: true,
		},
		{
			name: "fixed SATA data partition",
			drive: Drive{
				Device:        "/dev/sda1",
				ConnectionBus: "ata",
				HasFilesystem: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			drive := test.drive
			drive.IsSystemPartition = isSystemPartition(drive)
			drive.SafeToUnmount = isSafeToUnmount(drive)
			if drive.IsSystemPartition != test.wantSystem {
				t.Errorf("IsSystemPartition = %t, want %t", drive.IsSystemPartition, test.wantSystem)
			}
			if drive.SafeToUnmount != test.wantSafeUnmount {
				t.Errorf("SafeToUnmount = %t, want %t", drive.SafeToUnmount, test.wantSafeUnmount)
			}
		})
	}
}

func TestProtectedMountPointUsesPathBoundaries(t *testing.T) {
	if !isSystemPartition(Drive{MountPoint: "/home/user"}) {
		t.Fatal("expected a home subdirectory mount to be protected")
	}
	if isSystemPartition(Drive{MountPoint: "/homebrew"}) {
		t.Fatal("did not expect /homebrew to be treated as a /home subdirectory")
	}
}
