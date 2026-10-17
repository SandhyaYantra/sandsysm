package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/username/sandsysm/internal/config"
	"github.com/username/sandsysm/internal/control"
	"github.com/username/sandsysm/internal/control/audio"
	"github.com/username/sandsysm/internal/control/bluetooth"
	"github.com/username/sandsysm/internal/control/media"
	"github.com/username/sandsysm/internal/control/mount"
	"github.com/username/sandsysm/internal/control/wifi"
	"github.com/username/sandsysm/internal/monitor"
	"github.com/username/sandsysm/internal/ui/components"
)

type ramMsg monitor.RamStat
type diskMsg []monitor.DiskStat
type netMsg []monitor.NetStat
type cpuMsg monitor.CpuStat
type gpuMsg monitor.GpuStat
type batteryMsg monitor.BatteryStat
type mediaMonitorMsg monitor.MediaStat

type controlTickMsg time.Time

type drivesMsg struct {
	items []mount.Drive
	err   error
}

type wifiMsg struct {
	items []wifi.AP
	err   error
}

type bluetoothMsg struct {
	items []bluetooth.BTDevice
	err   error
}

type audioMsg struct {
	sinks   []audio.Sink
	sources []audio.Source
	err     error
}

type mediaMsg struct {
	players []string
	title   string
	artist  string
	status  string
	err     error
}

type actionResultMsg struct {
	text    string
	refresh bool
}

type systemControlsMsg struct {
	brightness, brightnessMax  int
	keyboard, keyboardMax      int
	airplane                   bool
	profiles                   []control.PowerProfile
	activeProfile              control.PowerProfile
	brightnessErr, keyboardErr error
	airplaneErr, profileErr    error
}

type PanelVisible struct {
	Monitor bool
	Disk    bool
	Net     bool
	Audio   bool
	Battery bool
}

type Model struct {
	cfg                config.Config
	ram                monitor.RamStat
	disks              []monitor.DiskStat
	nets               []monitor.NetStat
	cpu                monitor.CpuStat
	gpu                monitor.GpuStat
	battery            monitor.BatteryStat
	panels             PanelVisible
	focusedPanel       int
	ctx                context.Context
	cancel             context.CancelFunc
	ramCh              <-chan monitor.RamStat
	diskCh             <-chan []monitor.DiskStat
	netCh              <-chan []monitor.NetStat
	cpuCh              <-chan monitor.CpuStat
	gpuCh              <-chan monitor.GpuStat
	batteryCh          <-chan monitor.BatteryStat
	mediaCh            <-chan monitor.MediaStat
	drives             []mount.Drive
	drivesList         components.ListModel
	aps                []wifi.AP
	wifiList           components.ListModel
	btDevices          []bluetooth.BTDevice
	btList             components.ListModel
	profileList        components.ListModel
	sinks              []audio.Sink
	sources            []audio.Source
	selectedSink       int
	selectedSource     int
	players            []string
	nowPlayingTitle    string
	nowPlayingArtist   string
	nowPlayingStatus   string
	mediaMon           monitor.MediaStat
	netRxHistory       []float64
	netTxHistory       []float64
	controlTick        int
	audioFocus         int
	status             string
	wifiPasswordPrompt bool
	wifiPassword       string
	promptSSID         string
	brightness         int
	brightnessMax      int
	keyboardBrightness int
	keyboardMax        int
	airplaneMode       bool
	profiles           []control.PowerProfile
	activeProfile      control.PowerProfile
	termWidth          int
	windowHeight       int
	scrollOffset       int
}

func NewApp() *Model {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{
		cfg:       cfg,
		panels:    PanelVisible{Monitor: true, Disk: true, Net: true, Audio: true, Battery: true},
		ctx:       ctx,
		cancel:    cancel,
		ramCh:     monitor.StartRam(ctx),
		diskCh:    monitor.StartDisk(ctx),
		netCh:     monitor.StartNet(ctx),
		cpuCh:     monitor.StartCpu(ctx),
		gpuCh:     monitor.StartGpu(ctx),
		batteryCh: monitor.StartBattery(ctx),
		mediaCh:   monitor.StartMediaMonitor(ctx),
		battery:   monitor.BatteryStat{Capacity: -1, Status: "Unavailable"},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitRam(m.ramCh),
		waitDisk(m.diskCh),
		waitNet(m.netCh),
		waitCpu(m.cpuCh),
		waitGpu(m.gpuCh),
		waitBattery(m.batteryCh),
		waitMediaMonitor(m.mediaCh),
		fetchDrivesCmd(),
		fetchWifiCmd(),
		fetchBluetoothCmd(),
		fetchAudioCmd(),
		fetchMediaCmd(),
		fetchSystemControlsCmd(),
		startBluetoothDiscoveryCmd(),
		controlTick(),
	)
}

