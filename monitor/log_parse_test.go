package monitor

import (
	"testing"

	"errors"
	"net/url"
	"time"
)

func TestParsing(t *testing.T) {
	cases := []struct {
		in   string
		want *LogLine
		err  error
	}{
		{
			"127.0.0.1 - frank [11/Oct/2000:13:55:36 -0700] \"GET /apache_pb.gif HTTP/1.0\" 200 2326",
			&LogLine{
				Timestamp: time.Date(2000, 10, 11, 13, 55, 36, 0, time.FixedZone("-0700", -7*60*60)),
				IpAddr:    "127.0.0.1",
				Method:    "GET",
				Url:       url.URL{Path: "/apache_pb.gif"},
				Status:    200,
				Size:      2326,
			},
			nil,
		},
		{"bogus", nil, errors.New("Failed to parse line: bogus")},
	}
	for _, c := range cases {
		got, err := LogParse(c.in)

		if lineDiff := got.Equal(c.want); lineDiff != nil || !((err == nil && c.err == nil) || err.Error() == c.err.Error()) {
			t.Errorf("LogParse(%q)\nResult: (%q, %t)\nWanted: (%q, %t)\nDiff: %s", c.in, got, err, c.want, c.err, lineDiff)
		}
	}
}
