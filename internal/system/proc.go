package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type NetDevStats struct {
	RX uint64
	TX uint64
}

func ReadProcNetDev() (map[string]NetDevStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := make(map[string]NetDevStats)
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		separator := strings.LastIndexByte(line, ':')
		if separator < 0 {
			continue
		}

		iface := strings.TrimSpace(line[:separator])
		fields := strings.Fields(line[separator+1:])
		if len(fields) < 9 {
			return nil, fmt.Errorf("/proc/net/dev line %d: expected at least 9 counters", lineNumber)
		}

		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("/proc/net/dev line %d RX bytes: %w", lineNumber, err)
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("/proc/net/dev line %d TX bytes: %w", lineNumber, err)
		}
		stats[iface] = NetDevStats{RX: rx, TX: tx}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func ReadMemInfo() (map[string]uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			return nil, fmt.Errorf("/proc/meminfo line %d: expected a key and value", lineNumber)
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("/proc/meminfo line %d (%s): %w", lineNumber, key, err)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}