func waitRam(ch <-chan monitor.RamStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return ramMsg(value)
	}
}

func waitDisk(ch <-chan []monitor.DiskStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return diskMsg(value)
	}
}

func waitNet(ch <-chan []monitor.NetStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return netMsg(value)
	}
}

func waitCpu(ch <-chan monitor.CpuStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return cpuMsg(value)
	}
}

func waitGpu(ch <-chan monitor.GpuStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return gpuMsg(value)
	}
}

func waitBattery(ch <-chan monitor.BatteryStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return batteryMsg(value)
	}
}

func waitMediaMonitor(ch <-chan monitor.MediaStat) tea.Cmd {
	return func() tea.Msg {
		value, ok := <-ch
		if !ok {
			return nil
		}
		return mediaMonitorMsg(value)
	}
}

func controlTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return controlTickMsg(t) })
}

func fetchDrives() tea.Msg {
	items, err := mount.ListDrives()
	return drivesMsg{items: items, err: err}
}

func fetchWifi() tea.Msg {
	items, err := wifi.Scan()
	return wifiMsg{items: items, err: err}
}

func fetchBT() tea.Msg {
	items, err := bluetooth.ListDevices()
	return bluetoothMsg{items: items, err: err}
}

func fetchAudio() tea.Msg {
	sinks, sinkErr := audio.GetSinks()
	sources, sourceErr := audio.GetSources()
	return audioMsg{sinks: sinks, sources: sources, err: errorsJoin(sinkErr, sourceErr)}
}

func fetchMedia() tea.Msg {
	players, err := media.ListPlayers()
	if err != nil || len(players) == 0 {
		return mediaMsg{players: players, err: err}
	}
	title, artist, err := media.GetNowPlaying(players[0])
	return mediaMsg{players: players, title: title, artist: artist, err: err}
}

func fetchSystemControls() tea.Msg {
	var result systemControlsMsg
	result.brightness, result.brightnessMax, result.brightnessErr = control.GetBrightness()
	result.keyboard, result.keyboardMax, result.keyboardErr = control.GetKbdBrightness()
	result.airplane, result.airplaneErr = control.GetAirplaneMode()
	result.profiles, result.profileErr = control.ListProfiles()
	result.activeProfile, _ = control.GetActiveProfile()
	return result
}

func errorsJoin(first, second error) error {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return fmt.Errorf("%v; %w", first, second)
}

func fetchDrivesCmd() tea.Cmd {
	return func() tea.Msg { return fetchDrives() }
}

func fetchWifiCmd() tea.Cmd {
	return func() tea.Msg { return fetchWifi() }
}

func fetchBluetoothCmd() tea.Cmd {
	return func() tea.Msg { return fetchBT() }
}

func fetchAudioCmd() tea.Cmd {
	return func() tea.Msg { return fetchAudio() }
}

func fetchMediaCmd() tea.Cmd {
	return func() tea.Msg { return fetchMedia() }
}

func fetchSystemControlsCmd() tea.Cmd {
	return func() tea.Msg { return fetchSystemControls() }
}

func startBluetoothDiscoveryCmd() tea.Cmd {
	return func() tea.Msg {
		if err := bluetooth.StartDiscovery(); err != nil {
			return actionResultMsg{text: "Bluetooth discovery: " + err.Error()}
		}
		return actionResultMsg{text: "Bluetooth discovery started"}
	}
}

