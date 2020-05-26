package main

import (
	"regexp"
	"fmt"
	"strconv"
	"strings"
)

// MetricType represents the type of a message
type MetricType int

// Message contains parsed message from client
type Message struct {
	Name string
	Value float32
	Type MetricType
	Tags []string
}

const (
	MetricGauge = MetricType(iota)
	MetricCounter
	MetricTimer
	MetricHistogram
	MetricMeter
)

var messageRegExp = regexp.MustCompile(`^([^:]*):([^\|]*)(\|[a-z]*)(.*)?$`)

// Parse one message
func ParseMessage(msg string) (*Message, error) {
	matches := messageRegExp.FindAllStringSubmatch(msg, -1)

	// You can uncomment this line for debug purposes
	// fmt.Printf("M: %#v\n", matches)

	if matches == nil || len(matches) != 1 || len(matches[0]) < 4 {
		return nil, fmt.Errorf("Invalid message: %s", msg)
	}

	res := &Message{}
	t := matches[0][3]
	switch (t) {
	case "|g":
		res.Type = MetricGauge
	case "|m":
		res.Type = MetricMeter
	case "|ms":
		res.Type = MetricTimer
	case "|h":
		res.Type = MetricHistogram
	case "|c":
		res.Type = MetricCounter
	default:
		return nil, fmt.Errorf("Invalid metric type: %s", t)
	}

	res.Name = matches[0][1]
	v, err := strconv.ParseFloat(matches[0][2], 64)
	if err != nil {
		return nil, err
	}
	res.Value = float32(v)

	if len(matches[0]) == 5 && len(matches[0][4]) > 1 && matches[0][4][0] == '|' {
		extensions := strings.Split(matches[0][4][1:], "|")
		for _, ext := range(extensions) {
			if ext[0] == '@' { // sample rate
				v, err = strconv.ParseFloat(ext[1:], 64)
				if err != nil {
					// It could break metrics so lets just ignore this value
					return nil, err
				}

				if v < 0.0000001 { // This looks suspicios
					return nil, fmt.Errorf("Sample rate is too small: %f", v)
				}

				res.Value /= float32(v)
			} else if ext[0] == '#' { // tags
				res.Tags = strings.Split(ext[1:], ",")
			}
		}
	}

	return res, nil
}

// Parse list of messages from UDP packet
func ParseMessages(data []byte) ([]*Message, error) {
	res := make([]*Message, 0, 16)
	lines := strings.Split(string(data), "\n")
	var errs []string

	for _, line := range(lines) {
		m, err := ParseMessage(line)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		res = append(res, m)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("Can't parse messages: %#v", errs)
	}

	return res, nil
}
