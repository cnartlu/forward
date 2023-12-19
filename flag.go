package main

import (
	"bytes"
)

type StringsFlag []string

func (f *StringsFlag) String() string {
	buf := bytes.Buffer{}
	buf.WriteString("[")
	for _, s := range *f {
		buf.WriteString("\"")
		buf.WriteString(s)
		buf.WriteString("\"")
	}
	buf.WriteString("]")
	return buf.String()
}

func (f *StringsFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