func actionCmd(success string, action func() error) tea.Cmd {
	return func() tea.Msg {
		if err := action(); err != nil {
			return actionResultMsg{text: success + " failed: " + err.Error(), refresh: true}
		}
		return actionResultMsg{text: success, refresh: true}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.wifiPasswordPrompt {
			return m.updateWifiPassword(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "pgdown", "ctrl+f":
			m.scrollOffset += maxInt(m.windowHeight-2, 1)
		case "pgup":
			m.scrollOffset -= maxInt(m.windowHeight-2, 1)
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		case "home":
			m.scrollOffset = 0
		case "end":
			m.scrollOffset = int(^uint(0) >> 1)
		case "1":
			m.panels.Monitor = !m.panels.Monitor
			m.keepFocusVisible()
		case "2":
			m.panels.Disk = !m.panels.Disk
			m.keepFocusVisible()
		case "3":
			m.panels.Net = !m.panels.Net
			m.keepFocusVisible()
		case "4":
			m.panels.Audio = !m.panels.Audio
			m.keepFocusVisible()
		case "5":
			m.panels.Battery = !m.panels.Battery
			m.keepFocusVisible()
		case "l", "tab", "right":
			m.focusedPanel = m.nextVisiblePanel(1)
		case "h", "shift+tab", "left":
			m.focusedPanel = m.nextVisiblePanel(-1)
		case "j", "down":
			m.moveSelection(1)
		case "k", "up":
			m.moveSelection(-1)
		case "enter":
			return m, m.enterAction()
		case "o":
			m.audioFocus = 0
			m.focusedPanel = 3
			m.status = "Output list focused"
		case "i":
			m.audioFocus = 1
			m.focusedPanel = 3
			m.status = "Microphone list focused"
		case "t":
			m.audioFocus = 2
			m.focusedPanel = 3
			m.status = "Bluetooth list focused"
		case "r":
			if m.focusedPanel == 3 && m.audioFocus == 2 {
				return m, startBluetoothDiscoveryCmd()
			}
		case "C":
			if m.focusedPanel == 2 && m.panels.Net {
				m.status = "Scanning Wi-Fi access points..."
				return m, fetchWifiCmd()
			}
		case "m":
			if m.focusedPanel == 3 && m.audioFocus == 1 {
				return m, actionCmd("Microphone mute toggled", audio.ToggleMicMute)
			}
			return m, actionCmd("Output mute toggled", audio.ToggleMute)
		case "M":
			return m, actionCmd("Microphone mute toggled", audio.ToggleMicMute)
		case "+", "=":
			if m.focusedPanel == 3 {
				return m, m.adjustSelectedSink(5)
			}
			return m, m.adjustBrightness(5)
		case "-":
			if m.focusedPanel == 3 {
				return m, m.adjustSelectedSink(-5)
			}
			return m, m.adjustBrightness(-5)
		case "B":
			return m, m.adjustBrightness(5)
		case "ctrl+b":
			return m, m.adjustBrightness(-5)
		case "K":
			return m, m.adjustKeyboardBrightness(1)
		case "ctrl+k":
			return m, m.adjustKeyboardBrightness(-1)
		case "A":
			return m, actionCmd("Airplane mode toggled", func() error {
				return control.SetAirplaneMode(!m.airplaneMode)
			})
		case "P":
			return m, m.cyclePowerProfile()
		case "f":
			if m.focusedPanel == 2 && m.wifiList.Selected >= 0 && m.wifiList.Selected < len(m.aps) {
				ssid := m.aps[m.wifiList.Selected].SSID
				return m, actionCmd("Forgot Wi-Fi network "+ssid, func() error { return wifi.Forget(ssid) })
			}
		case "p", "n", "b":
			if m.focusedPanel == 3 && len(m.players) > 0 {
				player := m.players[0]
				switch msg.String() {
				case "p":
					return m, actionCmd("Media toggled", func() error { return media.PlayPause(player) })
				case "n":
					return m, actionCmd("Skipped to next track", func() error { return media.Next(player) })
				case "b":
					return m, actionCmd("Returned to previous track", func() error { return media.Previous(player) })
				}
			}
		}
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.windowHeight = msg.Height
	case ramMsg:
		m.ram = monitor.RamStat(msg)
		return m, waitRam(m.ramCh)
	case diskMsg:
		m.disks = []monitor.DiskStat(msg)
		return m, waitDisk(m.diskCh)
	case netMsg:
		m.nets = []monitor.NetStat(msg)
		rx, tx := 0.0, 0.0
		for _, network := range m.nets {
			rx += network.RxSpeed
			tx += network.TxSpeed
		}
		m.netRxHistory = appendHistory(m.netRxHistory, rx)
		m.netTxHistory = appendHistory(m.netTxHistory, tx)
		return m, waitNet(m.netCh)
	case cpuMsg:
		m.cpu = monitor.CpuStat(msg)
		return m, waitCpu(m.cpuCh)
	case gpuMsg:
		m.gpu = monitor.GpuStat(msg)
		return m, waitGpu(m.gpuCh)
	case batteryMsg:
		m.battery = monitor.BatteryStat(msg)
		return m, waitBattery(m.batteryCh)
	case mediaMonitorMsg:
		m.mediaMon = monitor.MediaStat(msg)
		m.nowPlayingTitle = m.mediaMon.Title
		m.nowPlayingArtist = m.mediaMon.Artist
		m.nowPlayingStatus = m.mediaMon.Status
		return m, waitMediaMonitor(m.mediaCh)
	case drivesMsg:
		if msg.err != nil {
			m.status = "Drive refresh failed: " + msg.err.Error()
			break
		}
		m.drives = msg.items
		actionableDrives := safeRemovableDrives(m.drives)
		m.drivesList.Items = make([]components.ListItem, 0, len(actionableDrives))
		for _, drive := range actionableDrives {
			mountState := "Unmounted"
			if drive.MountPoint != "" {
				mountState = drive.MountPoint
			}
			m.drivesList.Items = append(m.drivesList.Items, components.ListItem{
				Title: drive.Device,
				Desc:  fmt.Sprintf("%.1f GB %s", float64(drive.Size)/(1024*1024*1024), mountState),
			})
		}
		m.drivesList.Selected = clampSelection(m.drivesList.Selected, len(actionableDrives))
	case wifiMsg:
		if msg.err != nil {
			m.status = "Wi-Fi refresh failed: " + msg.err.Error()
			break
		}
		m.aps = msg.items
		m.wifiList.Items = make([]components.ListItem, 0, len(m.aps))
		for _, ap := range m.aps {
			ssid := ap.SSID
			if ap.Connected {
				ssid = "* " + ssid
			}
			m.wifiList.Items = append(m.wifiList.Items, components.ListItem{
				Title: ssid,
				Desc:  fmt.Sprintf("%d%% %s %s %d MHz", ap.Signal, ap.Security, ap.BSSID, ap.Freq),
			})
		}
		m.wifiList.Selected = clampSelection(m.wifiList.Selected, len(m.aps))
	case bluetoothMsg:
		if msg.err != nil {
			m.status = "Bluetooth refresh failed: " + msg.err.Error()
			break
		}
		m.btDevices = msg.items
		m.btList.Items = make([]components.ListItem, 0, len(m.btDevices))
		for _, device := range m.btDevices {
			state := ""
			if device.Paired {
				state += " Paired"
			}
			if device.Connected {
				state += " Connected"
			}
			m.btList.Items = append(m.btList.Items, components.ListItem{
				Title: device.Name,
				Desc:  fmt.Sprintf("%s RSSI %d dBm%s", device.Address, device.RSSI, state),
			})
		}
		m.btList.Selected = clampSelection(m.btList.Selected, len(m.btDevices))
	case audioMsg:
		m.sinks = msg.sinks
		m.sources = msg.sources
		m.selectedSink = clampSelection(m.selectedSink, len(m.sinks))
		m.selectedSource = clampSelection(m.selectedSource, len(m.sources))
		if msg.err != nil {
			m.status = "Audio refresh: " + msg.err.Error()
		}
	case mediaMsg:
		m.players = msg.players
		if len(msg.players) == 0 {
			m.nowPlayingTitle = ""
			m.nowPlayingArtist = ""
			m.nowPlayingStatus = ""
		} else if msg.title != "" {
			m.nowPlayingTitle = msg.title
			m.nowPlayingArtist = msg.artist
		}
		if msg.err != nil {
			m.status = "Media refresh: " + msg.err.Error()
		}
	case actionResultMsg:
		m.status = msg.text
		if msg.refresh {
			return m, tea.Batch(fetchDrivesCmd(), fetchBluetoothCmd(), fetchAudioCmd(), fetchMediaCmd(), fetchSystemControlsCmd())
		}
	case systemControlsMsg:
		m.brightness, m.brightnessMax = msg.brightness, msg.brightnessMax
		m.keyboardBrightness, m.keyboardMax = msg.keyboard, msg.keyboardMax
		m.airplaneMode = msg.airplane
		m.profiles, m.activeProfile = msg.profiles, msg.activeProfile
		m.profileList.Items = make([]components.ListItem, 0, len(m.profiles))
		for _, profile := range m.profiles {
			m.profileList.Items = append(m.profileList.Items, components.ListItem{Title: string(profile)})
		}
		m.profileList.Selected = clampSelection(m.profileList.Selected, len(m.profiles))
		if msg.airplaneErr != nil {
			m.status = "Airplane state unavailable: " + msg.airplaneErr.Error()
		}
	case controlTickMsg:
		m.controlTick++
		return m, tea.Batch(fetchDrivesCmd(), fetchBluetoothCmd(), fetchAudioCmd(), fetchMediaCmd(), fetchSystemControlsCmd(), controlTick())
	}
	return m, nil
}

func (m Model) updateWifiPassword(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.wifiPasswordPrompt = false
		m.wifiPassword = ""
		m.status = "Wi-Fi connection cancelled"
	case "enter":
		ssid, password := m.promptSSID, m.wifiPassword
		m.wifiPasswordPrompt = false
		m.wifiPassword = ""
		return m, actionCmd("Connected to "+ssid, func() error { return wifi.Connect(ssid, password) })
	case "backspace", "ctrl+h":
		passwordRunes := []rune(m.wifiPassword)
		if len(passwordRunes) > 0 {
			m.wifiPassword = string(passwordRunes[:len(passwordRunes)-1])
		}
	default:
		if len(key.Runes) > 0 {
			m.wifiPassword += string(key.Runes)
		}
	}
	return m, nil
}

