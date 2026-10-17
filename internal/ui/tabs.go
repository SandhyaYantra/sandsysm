package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/username/sandsysm/internal/control"
	"github.com/username/sandsysm/internal/control/audio"
	"github.com/username/sandsysm/internal/control/bluetooth"
	"github.com/username/sandsysm/internal/control/mount"
	"github.com/username/sandsysm/internal/control/wifi"
	"github.com/username/sandsysm/internal/monitor"
	"github.com/username/sandsysm/internal/ui/components"
)

func RenderMonitorPanel(ram monitor.RamStat, cpu monitor.CpuStat, gpu monitor.GpuStat, brightness, brightnessMax, keyboardBrightness, keyboardMax int) string {
	return RenderCPUPanel(cpu) + "\n" + RenderMemoryPanel(ram, gpu, brightness, brightnessMax, keyboardBrightness, keyboardMax)
}

func RenderCPUPanel(cpu monitor.CpuStat) string {
	var view strings.Builder
	view.WriteString(components.RenderGauge("Total", cpu.Total) + "\n")
	cores := make([]string, 0, len(cpu.PerCore))
	for index, percentage := range cpu.PerCore {
		cores = append(cores, components.RenderGauge(fmt.Sprintf("C%d", index+1), percentage))
	}
	left, right := make([]string, 0, (len(cores)+1)/2), make([]string, 0, len(cores)/2)
	for index, core := range cores {
		if index%2 == 0 {
			left = append(left, core)
		} else {
			right = append(right, core)
		}
	}
	rows := len(left)
	if len(right) > rows {
		rows = len(right)
	}
	for row := 0; row < rows; row++ {
		leftText, rightText := "", ""
		if row < len(left) {
			leftText = left[row]
		}
		if row < len(right) {
			rightText = right[row]
		}
		view.WriteString(fmt.Sprintf("%-26s %s\n", leftText, rightText))
	}
	return view.String()
}

func RenderMemoryPanel(ram monitor.RamStat, gpu monitor.GpuStat, brightness, brightnessMax, keyboardBrightness, keyboardMax int) string {
	var view strings.Builder
	if ram.Total == 0 {
		view.WriteString("RAM Waiting for data\n")
	} else {
		percent := float64(ram.Used) / float64(ram.Total) * 100
		fmt.Fprintf(&view, "RAM %s | Free %d MB | Available %d MB | Cache %d MB | Buffers %d MB\n",
			components.RenderBar(ram.Used*1024*1024, ram.Total*1024*1024, percent),
			ram.Free, ram.Available, ram.Cache, ram.Buffers)
	}
	name := gpu.Name
	if name == "" {
		name = "GPU"
	}
	view.WriteString(components.RenderGauge(name, gpu.Percent) + "\n")
	if brightnessMax > 0 {
		percent := brightness * 100 / brightnessMax
		view.WriteString(fmt.Sprintf("Display %s %d%% (B +/-)\n", components.RenderSlider(percent, 100, ""), percent))
	} else {
		view.WriteString("Display brightness unavailable\n")
	}
	if keyboardMax > 0 {
		percent := keyboardBrightness * 100 / keyboardMax
		view.WriteString(fmt.Sprintf("Kbd %s %d%% (K +/-)\n", components.RenderSlider(percent, 100, ""), percent))
	} else {
		view.WriteString("Keyboard backlight unavailable\n")
	}
	return view.String()
}

