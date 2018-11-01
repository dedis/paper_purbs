package purbs

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"log"
	"strconv"
	"strings"
	"syscall"
)

func (purb *Purb) VisualRepresentation(withBoundaries bool) string {

	lines := make([]string, 0)

	verbose := purb.isVerbose
	purb.isVerbose = false
	bytes := purb.ToBytes() // we don't want this to be verbose
	purb.isVerbose = verbose

	lines = append(lines, "*** PURB Details ***")
	lines = append(lines, fmt.Sprintf("Original Data: len %v", len(purb.originalData)))
	lines = append(lines, fmt.Sprintf("PURB: header at 0 (len %v), payload at %v (len %v), total %v bytes", purb.Header.Length, purb.Header.Length, len(purb.Payload), len(bytes)))

	lines = append(lines, fmt.Sprintf("Nonce: %+v (len %v)", purb.Nonce, len(purb.Nonce)))

	for _, cornerstone := range purb.Header.Cornerstones {
		lines = append(lines, fmt.Sprintf("Cornerstones: %+v @ offset %v (len %v)", cornerstone.SuiteName, cornerstone.Offset, purb.infoMap[cornerstone.SuiteName].CornerstoneLength))

		lines = append(lines, fmt.Sprintf("  Value: %v", cornerstone.Bytes))
		lines = append(lines, fmt.Sprintf("  Allowed positions for this suite: %v", purb.infoMap[cornerstone.SuiteName].AllowedPositions))

		cornerstoneStartPosUsed := make([]int, 0)
		for _, startPos := range purb.infoMap[cornerstone.SuiteName].AllowedPositions {
			if startPos < len(bytes) {
				cornerstoneStartPosUsed = append(cornerstoneStartPosUsed, startPos)
			}
		}

		cornerstoneRangesUsed := make([]string, 0)
		cornerstoneRangesValues := make([][]byte, 0)
		for _, startPos := range cornerstoneStartPosUsed {
			endPos := startPos + purb.infoMap[cornerstone.SuiteName].CornerstoneLength
			if endPos > len(bytes) {
				endPos = len(bytes)
			}
			cornerstoneRangesUsed = append(cornerstoneRangesUsed, strconv.Itoa(startPos)+":"+strconv.Itoa(endPos))
			cornerstoneRangesValues = append(cornerstoneRangesValues, bytes[startPos:endPos])
		}
		lines = append(lines, fmt.Sprintf("  Positions used: %v", cornerstoneRangesUsed))

		xor := make([]byte, len(cornerstoneRangesValues[0]))
		// XORed, those values should give back the cornerstone value
		for i := range cornerstoneRangesValues {
			lines = append(lines, fmt.Sprintf("  Value @ pos[%v]: %v", cornerstoneRangesUsed[i], cornerstoneRangesValues[i]))

			for i, b := range cornerstoneRangesValues[i] {
				xor[i] ^= b
			}
		}

		lines = append(lines, fmt.Sprintf("  Recomputed value: %v", xor))

	}
	for suiteName, entrypoints := range purb.Header.EntryPoints {
		lines = append(lines, fmt.Sprintf("Entrypoints for suite %v", suiteName))
		for index, entrypoint := range entrypoints {
			lines = append(lines, fmt.Sprintf("  Entrypoints [%v]: %+v @ offset %v (len %v)", index, entrypoint.SharedSecret, entrypoint.Offset, purb.Header.EntryPointLength))
		}
	}
	lines = append(lines, fmt.Sprintf("Padded Payload: %+v @ offset %v (len %v)", purb.Payload, purb.Header.Length, len(purb.Payload)))

	if !withBoundaries {
		return strings.Join(lines, "\n")
	}

	// just cosmetics
	max := 0
	for _, line := range lines {
		if len(line) > max {
			max = len(line)
		}
	}

	for i := range lines {
		lines[i] = "| " + lines[i] + strings.Repeat(" ", max-len(lines[i])) + " |"
	}

	top := strings.Repeat("_", max+4) + "\n"
	body := strings.Join(lines, "\n") + " \n"
	bottom := strings.Repeat("-", max+4) + "\n"

	return "\n" + top + body + bottom
}

// KDF derives a key from shared bytes
func KDF(password []byte) []byte {
	return pbkdf2.Key(password, nil, 1, CORNERSTONE_LENGTH, sha256.New)
}

// Helpers for measurement of CPU cost of operations
type Monitor struct {
	CPUtime float64
}

func newMonitor() *Monitor {
	var m Monitor
	m.CPUtime = getCPUTime()
	return &m
}

func (m *Monitor) reset() {
	m.CPUtime = getCPUTime()
}

func (m *Monitor) record() float64 {
	return getCPUTime() - m.CPUtime
}

func (m *Monitor) recordAndReset() float64 {
	old := m.CPUtime
	m.CPUtime = getCPUTime()
	return m.CPUtime - old
}

// Returns the sum of the system and the user CPU time used by the current process so far.
func getCPUTime() float64 {
	rusage := &syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, rusage); err != nil {
		log.Fatalln("Couldn't get rusage time:", err)
		return -1
	}
	s, u := rusage.Stime, rusage.Utime // system and user time
	return iiToF(int64(s.Sec), int64(s.Usec)) + iiToF(int64(u.Sec), int64(u.Usec))
}

// Converts to milliseconds
func iiToF(sec int64, usec int64) float64 {
	return float64(sec)*1000.0 + float64(usec)/1000.0
}