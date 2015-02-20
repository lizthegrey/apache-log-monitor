package monitor

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// Common Log Format fields:     " %h      %l    %u    %t              \"%m       %U%q    %H    \" %>s      %b       "
// Extraction index:               $1                     $2             $3       $4               $5       $6
var extract = regexp.MustCompile("^([^ ]+) [^ ]+ [^ ]+ \\[([^\\]]+)\\] \"([A-Z]+) ([^ ]+) [^\"]+\" ([0-9]+) ([0-9]+)$")

type LogLine struct {
	Timestamp time.Time
	IpAddr    string
	Method    string
	Url       *url.URL
	Status    int
	Size      int64
}

func LogParse(line string) (*LogLine, error) {
	results := extract.FindStringSubmatch(line)
	if results == nil {
		return nil, errors.New("Failed to parse line.")
	}

	uri, err := url.Parse(results[4])
	if err != nil {
		return nil, err
	}

	// Use time specification specified by CLF.
	stamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", results[2])
	if err != nil {
		return nil, err
	}

	status, err := strconv.Atoi(results[5])
	if err != nil {
		return nil, err
	}

	size, err := strconv.ParseInt(results[6], 10, 0)
	if err != nil {
		return nil, err
	}

	return &LogLine{
		Timestamp: stamp,
		IpAddr:    results[1],
		Method:    results[3],
		Url:       uri,
		Status:    status,
		Size:      size,
	}, nil
}
