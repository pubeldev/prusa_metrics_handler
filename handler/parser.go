package handler

import (
	"bufio"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	handlerIdentifier = uuid.New().String()[:8]
	// Regex for RFC3164/5424 hybrid syslog header used in the python script
	syslogMsgRe   = regexp.MustCompile(`^<(\d+)>1\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)`)
	validMetricRe = regexp.MustCompile(`^[a-zA-Z_0-9]+$`)
	prefix        string
)

type Point struct {
	Timestamp   time.Time
	Measurement string
	Values      map[string]interface{}
	Tags        map[string]string
}

type Parser interface {
	Parse(text string) ([]Point, error)
	MakeDuration(ts int64) time.Duration
}

type LineProtocolParser struct {
	Version int
}

func (p *Point) IsError() bool {
	_, ok := p.Values["error"]
	return ok
}

func (lpp *LineProtocolParser) MakeDuration(ts int64) time.Duration {
	if lpp.Version < 4 {
		return time.Duration(ts) * time.Millisecond
	}
	return time.Duration(ts) * time.Microsecond
}

// parseLineProtocol is a simplified manual parser to replace the python 'line_protocol_parser'
// It handles standard "measurement,tag=v field=v time" format
func parseLineProtocol(line string) (string, map[string]string, map[string]interface{}, int64, error) {
	// This is a naive implementation. For production with complex escaping,
	// consider using "github.com/influxdata/line-protocol"
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return "", nil, nil, 0, fmt.Errorf("line too short")
	}

	// 1. Measurement and Tags
	measTags := strings.Split(parts[0], ",")
	measurement := measTags[0]
	tags := make(map[string]string)
	for _, t := range measTags[1:] {
		kv := strings.SplitN(t, "=", 2)
		if len(kv) == 2 {
			tags[kv[0]] = kv[1]
		}
	}

	// 2. Fields
	fieldsStr := parts[1]
	fields := make(map[string]interface{})
	for _, f := range strings.Split(fieldsStr, ",") {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 {
			k := kv[0]
			vStr := kv[1]
			// Try parsing number
			if strings.HasSuffix(vStr, "i") { // integer
				if v, err := strconv.ParseInt(vStr[:len(vStr)-1], 10, 64); err == nil {
					fields[k] = v
				}
			} else if v, err := strconv.ParseFloat(vStr, 64); err == nil {
				fields[k] = v
			} else { // string or boolean
				if vStr == "t" || vStr == "T" || vStr == "true" || vStr == "True" {
					fields[k] = true
				} else if vStr == "f" || vStr == "F" || vStr == "false" || vStr == "False" {
					fields[k] = false
				} else {
					fields[k] = strings.Trim(vStr, "\"")
				}
			}
		}
	}

	// 3. Timestamp (optional in standard LP, but required here)
	var ts int64
	if len(parts) == 3 {
		ts, _ = strconv.ParseInt(parts[2], 10, 64)
	}

	return measurement, tags, fields, ts, nil
}

func (lpp *LineProtocolParser) Parse(text string) ([]Point, error) {
	var points []Point
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()
		if lpp.Version >= 4 && len(line) > 10000 { // Rough equivalent to 'value too long' check
			continue
		}

		measurement, tags, fields, timeRaw, err := parseLineProtocol(line)
		if err != nil {
			log.Info().Msgf("Line format error: %v", line)
			continue
		}

		// Cleanup fields
		if len(fields) == 1 {
			if v, ok := fields["v"]; ok {
				fields = map[string]interface{}{"value": v}
			}
		}

		if lpp.Version == 3 {
			if _, ok := tags["_seq"]; ok {
				continue
			}
		}

		if !validMetricRe.MatchString(measurement) {
			log.Info().Msgf("Invalid metric name: %s", measurement)
			points = append(points, Point{
				Timestamp:   time.Unix(0, 0), // Placeholder, corrected later
				Measurement: prefix + "metric_error",
				Values: map[string]interface{}{
					"error":       "parse",
					"metric_name": measurement,
					"message":     line,
				},
				Tags: map[string]string{},
			})
			continue
		}

		// Check for NaN
		if val, ok := fields["value"]; ok {
			if fVal, isFloat := val.(float64); isFloat && math.IsNaN(fVal) {
				continue
			}
		}

		points = append(points, Point{
			Timestamp:   time.Unix(0, 0).Add(time.Duration(timeRaw)), // Store raw offset temporarily
			Measurement: prefix + measurement,
			Values:      fields,
			Tags:        tags,
		})
	}
	return points, nil
}
