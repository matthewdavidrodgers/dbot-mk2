package utils

import (
	"errors"
	"strings"
)

// Check panics if error is not nil
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// ParseArgString parses a string into a map
// e.g. -f=val is parsed into m["f"] = "val"
func ParseArgString(argString string, argFlags []string) (map[string]string, error) {
	args := make(map[string]string, len(argFlags))
	e := errors.New("ERROR: malformed argString")

	idx := 0
	for idx < len(argString) {
		charByte := argString[idx]
		char := string(charByte)
		if char == " " {
			idx++
			continue
		}
		if char == "-" {
			if idx+1 == len(argString) {
				return nil, e
			}
			flagByte := argString[idx+1]
			if !((flagByte >= 65 && flagByte <= 90) || (flagByte >= 97 && flagByte <= 122)) {
				return nil, e
			}
			flag := string(flagByte)

			if idx+2 == len(argString) || string(argString[idx+2]) != "=" {
				return nil, e
			}

			innerIdx := idx + 3
			flagValue := make([]byte, 0)
			for innerIdx < len(argString) {
				innerCharByte := argString[innerIdx]
				innerChar := string(innerCharByte)

				if innerChar == " " {
					break
				}
				flagValue = append(flagValue, innerCharByte)
				innerIdx++
			}
			if len(flagValue) == 0 {
				return nil, e
			}
			args[flag] = string(flagValue)
			idx = innerIdx + 1
			continue
		}
		return nil, e
	}
	return args, nil
}

// ReadLastLines gets a readable string from the last "l" lines, offset by "o"
func ReadLastLines(bytes []byte, l int, o int) string {
	lines := strings.Split(string(bytes), "\n")
	firstLineIndex := len(lines) - l - o
	lastLineIndex := len(lines) - o
	if firstLineIndex < 0 {
		firstLineIndex = 0
	}
	return strings.Join(lines[firstLineIndex:lastLineIndex], "\n")
}
