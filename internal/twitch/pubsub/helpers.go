package pubsub

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"TwitchChannelPointsMiner/internal/streamer"
)

func newEmptyStream() *streamer.Stream {
	return streamer.NewStream()
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	for i := range buf {
		nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		buf[i] = charset[nBig.Int64()]
	}
	return string(buf)
}

func randomInt(min, max int) int {
	if max <= min {
		return min
	}
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return min + int(nBig.Int64())
}

func fromFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func stringOrDefault(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func formatNumber(value int) string {
	sign := ""
	v := value
	if v < 0 {
		sign = "-"
		v = -v
	}
	switch {
	case v >= 1_000_000:
		return sign + trimZeros(fmt.Sprintf("%.2fM", float64(v)/1_000_000))
	case v >= 1_000:
		return sign + trimZeros(fmt.Sprintf("%.2fk", float64(v)/1_000))
	default:
		return fmt.Sprintf("%s%d", sign, v)
	}
}

func formatFloat(val float64) string {
	return trimZeros(fmt.Sprintf("%.2f", val))
}

func trimZeros(val string) string {
	val = strings.TrimRight(strings.TrimRight(val, "0"), ".")
	if val == "" || val == "-" {
		return "0"
	}
	return val
}

func navigate(data interface{}, path string) interface{} {
	if data == nil {
		return nil
	}
	current := data
	parts := strings.Split(path, ".")
	for _, p := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[p]
		if !ok {
			return nil
		}
	}
	return current
}
