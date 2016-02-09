package main

import (
	"github.com/elastic/libbeat/beat"
	unifiedbeat "github.com/elastic/unifiedbeat/beat"
)

var Version = "1.1"
var Name = "unifiedbeat"

// The basic model of execution:
// - Prospector: finds files in paths/globs to harvest, starts harvesters
// - Harvester: reads a file, sends events to the spooler
// - Spooler: buffers events until ready to flush to the publisher
// - Publisher: writes to the network, notifies registrar
// - Registrar: records positions of files read
// Finally, a Prospector uses the registrar information, on restart,
// to determine where in each file to restart a harvester.

func main() {
	beat.Run(Name, Version, unifiedbeat.New())
}
