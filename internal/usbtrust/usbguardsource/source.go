package usbguardsource

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

var vidPIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{4}:[0-9a-fA-F]{4}$`,
)

type Observation struct {
	Target string
	Device usbtrust.ObservedDevice
}

// ParseLine parses both:
//
//	allow id 1234:5678 ...
//
// and runtime list-devices output:
//
//	31: block id 1234:5678 ...
//
// The USBGuard rule remains observation data only. It never becomes the
// authoritative trust database.
func ParseLine(line string) (Observation, error) {
	tokens, err := tokenize(strings.TrimSpace(line))
	if err != nil {
		return Observation{}, err
	}

	if len(tokens) < 3 {
		return Observation{}, fmt.Errorf(
			"USBGuard observation is incomplete",
		)
	}

	var runtimeID string

	if strings.HasSuffix(tokens[0], ":") {
		runtimeID = strings.TrimSuffix(tokens[0], ":")

		if _, err := strconv.ParseUint(runtimeID, 10, 64); err != nil {
			return Observation{}, fmt.Errorf(
				"invalid USBGuard runtime id %q",
				runtimeID,
			)
		}

		tokens = tokens[1:]
	}

	target := tokens[0]

	switch target {
	case "allow", "block", "reject":
	default:
		return Observation{}, fmt.Errorf(
			"unsupported USBGuard target %q",
			target,
		)
	}

	tokens = tokens[1:]

	identity := usbtrust.Identity{}

	for len(tokens) > 0 {
		key := tokens[0]
		tokens = tokens[1:]

		switch key {
		case "id":
			value, rest, err := takeValue(key, tokens)
			if err != nil {
				return Observation{}, err
			}
			tokens = rest

			if !vidPIDPattern.MatchString(value) {
				return Observation{}, fmt.Errorf(
					"invalid USB VID:PID %q",
					value,
				)
			}

			identity.VIDPID = strings.ToLower(value)

		case "serial":
			identity.Serial, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "name":
			identity.Name, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "hash":
			identity.Hash, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "parent-hash":
			identity.ParentHash, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "via-port":
			identity.Port, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "with-connect-type":
			identity.ConnectType, tokens, err = takeValue(
				key,
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		case "with-interface":
			identity.Interfaces, tokens, err = takeInterfaces(
				tokens,
			)
			if err != nil {
				return Observation{}, err
			}

		default:
			return Observation{}, fmt.Errorf(
				"unsupported USBGuard observation attribute %q",
				key,
			)
		}
	}

	if identity.VIDPID == "" {
		return Observation{}, fmt.Errorf(
			"USBGuard observation has no id",
		)
	}

	if identity.Hash == "" {
		return Observation{}, fmt.Errorf(
			"USBGuard observation %s has no descriptor hash",
			identity.VIDPID,
		)
	}

	return Observation{
		Target: target,
		Device: usbtrust.ObservedDevice{
			RuntimeID: runtimeID,
			Identity:  identity,
		},
	}, nil
}

// ParseLines parses non-empty USBGuard output lines in deterministic order.
func ParseLines(input string) ([]Observation, error) {
	var observations []Observation

	for number, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		observation, err := ParseLine(line)
		if err != nil {
			return nil, fmt.Errorf(
				"USBGuard line %d: %w",
				number+1,
				err,
			)
		}

		observations = append(observations, observation)
	}

	sort.SliceStable(observations, func(i, j int) bool {
		return observations[i].Device.Identity.VIDPID <
			observations[j].Device.Identity.VIDPID
	})

	return observations, nil
}

func takeValue(
	key string,
	tokens []string,
) (string, []string, error) {
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf(
			"USBGuard attribute %q has no value",
			key,
		)
	}

	return tokens[0], tokens[1:], nil
}

func takeInterfaces(
	tokens []string,
) ([]string, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf(
			"with-interface has no value",
		)
	}

	if tokens[0] != "{" {
		return []string{
			strings.ToLower(tokens[0]),
		}, tokens[1:], nil
	}

	tokens = tokens[1:]

	var interfaces []string

	for len(tokens) > 0 {
		if tokens[0] == "}" {
			if len(interfaces) == 0 {
				return nil, nil, fmt.Errorf(
					"empty with-interface set",
				)
			}

			return interfaces, tokens[1:], nil
		}

		interfaces = append(
			interfaces,
			strings.ToLower(tokens[0]),
		)

		tokens = tokens[1:]
	}

	return nil, nil, fmt.Errorf(
		"unterminated with-interface set",
	)
}

func tokenize(input string) ([]string, error) {
	var tokens []string

	for len(input) > 0 {
		input = strings.TrimLeft(input, " \t\r\n")
		if input == "" {
			break
		}

		switch input[0] {
		case '{', '}':
			tokens = append(tokens, input[:1])
			input = input[1:]

		case '"':
			token, rest, err := takeQuoted(input)
			if err != nil {
				return nil, err
			}

			tokens = append(tokens, token)
			input = rest

		default:
			end := 0

			for end < len(input) {
				switch input[end] {
				case ' ', '\t', '\r', '\n', '{', '}':
					goto done
				}

				end++
			}

		done:
			if end == 0 {
				return nil, fmt.Errorf(
					"cannot tokenize USBGuard observation",
				)
			}

			tokens = append(tokens, input[:end])
			input = input[end:]
		}
	}

	return tokens, nil
}

func takeQuoted(
	input string,
) (string, string, error) {
	escaped := false

	for index := 1; index < len(input); index++ {
		switch {
		case escaped:
			escaped = false

		case input[index] == '\\':
			escaped = true

		case input[index] == '"':
			literal := input[:index+1]

			value, err := strconv.Unquote(literal)
			if err != nil {
				return "", "", fmt.Errorf(
					"decode USBGuard quoted value: %w",
					err,
				)
			}

			return value, input[index+1:], nil
		}
	}

	return "", "", fmt.Errorf(
		"unterminated USBGuard quoted value",
	)
}
