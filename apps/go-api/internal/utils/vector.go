package utils

import (
	"strconv"
	"strings"
)

// VectorToString converts a float32 slice to PostgreSQL pgvector string format: [0.012,-0.045,...]
func VectorToString(vec []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}
