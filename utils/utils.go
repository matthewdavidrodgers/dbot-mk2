package utils

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

// Check panics if error is not nil
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func contains(col []string, s string) bool {
	for _, item := range col {
		if item == s {
			return true
		}
	}
	return false
}

// MalformedParseError is an error indicating a general failure to parse a string
type MalformedParseError struct{}

func (e *MalformedParseError) Error() string {
	return "PARSE ERROR: malformed argument string"
}

// InvalidFlagError is an error indicating an illegal flag was passed
type InvalidFlagError struct{ Found string }

func (e *InvalidFlagError) Error() string {
	return fmt.Sprintf("PARSE ERROR: unrecognized flag \"%s\" provided", e.Found)
}

// ParseArgString parses a string into a map
// e.g. -foo=val is parsed into m["foo"] = "val"
func ParseArgString(argString string, argFlags []string, allowUnnamed bool) (map[string]string, error) {
	args := make(map[string]string, len(argFlags))

	acceptingUnnamed := allowUnnamed

	idx := 0
	for idx < len(argString) {
		charByte := argString[idx]
		char := string(charByte)
		if char == " " {
			idx++
			continue
		}
		if char == "-" {
			acceptingUnnamed = false

			flagByteStart := idx + 1
			flagByteEnd := flagByteStart
			innerIdx := flagByteStart
			for flagByteStart == flagByteEnd {
				if innerIdx == len(argString) {
					return nil, &MalformedParseError{}
				}
				if string(argString[innerIdx]) == "=" {
					flagByteEnd = innerIdx
					break
				}

				innerIdx++
			}

			flag := argString[flagByteStart:flagByteEnd]
			if !contains(argFlags, flag) {
				return nil, &InvalidFlagError{Found: flag}
			}
			idx = innerIdx

			flagValueByteStart := idx + 1
			flagValueByteEnd := flagValueByteStart
			innerIdx = flagValueByteStart
			for flagValueByteStart == flagValueByteEnd {
				if innerIdx == len(argString) || string(argString[innerIdx]) == " " {
					flagValueByteEnd = innerIdx
					break
				}

				innerIdx++
			}

			flagValue := argString[flagValueByteStart:flagValueByteEnd]
			args[flag] = flagValue
			idx = innerIdx + 1

			continue
		}
		if acceptingUnnamed {
			flag := "_unnamed"

			flagValueByteStart := idx
			flagValueByteEnd := flagValueByteStart
			innerIdx := flagValueByteStart
			for flagValueByteStart == flagValueByteEnd {
				if innerIdx == len(argString) || string(argString[innerIdx]) == " " {
					flagValueByteEnd = innerIdx
					break
				}

				innerIdx++
			}

			flagValue := argString[flagValueByteStart:flagValueByteEnd]
			args[flag] = flagValue
			idx = innerIdx + 1

			continue
		}
		return nil, &MalformedParseError{}
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

// ReplaceNamedValueInTextFile replaces a value for a key in a file simple key=value file and writes it back to disk
func ReplaceNamedValueInTextFile(filename string, key string, value string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("ERR", err)
		return err
	}
	re := regexp.MustCompile(`(^|\n)` + key + `=.*`)
	edited := re.ReplaceAll(contents, []byte("${1}"+key+"="+value))

	return ioutil.WriteFile(filename, edited, 0666)
}

// GetNamedValueInTextFile gets the value for a key in a simple key=falue file
func GetNamedValueInTextFile(filename string, key string) (string, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("ERR", err)
		return "", err
	}
	re := regexp.MustCompile(`(?:^|\n)` + key + `=(.*)`)
	matches := re.FindStringSubmatch(string(contents))
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], nil
}
