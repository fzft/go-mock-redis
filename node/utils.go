package node

import "strings"

func mapChars(s, from, to string) string {
	for i := 0; i < len(from); i++ {
		s = strings.ReplaceAll(s, string(from[i]), string(to[i]))
	}
	return s
}
