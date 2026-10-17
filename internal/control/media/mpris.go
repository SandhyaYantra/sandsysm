package media

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	playerPrefix = "org.mpris.MediaPlayer2."
	playerPath   = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	playerIface  = "org.mpris.MediaPlayer2.Player"
)

func ListPlayers() ([]string, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return nil, fmt.Errorf("list session bus names: %w", err)
	}
	players := make([]string, 0)
	for _, name := range names {
		if strings.HasPrefix(name, playerPrefix) {
			players = append(players, name)
		}
	}
	return players, nil
}

func PlayPause(player string) error {
	return callPlayer(player, "PlayPause")
}

func Next(player string) error {
	return callPlayer(player, "Next")
}

func Previous(player string) error {
	return callPlayer(player, "Previous")
}

func GetNowPlaying(player string) (title, artist string, err error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return "", "", err
	}
	defer conn.Close()
	metadata, err := conn.Object(player, playerPath).GetProperty(playerIface + ".Metadata")
	if err != nil {
		return "", "", fmt.Errorf("get MPRIS metadata: %w", err)
	}
	properties, ok := metadata.Value().(map[string]dbus.Variant)
	if !ok {
		return "", "", fmt.Errorf("unexpected MPRIS metadata type %T", metadata.Value())
	}
	if value, ok := properties["xesam:title"].Value().(string); ok {
		title = value
	}
	artist = mprisArtist(properties["xesam:artist"])
	return title, artist, nil
}

func callPlayer(player, method string) error {
	if !strings.HasPrefix(player, playerPrefix) {
		return fmt.Errorf("invalid MPRIS player name %q", player)
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.Object(player, playerPath).Call(playerIface+"."+method, 0).Err; err != nil {
		return fmt.Errorf("MPRIS %s: %w", method, err)
	}
	return nil
}

func mprisArtist(variant dbus.Variant) string {
	switch artists := variant.Value().(type) {
	case []string:
		return strings.Join(artists, ", ")
	case []interface{}:
		values := make([]string, 0, len(artists))
		for _, value := range artists {
			if artist, ok := value.(string); ok {
				values = append(values, artist)
			}
		}
		return strings.Join(values, ", ")
	default:
		return ""
	}
}