func (m *Model) moveSelection(direction int) {
	switch m.focusedPanel {
	case 1:
		if direction < 0 {
			m.drivesList.Up()
		} else {
			m.drivesList.Down()
		}
	case 2:
		if direction < 0 {
			m.wifiList.Up()
		} else {
			m.wifiList.Down()
		}
	case 3:
		switch m.audioFocus {
		case 0:
			m.selectedSink = moveIndex(m.selectedSink, direction, len(m.sinks))
		case 1:
			m.selectedSource = moveIndex(m.selectedSource, direction, len(m.sources))
		case 2:
			if direction < 0 {
				m.btList.Up()
			} else {
				m.btList.Down()
			}
		}
	case 4:
		if direction < 0 {
			m.profileList.Up()
		} else {
			m.profileList.Down()
		}
	}
}

func (m *Model) keepFocusVisible() {
	if !m.panelIsVisible(m.focusedPanel) {
		m.focusedPanel = m.nextVisiblePanel(1)
	}
}

func (m *Model) nextVisiblePanel(direction int) int {
	for step := 1; step <= 5; step++ {
		index := (m.focusedPanel + direction*step + 10) % 5
		if m.panelIsVisible(index) {
			return index
		}
	}
	return m.focusedPanel
}

func (m *Model) panelIsVisible(index int) bool {
	switch index {
	case 0:
		return m.panels.Monitor
	case 1:
		return m.panels.Disk
	case 2:
		return m.panels.Net
	case 3:
		return m.panels.Audio
	case 4:
		return m.panels.Battery
	default:
		return false
	}
}

