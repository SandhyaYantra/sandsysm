package monitor

import (
	"context"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

type MediaStat struct {
	Player string
	Title  string
	Artist string
	Album  string
	Status string
}

func StartMediaMonitor(ctx context.Context) <-chan MediaStat {
	ch := make(chan MediaStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		var conn *dbus.Conn
		defer func() {
			if conn != nil {
				_ = conn.Close()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if conn == nil {
					var err error
					conn, err = dbus.ConnectSessionBus()
					if err != nil {
						conn = nil
						sendLatestMedia(ctx, ch, MediaStat{})
						continue
					}
				}

				stat := readNowPlaying(ctx, conn)
				if stat.Player == "" {
					if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Err; err != nil {
						_ = conn.Close()
						conn = nil
					}
				}
				sendLatestMedia(ctx, ch, stat)
			}
		}
	}()
	return ch
}

func readNowPlaying(ctx context.Context, conn *dbus.Conn) MediaStat {
	var names []string
	if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return MediaStat{}
	}

	for _, name := range names {
		if !strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			continue
		}
		object := conn.Object(name, dbus.ObjectPath("/org/mpris/MediaPlayer2"))
		statusVariant, err := object.GetProperty("org.mpris.MediaPlayer2.Player.PlaybackStatus")
		if err != nil {
			continue
		}
		metadataVariant, err := object.GetProperty("org.mpris.MediaPlayer2.Player.Metadata")
		if err != nil {
			continue
		}

		metadata, ok := metadataVariant.Value().(map[string]dbus.Variant)
		if !ok {
			continue
		}
		stat := MediaStat{Player: name}
		stat.Status, _ = statusVariant.Value().(string)
		stat.Title, _ = metadataString(metadata, "xesam:title")
		stat.Album, _ = metadataString(metadata, "xesam:album")
		stat.Artist = metadataArtists(metadata["xesam:artist"])
		return stat
	}
	return MediaStat{}
}

func metadataString(metadata map[string]dbus.Variant, key string) (string, bool) {
	variant, ok := metadata[key]
	if !ok {
		return "", false
	}
	value, ok := variant.Value().(string)
	return value, ok
}

func metadataArtists(variant dbus.Variant) string {
	value := variant.Value()
	switch artists := value.(type) {
	case []string:
		return strings.Join(artists, ", ")
	case []interface{}:
		parts := make([]string, 0, len(artists))
		for _, artist := range artists {
			if text, ok := artist.(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func sendLatestMedia(ctx context.Context, ch chan MediaStat, stat MediaStat) {
	select {
	case ch <- stat:
	case <-ctx.Done():
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- stat:
		case <-ctx.Done():
		}
	}
}