func RenderBatteryPanel(battery monitor.BatteryStat, profiles []control.PowerProfile, active control.PowerProfile, selected int, airplane bool) string {
	var view strings.Builder
	view.WriteString(StyleTitle.Render("BATTERY / POWER") + "\n")
	if battery.Capacity < 0 {
		view.WriteString("No battery detected\n")
	} else {
		fmt.Fprintf(&view, "Battery %d%%  %s  %.2f W  %.2f V\n", battery.Capacity, battery.Status, battery.Watts, battery.Voltage)
	}
	airplaneStatus := "Off"
	if airplane {
		airplaneStatus = "On (radios blocked)"
	}
	view.WriteString("Airplane mode: " + airplaneStatus + " (A toggle)\n")
	view.WriteString("Power profile (j/k select, Enter apply, P cycle):\n")
	if len(profiles) == 0 {
		view.WriteString("  powerprofilesctl unavailable\n")
	} else {
		for index, profile := range profiles {
			marker := "  "
			if index == selected {
				marker = "> "
			}
			activeMarker := " "
			if profile == active {
				activeMarker = "*"
			}
			line := fmt.Sprintf("%s%s %s", marker, activeMarker, profile)
			if index == selected {
				line = StyleTitle.Render(line)
			}
			view.WriteString(line + "\n")
		}
	}
	return view.String()
}

func RenderDiskTab(disks []monitor.DiskStat, drives []mount.Drive, list components.ListModel) string {
	var view strings.Builder
	view.WriteString(StyleTitle.Render("Internal / mounted filesystems (read-only)") + "\n")
	visibleInternal := 0
	seenMounts := make(map[string]bool)
	for _, drive := range drives {
		if drive.SafeToUnmount || drive.MountPoint == "" {
			continue
		}
		tag := "[READ-ONLY]"
		if drive.IsSystemPartition || drive.HintSystem || isProtectedMountPoint(drive.MountPoint) {
			tag = "[SYSTEM]"
		}
		line := fmt.Sprintf("%s %s %s", tag, drive.Device, drive.MountPoint)
		if usage, ok := diskUsageAt(disks, drive.MountPoint); ok && usage.Total > 0 {
			line += " " + components.RenderBar(usage.Used*1024*1024*1024, usage.Total*1024*1024*1024, usage.Percent)
		} else {
			line += fmt.Sprintf(" %.1f GB (usage unavailable)", float64(drive.Size)/(1024*1024*1024))
		}
		view.WriteString(fitLine(line, 80) + "\n")
		seenMounts[drive.MountPoint] = true
		visibleInternal++
	}
	for _, disk := range disks {
		if disk.Total == 0 || seenMounts[disk.Mount] {
			continue
		}
		if drive, ok := driveMountedAt(drives, disk.Mount); ok && drive.SafeToUnmount {
			continue
		}
		tag := "[READ-ONLY]"
		device := ""
		if drive, ok := driveMountedAt(drives, disk.Mount); ok {
			device = drive.Device + " "
			if drive.IsSystemPartition || drive.HintSystem {
				tag = "[SYSTEM]"
			}
		} else if isProtectedMountPoint(disk.Mount) {
			tag = "[SYSTEM]"
		}
		line := fmt.Sprintf("%s %s%s %s", tag, device, disk.Mount,
			components.RenderBar(disk.Used*1024*1024*1024, disk.Total*1024*1024*1024, disk.Percent))
		view.WriteString(fitLine(line, 80) + "\n")
		seenMounts[disk.Mount] = true
		visibleInternal++
	}
	if visibleInternal == 0 {
		view.WriteString("  No mounted filesystems found.\n")
	}
	view.WriteString("  Internal/system partitions are read-only; Enter cannot unmount them.\n")

	view.WriteString("\n" + StyleTitle.Render("Removable drives (j/k select, Enter mount/unmount)") + "\n")
	removable := make([]mount.Drive, 0, len(drives))
	for _, drive := range drives {
		if drive.SafeToUnmount && drive.HasFilesystem {
			removable = append(removable, drive)
		}
	}
	if len(removable) == 0 {
		view.WriteString("  No removable drive - plug USB/SD/Phone.\n")
		return view.String()
	}
	items := make([]components.ListItem, 0, len(removable))
	for _, drive := range removable {
		sizeGB := float64(drive.Size) / (1024 * 1024 * 1024)
		mountStatus := "Unmounted"
		if drive.MountPoint != "" {
			mountStatus = "Mounted: " + drive.MountPoint
		}
		if drive.MountPoint != "" {
			if usage, ok := diskUsageAt(disks, drive.MountPoint); ok {
				mountStatus = fmt.Sprintf("%s %.1f/%.1f GB (%.0f%%)", drive.MountPoint,
					float64(usage.Used), float64(usage.Total), usage.Percent)
			}
		}
		items = append(items, components.ListItem{
			Title: "[REMOVABLE] " + drive.Device,
			Desc:  fmt.Sprintf("%.1f GB | %s", sizeGB, mountStatus),
		})
	}
	list.Items = items
	view.WriteString(fitLine(list.View(), 80) + "\n")
	view.WriteString("  Enter acts only on removable USB/SD storage.\n")
	return view.String()
}

