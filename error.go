package main

import "strings"

type MultiError []error

func (m MultiError) Error() string {
	var b strings.Builder
	b.WriteString("multiple errors:")
	for _, err := range m {
		b.WriteString("\n- " + err.Error())
	}
	return b.String()
}
