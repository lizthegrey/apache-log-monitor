package monitor

import (
	"fmt"
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
	Url       url.URL
	Status    int
	Size      int64
}

func (l *LogLine) Equal(other *LogLine) error {
	if l == nil || other == nil {
		if l == nil && other == nil {
			return nil
		}
		return fmt.Errorf("Existence: %s != %s", l, other)
	}
	if !l.Timestamp.Equal(other.Timestamp) {
		return fmt.Errorf("Timestamp: %s != %s", l.Timestamp, other.Timestamp)
	}
	if l.IpAddr != other.IpAddr {
		return fmt.Errorf("IpAddr: %s != %s", l.IpAddr, other.IpAddr)
	}
	if l.Method != other.Method {
		return fmt.Errorf("Method: %s != %s", l.Method, other.Method)
	}
	if l.Url != other.Url {
		return fmt.Errorf("Url: %s != %s", l.Url, other.Url)
	}
	if l.Status != other.Status {
		return fmt.Errorf("Status: %s != %s", l.Status, other.Status)
	}
	if l.Size != other.Size {
		return fmt.Errorf("Size: %s != %s", l.Size, other.Size)
	}
	return nil
}

func LogParse(line string) (*LogLine, error) {
	results := extract.FindStringSubmatch(line)
	if results == nil {
		return nil, fmt.Errorf("Failed to parse line: %s", line)
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
		Url:       *uri,
		Status:    status,
		Size:      size,
	}, nil
}