func driveMountedAt(drives []mount.Drive, mountPoint string) (mount.Drive, bool) {
	for _, drive := range drives {
		if drive.MountPoint == mountPoint && mountPoint != "" {
			return drive, true
		}
	}
	return mount.Drive{}, false
}

func diskUsageAt(disks []monitor.DiskStat, mountPoint string) (monitor.DiskStat, bool) {
	for _, disk := range disks {
		if disk.Mount == mountPoint {
			return disk, true
		}
	}
	return monitor.DiskStat{}, false
}

func isProtectedMountPoint(mountPoint string) bool {
	clean := filepath.Clean(mountPoint)
	for _, protected := range []string{"/", "/boot", "/boot/efi", "/efi", "/home", "/var", "/usr", "/windows"} {
		if clean == protected || strings.HasPrefix(clean, protected+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func RenderNetTab(nets []monitor.NetStat, aps []wifi.AP, list components.ListModel, rxHistory, txHistory []float64) string {
	return RenderNetPanel(nets, aps, list, rxHistory, txHistory, 76)
}

func RenderNetPanel(nets []monitor.NetStat, aps []wifi.AP, list components.ListModel, rxHistory, txHistory []float64, width int) string {
	var view strings.Builder
	graphWidth := width - 6
	if graphWidth < 8 {
		graphWidth = 8
	}
	view.WriteString(StyleTitle.Render("Network throughput (MiB/s)") + "\n")
	view.WriteString("RX " + components.RenderGraph(rxHistory, graphWidth, 1) + "\n")
	view.WriteString("TX " + components.RenderGraph(txHistory, graphWidth, 1) + "\n")
	view.WriteString(StyleTitle.Render("Interfaces") + "\n")
	if len(nets) == 0 {
		view.WriteString("  No network interfaces found.\n")
	} else {
		for _, network := range nets {
			view.WriteString(fitLine(fmt.Sprintf("%s  IP %s  RX %d B  TX %d B",
				network.Iface, network.IP, network.RxBytes, network.TxBytes), 80) + "\n")
		}
	}
	view.WriteString("\n" + StyleTitle.Render("Wi-Fi access points (j/k select, Enter connect, f forget)") + "\n")
	if len(aps) == 0 {
		view.WriteString("  No access points found or NetworkManager unavailable.\n")
	} else {
		items := make([]components.ListItem, 0, len(aps))
		for _, ap := range aps {
			ssid := ap.SSID
			if ap.Connected {
				ssid = "* " + ssid
			}
			items = append(items, components.ListItem{
				Title: ssid,
				Desc:  fmt.Sprintf("%d%% %s %s %d MHz", ap.Signal, ap.Security, ap.BSSID, ap.Freq),
			})
		}
		list.Items = items
		view.WriteString(fitLine(list.View(), 80) + "\n")
	}
	return view.String()
}

func RenderAudioTab(
	sinks []audio.Sink,
	sources []audio.Source,
	devices []bluetooth.BTDevice,
	players []string,
	title, artist, status string,
	focus, selectedSink, selectedSource int,
	btList components.ListModel,
) string {
	return RenderAudioPanel(sinks, sources, focus, selectedSink, selectedSource) + "\n" +
		RenderBluetoothMediaPanel(devices, players, title, artist, status, focus == 2, btList)
}

func RenderAudioPanel(sinks []audio.Sink, sources []audio.Source, focus, selectedSink, selectedSource int) string {
	var view strings.Builder
	view.WriteString(StyleTitle.Render("Audio outputs  (o focus, j/k select, +/- volume, m mute)") + "\n")
	if len(sinks) == 0 {
		view.WriteString("  No audio sinks found (is PulseAudio/PipeWire running?).\n")
	} else {
		for index, sink := range sinks {
			marker := "  "
			if index == selectedSink {
				marker = "> "
			}
			active := " "
			if sink.InUse {
				active = "*"
			}
			mute := ""
			if sink.Muted {
				mute = " MUTED"
			}
			line := fmt.Sprintf("%s%s %s  Vol %s%s", marker, active, sink.Name,
				components.RenderSlider(sink.Volume, 100, ""), mute)
			if focus == 0 && index == selectedSink {
				line = components.StyleTitle.Render(line)
			}
			view.WriteString(fitLine(line, 80) + "\n")
		}
	}

	view.WriteString("\n" + StyleTitle.Render("Microphone sources  (i focus, Enter set default, M mute)") + "\n")
	if len(sources) == 0 {
		view.WriteString("  No microphone sources found.\n")
	} else {
		for index, source := range sources {
			marker := "  "
			if index == selectedSource {
				marker = "> "
			}
			active := " "
			if source.InUse {
				active = "*"
			}
			line := marker + active + " " + source.Desc + " - " + source.Name
			if focus == 1 && index == selectedSource {
				line = components.StyleTitle.Render(line)
			}
			view.WriteString(fitLine(line, 80) + "\n")
		}
	}

	return view.String()
}

func RenderBluetoothMediaPanel(devices []bluetooth.BTDevice, players []string, title, artist, status string, focused bool, btList components.ListModel) string {
	var view strings.Builder
	view.WriteString(StyleTitle.Render("Bluetooth (r scan, Enter pair/connect)") + "\n")
	if len(devices) == 0 {
		view.WriteString("  No discovered Bluetooth devices.\n")
	} else {
		items := make([]components.ListItem, 0, len(devices))
		for _, device := range devices {
			flags := ""
			if device.Paired {
				flags += " Paired"
			}
			if device.Connected {
				flags += " Connected"
			}
			items = append(items, components.ListItem{
				Title: device.Name,
				Desc:  fmt.Sprintf("%s RSSI %d dBm%s", device.Address, device.RSSI, flags),
			})
		}
		btList.Items = items
		if focused {
			view.WriteString(fitLine(btList.View(), 80) + "\n")
		} else {
			btList.Selected = -1
			view.WriteString(fitLine(btList.View(), 80) + "\n")
		}
	}

	view.WriteString("\n" + StyleTitle.Render("Media (p play/pause, n next, b previous)") + "\n")
	if len(players) == 0 {
		view.WriteString("  No MPRIS players found.\n")
	} else {
		view.WriteString(fitLine("  "+players[0], 80) + "\n")
		if title == "" && artist == "" {
			view.WriteString("  Nothing playing\n")
		} else {
			view.WriteString(fitLine(fmt.Sprintf("  %s - %s [%s]", title, artist, status), 80) + "\n")
		}
	}
	return view.String()
}

func fitLine(text string, width int) string {
	if lipgloss.Width(text) <= width {
		return text
	}
	var result strings.Builder
	for _, character := range text {
		candidate := result.String() + string(character) + "…"
		if lipgloss.Width(candidate) > width {
			break
		}
		result.WriteRune(character)
	}
	return result.String() + "…"
}
