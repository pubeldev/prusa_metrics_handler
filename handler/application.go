package handler

import (
	"fmt"
	"math"
	"os"
	"pubeldev/prusa_metrics_handler/config"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/rs/zerolog/log"
)

type Application struct {
	Client     influxdb2.Client
	WriteAPI   api.WriteAPI
	SyslogAddr string
	PointsChan chan Point
	Printers   map[string]*RemotePrinter
	PrintersMu sync.RWMutex
	Parsers    map[int]Parser
	Tags       map[string]string
	Prefix     string
}

func NewApplication(client influxdb2.Client, cfg config.Config) *Application {
	writeAPI := client.WriteAPI(cfg.DBOrg, cfg.DBBucket)

	hostname, _ := os.Hostname()
	prefix = cfg.Prefix

	return &Application{
		Client:     client,
		WriteAPI:   writeAPI,
		SyslogAddr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		PointsChan: make(chan Point, 10000), // Buffer
		Printers:   make(map[string]*RemotePrinter),
		Parsers: map[int]Parser{
			2: &LineProtocolParser{Version: 2},
			3: &LineProtocolParser{Version: 3},
			4: &LineProtocolParser{Version: 4},
		},
		Tags: map[string]string{
			"hostname":           hostname,
			"port":               strconv.Itoa(cfg.Port),
			"session_start_time": time.Now().UTC().Format("2006-01-02 15:04:05"),
		},
	}

}

func (app *Application) GetPrinter(mac string) *RemotePrinter {
	app.PrintersMu.Lock()
	defer app.PrintersMu.Unlock()

	if _, ok := app.Printers[mac]; !ok {
		app.Printers[mac] = NewRemotePrinter(mac)
	}
	return app.Printers[mac]
}

func (app *Application) ProcessMessage(data []byte) {
	strData := string(data)
	matches := syslogMsgRe.FindStringSubmatch(strData)

	if matches == nil {
		return
	}
	// Regex groups: 0:Full, 1:PRI, 2:TM, 3:HOST(MAC), 4:APP, 5:PROCID, 6:SD, 7:MSGID, 8:MSG
	// Python re groupdict: HOST=matches[3], APP=matches[4], MSG=matches[8]

	appLabel := matches[4]
	if appLabel != "buddy" { //
		return
	}

	macAddress := matches[3]
	msgContent := strings.TrimSpace(matches[8])

	// Split Header and Serialized Points
	parts := strings.SplitN(msgContent, " ", 2)
	if len(parts) < 2 {
		return
	}
	headerStr, serializedPoints := parts[0], parts[1]

	// Parse Header: "v=1,msg=10,tm=..."
	headerDict := make(map[string]string)
	for _, kv := range strings.Split(headerStr, ",") {
		pair := strings.Split(kv, "=")
		if len(pair) == 2 {
			headerDict[pair[0]] = pair[1]
		}
	}

	printer := app.GetPrinter(macAddress)
	app.HandleParsedMessage(printer, headerDict, serializedPoints, len(data))
}

func (app *Application) HandleParsedMessage(printer *RemotePrinter, header map[string]string, serializedPoints string, dataSize int) {
	vStr := header["v"]
	if vStr == "" {
		vStr = "1"
	}
	version, _ := strconv.Atoi(vStr)

	parser, ok := app.Parsers[version]
	if !ok {
		log.Info().Msgf("received unsupported version %d", version)
		return
	}

	points, err := parser.Parse(serializedPoints)
	if err != nil {
		return
	}

	msgID, _ := strconv.Atoi(header["msg"])
	printer.RegisterReceivedMessage(msgID)

	// Log UDP Datagram metric
	app.PointsChan <- Point{
		Timestamp:   time.Now().UTC(),
		Measurement: prefix + "udp_datagram",
		Values: map[string]interface{}{
			"size":   8 + dataSize,
			"points": len(points),
		},
		Tags: map[string]string{"mac_address": printer.MacAddress},
	}

	// Timestamp Logic
	tmVal, _ := strconv.ParseInt(header["tm"], 10, 64)
	msgDelta := parser.MakeDuration(tmVal)

	printer.Mu.Lock()
	if printer.LastReceivedMsgID == nil || math.Abs(float64(*printer.LastReceivedMsgID-msgID)) >= 10 {
		printer.SessionStartTime = time.Now().UTC().Add(-msgDelta)
	}
	// Update Last ID
	newID := msgID
	printer.LastReceivedMsgID = &newID
	baseTimestamp := printer.SessionStartTime.Add(msgDelta)
	printer.Mu.Unlock()

	currentTimestamp := baseTimestamp

	for _, pt := range points {
		// pt.Timestamp currently holds the raw duration/int offset inside a Time object
		rawOffset := pt.Timestamp.Sub(time.Unix(0, 0))

		if version <= 2 {
			// v2: relative to previous point
			currentTimestamp = currentTimestamp.Add(parser.MakeDuration(int64(rawOffset)))
			pt.Timestamp = currentTimestamp
		} else {
			// v3/4: relative to base
			pt.Timestamp = baseTimestamp.Add(parser.MakeDuration(int64(rawOffset)))
		}

		// Merge Tags
		pt.Tags["mac_address"] = printer.MacAddress
		for k, v := range app.Tags {
			pt.Tags[k] = v
		}

		app.PointsChan <- pt
	}
}