func moveIndex(index, direction, length int) int {
	if length == 0 {
		return 0
	}
	index += direction
	return clampSelection(index, length)
}

func clampSelection(index, length int) int {
	if length == 0 || index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func safeRemovableDrives(drives []mount.Drive) []mount.Drive {
	result := make([]mount.Drive, 0, len(drives))
	for _, drive := range drives {
		if drive.SafeToUnmount && drive.HasFilesystem {
			result = append(result, drive)
		}
	}
	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func appendHistory(history []float64, value float64) []float64 {
	if len(history) < 60 {
		return append(history, value)
	}
	copy(history, history[1:])
	history[len(history)-1] = value
	return history
}

func (m *Model) enterAction() tea.Cmd {
	switch m.focusedPanel {
	case 1:
		actionableDrives := safeRemovableDrives(m.drives)
		index := m.drivesList.Selected
		if index < 0 || index >= len(actionableDrives) {
			if len(actionableDrives) == 0 && len(m.drives) > 0 {
				var protectedDrive *mount.Drive
				for _, drive := range m.drives {
					if !drive.SafeToUnmount && (protectedDrive == nil || drive.IsSystemPartition) {
						candidate := drive
						protectedDrive = &candidate
						if drive.IsSystemPartition {
							break
						}
					}
				}
				if protectedDrive != nil {
					mountPoint := protectedDrive.MountPoint
					if mountPoint == "" {
						mountPoint = protectedDrive.Device
					}
					if protectedDrive.IsSystemPartition {
						m.status = "Cannot unmount system partition " + mountPoint + " (safety lock)"
					} else {
						m.status = "Cannot mount fixed drive " + mountPoint + " (safety lock)"
					}
				}
			}
			return nil
		}
		drive := actionableDrives[index]
		if !drive.SafeToUnmount || !drive.HasFilesystem {
			mountPoint := drive.MountPoint
			if mountPoint == "" {
				mountPoint = drive.Device
			}
			m.status = "Cannot unmount system partition " + mountPoint + " (safety lock)"
			return nil
		}
		objectPath := drive.ObjectPath
		if objectPath == "" {
			objectPath = drive.Path
		}
		if drive.MountPoint == "" {
			return actionCmd("Mounted "+drive.Device, func() error {
				_, err := mount.Mount(objectPath)
				return err
			})
		}
		return actionCmd("Unmounted "+drive.Device, func() error { return mount.Unmount(objectPath) })
	case 2:
		index := m.wifiList.Selected
		if index < 0 || index >= len(m.aps) {
			return nil
		}
		ap := m.aps[index]
		if strings.EqualFold(ap.Security, "Open") {
			return actionCmd("Connected to "+ap.SSID, func() error { return wifi.Connect(ap.SSID, "") })
		}
		m.wifiPasswordPrompt = true
		m.promptSSID = ap.SSID
		m.wifiPassword = ""
		m.status = "Enter password for " + ap.SSID
		return nil
	case 3:
		switch m.audioFocus {
		case 0:
			if m.selectedSink >= 0 && m.selectedSink < len(m.sinks) {
				sink := m.sinks[m.selectedSink]
				volume := sink.Volume + 5
				if volume > 100 {
					volume = 100
				}
				return actionCmd("Raised volume for "+sink.Name, func() error { return audio.SetVolume(sink.Name, volume) })
			}
		case 1:
			if m.selectedSource >= 0 && m.selectedSource < len(m.sources) {
				source := m.sources[m.selectedSource]
				return actionCmd("Selected microphone "+source.Desc, func() error { return audio.SetDefaultSource(source.Name) })
			}
		case 2:
			index := m.btList.Selected
			if index >= 0 && index < len(m.btDevices) {
				device := m.btDevices[index]
				return actionCmd("Connected Bluetooth device "+device.Name, func() error {
					if !device.Paired {
						if err := bluetooth.Pair(device.Address); err != nil {
							return err
						}
						if err := bluetooth.Trust(device.Address); err != nil {
							return err
						}
					}
					return bluetooth.ConnectBT(device.Address)
				})
			}
		}
	case 4:
		index := m.profileList.Selected
		if index < 0 || index >= len(m.profiles) {
			return nil
		}
		profile := m.profiles[index]
		return actionCmd("Set power profile to "+string(profile), func() error { return control.SetProfile(profile) })
	}
	return nil
}

func (m Model) adjustSelectedSink(delta int) tea.Cmd {
	if m.focusedPanel != 3 || m.audioFocus != 0 || m.selectedSink < 0 || m.selectedSink >= len(m.sinks) {
		return nil
	}
	sink := m.sinks[m.selectedSink]
	volume := sink.Volume + delta
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	return actionCmd("Set volume for "+sink.Name, func() error { return audio.SetVolume(sink.Name, volume) })
}

func (m Model) adjustBrightness(delta int) tea.Cmd {
	if m.brightnessMax <= 0 {
		return actionCmd("Brightness unavailable", func() error { return fmt.Errorf("no backlight device found") })
	}
	value := m.brightness + delta*m.brightnessMax/100
	percent := value * 100 / m.brightnessMax
	return actionCmd(fmt.Sprintf("Set display brightness to %d%%", percent), func() error { return control.SetBrightness(value) })
}

func (m Model) adjustKeyboardBrightness(delta int) tea.Cmd {
	if m.keyboardMax <= 0 {
		return actionCmd("Keyboard backlight unavailable", func() error { return fmt.Errorf("no keyboard backlight found") })
	}
	step := m.keyboardMax / 3
	if step < 1 {
		step = 1
	}
	value := m.keyboardBrightness + delta*step
	percent := value * 100 / m.keyboardMax
	return actionCmd(fmt.Sprintf("Set keyboard backlight to %d%%", percent), func() error {
		return control.SetKbdBrightness(value)
	})
}

func (m Model) cyclePowerProfile() tea.Cmd {
	if len(m.profiles) == 0 {
		return actionCmd("Power profiles unavailable", func() error { return fmt.Errorf("no supported power profiles") })
	}
	index := 0
	for i, profile := range m.profiles {
		if profile == m.activeProfile {
			index = (i + 1) % len(m.profiles)
			break
		}
	}
	profile := m.profiles[index]
	return actionCmd("Set power profile to "+string(profile), func() error { return control.SetProfile(profile) })
}

func (m Model) View() string {
	width := m.termWidth
	if width <= 0 {
		width = 80
	}
	if width < 48 {
		width = 48
	}
	fullWidth := width
	if fullWidth < 40 {
		fullWidth = 40
	}
	left60 := (fullWidth - 1) * 6 / 10
	right40 := fullWidth - left60 - 1
	leftHalf := (fullWidth - 1) / 2
	rightHalf := fullWidth - leftHalf - 1

	type panelBox struct {
		title   string
		content string
		width   int
		focused bool
	}
	type panelRow struct {
		boxes  []panelBox
		height int
	}
	rows := make([]panelRow, 0, 5)
	if m.panels.Monitor {
		rows = append(rows, panelRow{height: 12, boxes: []panelBox{{"CPU", RenderCPUPanel(m.cpu), fullWidth, m.focusedPanel == 0}}})
	}
	if m.panels.Monitor || m.panels.Battery {
		row := panelRow{height: 9}
		if m.panels.Monitor {
			row.boxes = append(row.boxes, panelBox{"Memory / GPU / Backlight",
				RenderMemoryPanel(m.ram, m.gpu, m.brightness, m.brightnessMax, m.keyboardBrightness, m.keyboardMax), left60, m.focusedPanel == 0})
		}
		if m.panels.Battery {
			boxWidth := fullWidth
			if m.panels.Monitor {
				boxWidth = right40
			}
			row.boxes = append(row.boxes, panelBox{"Battery / Power",
				RenderBatteryPanel(m.battery, m.profiles, m.activeProfile, m.profileList.Selected, m.airplaneMode), boxWidth, m.focusedPanel == 4})
		}
		rows = append(rows, row)
	}
	if m.panels.Disk {
		rows = append(rows, panelRow{height: 12, boxes: []panelBox{{"Disk / Removable", RenderDiskTab(m.disks, m.drives, m.drivesList), fullWidth, m.focusedPanel == 1}}})
	}
	if m.panels.Net {
		rows = append(rows, panelRow{height: 12, boxes: []panelBox{{"Network / Wi-Fi", RenderNetPanel(m.nets, m.aps, m.wifiList,
			m.netRxHistory, m.netTxHistory, fullWidth-4), fullWidth, m.focusedPanel == 2}}})
	}
	if m.panels.Audio {
		btContent := RenderBluetoothMediaPanel(m.btDevices, m.players, m.nowPlayingTitle, m.nowPlayingArtist, m.nowPlayingStatus, m.audioFocus == 2, m.btList)
		rows = append(rows, panelRow{height: 12, boxes: []panelBox{
			{"Audio / Sources", RenderAudioPanel(m.sinks, m.sources, m.audioFocus, m.selectedSink, m.selectedSource), leftHalf, m.focusedPanel == 3 && m.audioFocus != 2},
			{"Bluetooth / Media", btContent, rightHalf, m.focusedPanel == 3 && m.audioFocus == 2},
		}})
	}
	if len(rows) == 0 {
		rows = append(rows, panelRow{height: 5, boxes: []panelBox{{"sandsysm", "All panels are hidden. Press 1-5 to show a panel.", fullWidth, true}}})
	}
	focusNames := [...]string{"Monitor", "Disk", "Net/Wi-Fi", "Audio", "Battery/Power"}
	focusName := "Dashboard"
	if m.focusedPanel >= 0 && m.focusedPanel < len(focusNames) {
		focusName = focusNames[m.focusedPanel]
	}
	header := StyleTitle.Render(fitLine(" sandsysm | dashboard | focus: "+focusName+" ", width))
	help := "1-5 panels | h/l focus | j/k select | PgUp/PgDn scroll | C Wi-Fi scan | A airplane | q quit"
	if m.wifiPasswordPrompt {
		help = fmt.Sprintf("Password for %s: %s_  Enter connect, Esc cancel", m.promptSSID, strings.Repeat("*", len([]rune(m.wifiPassword))))
	} else {
		help += " | o/i/t audio focus | +/- volume/brightness | B/ctrl+b brightness | K/ctrl+k keyboard | M mic mute | P profile | p/n/b media"
	}
	footer := []string{fitLine(help, width)}
	if m.status != "" {
		footer = append(footer, fitLine("Status: "+m.status, width))
	}
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(strings.Join(footer, "\n"))
	separatorHeight := maxInt(len(rows)-1, 0)
	footerGap := 1
	availableHeight := m.windowHeight - headerHeight - footerHeight - separatorHeight - footerGap
	baseHeight := 0
	for _, row := range rows {
		baseHeight += row.height
	}
	if availableHeight > baseHeight {
		rows[len(rows)-1].height += availableHeight - baseHeight
	}
	renderedRows := make([]string, 0, len(rows)*2+1)
	for _, row := range rows {
		boxes := make([]string, 0, len(row.boxes))
		for _, box := range row.boxes {
			boxes = append(boxes, components.Box(box.title, box.content, box.width, row.height, box.focused))
		}
		if len(boxes) == 1 {
			renderedRows = append(renderedRows, boxes[0])
		} else {
			renderedRows = append(renderedRows, lipgloss.JoinHorizontal(lipgloss.Top, boxes[0], " ", boxes[1]))
		}
	}
	contentRows := make([]string, 0, len(renderedRows)*2+2)
	contentRows = append(contentRows, header)
	for index, row := range renderedRows {
		if index > 0 {
			contentRows = append(contentRows, "")
		}
		contentRows = append(contentRows, row)
	}
	contentRows = append(contentRows, "", strings.Join(footer, "\n"))
	content := strings.Join(contentRows, "\n")
	lines := strings.Split(content, "\n")
	height := m.windowHeight
	if height <= 0 {
		height = 24
	}
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := m.scrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	visibleLines := lines[offset:end]
	for len(visibleLines) < height {
		visibleLines = append(visibleLines, "")
	}
	return strings.Join(visibleLines, "\n")
}

func (m Model) monitorView() string {
	var view strings.Builder
	fmt.Fprintf(&view, "CPU  %.1f%%\n", m.cpu.Total)
	if m.ram.Total == 0 {
		view.WriteString("RAM  Waiting for first sample...\n")
	} else {
		percent := float64(m.ram.Used) / float64(m.ram.Total) * 100
		fmt.Fprintf(&view, "RAM  %s\n     Available %d MB | Cache %d MB | Buffers %d MB\n",
			components.RenderBar(m.ram.Used*1024*1024, m.ram.Total*1024*1024, percent),
			m.ram.Available, m.ram.Cache, m.ram.Buffers,
		)
	}
	if len(m.nets) == 0 {
		view.WriteString("NET  Waiting for interface data...\n")
	} else {
		view.WriteString("\nNetwork interfaces\n")
		for _, network := range m.nets {
			fmt.Fprintf(&view, "%s  RX %.2f MiB/s  TX %.2f MiB/s\n", network.Iface, network.RxSpeed, network.TxSpeed)
			if network.IP != "" {
				fmt.Fprintf(&view, "  IP %s\n", network.IP)
			}
		}
	}
	return view.String()
}

func (m *Model) Run() error {
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	m.cancel()
	return err
}
