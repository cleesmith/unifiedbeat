package beat

import (
	"fmt"
	"os"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"

	cfg "github.com/elastic/unifiedbeat/config"
	. "github.com/elastic/unifiedbeat/crawler"
	. "github.com/elastic/unifiedbeat/input"
)

// Beater object. Contains all objects needed to run the beat
type Unifiedbeat struct {
	UbConfig      *cfg.Config
	publisherChan chan []*FileEvent // channel from harvesters to spooler
	Spooler       *Spooler
	registrar     *Registrar
}

func New() *Unifiedbeat {
	return &Unifiedbeat{}
}

// Config setups up the unifiedbeat configuration by fetch all additional config files
func (ub *Unifiedbeat) Config(b *beat.Beat) error {
	// Load Base config
	err := cfgfile.Read(&ub.UbConfig, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	if len(ub.UbConfig.Unifiedbeat.Prospectors) > 1 {
		fmt.Println("\nError only one 'sensor:' may be specified in the unifiedbeat.yml file!\n")
		return fmt.Errorf("Error only one 'sensor:' may be specified in the unifiedbeat.yml file!")
	}
	// logp.Info("config=%#v\n", ub.UbConfig.Unifiedbeat.Prospectors)
	// os.Exit(999)

	// Check if optional config_dir is set to fetch additional prospector config files
	ub.UbConfig.FetchConfigs()
	return nil
}

func (ub *Unifiedbeat) Setup(b *beat.Beat) error {
	if ub.UbConfig.Unifiedbeat.Geoip2Path == "" {
		fmt.Printf("Setup: 'geoip2_path:' not specified.\n")
	} else {
		// use GeoIP2 databases for geocoding both IPv4/6 addresses:
		err := OpenGeoIp2DB(ub.UbConfig.Unifiedbeat.Geoip2Path)
		if err != nil {
			fmt.Printf("Setup: GeoIp2 error: %v\n", err)
			os.Exit(1)
		}
		// assumes that libbeat already called common.LoadGeoIPData():
		// fmt.Printf("Setup: config=%#v \t len=%v\n", b.Config.Shipper.Geoip.Paths, len(*b.Config.Shipper.Geoip.Paths))
		if publisher.Publisher.GeoLite != nil {
			// don't allow both GeoIP and GeoIP2 databases at the same time
			fmt.Printf("Setup: error: only 'geoip2_path:' or 'shipper: geoip: paths:' is allowed -- not both!\n")
			logp.Critical("Setup: error: only 'geoip2_path:' or 'shipper: geoip: paths:' is allowed -- not both!")
			os.Exit(1)
		}
		logp.Info("Setup: activated GeoIp2 for IPv4 and IPv6 geolocation information.")
	}

	// load Rules/SourceFiles
	multipleLineWarnings, duplicateRuleWarnings, err := LoadRules(ub.UbConfig.Unifiedbeat.Rules.GenMsgMapPath, ub.UbConfig.Unifiedbeat.Rules.Paths)
	if err != nil {
		fmt.Printf("Setup: loading Rules error: %v\n", err)
		logp.Critical("Setup: loading Rules error: %v", err)
		os.Exit(1)
	}
	logp.Info("Rules warnings: %v multiple line rules rejected, %v duplicate rules rejected", multipleLineWarnings, duplicateRuleWarnings)
	logp.Info("Rules stats: %v rule files read, %v rules created", len(SourceFiles), len(Rules))
	return nil
}

func (ub *Unifiedbeat) Run(b *beat.Beat) error {
	defer func() {
		p := recover()
		if p == nil {
			return
		}
		fmt.Printf("recovered panic: %v", p)
		os.Exit(1)
	}()

	var err error

	// Init channels
	ub.publisherChan = make(chan []*FileEvent, 1)

	// Setup registrar to persist state
	ub.registrar, err = NewRegistrar(ub.UbConfig.Unifiedbeat.RegistryFile)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	crawl := &Crawler{
		Registrar: ub.registrar,
	}

	// Load the previous log file locations now, for use in prospector
	ub.registrar.LoadState()

	// Init and Start spooler: Harvesters dump events into the spooler.
	ub.Spooler = NewSpooler(ub)
	err = ub.Spooler.Config()
	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	// Start up spooler
	go ub.Spooler.Run()

	crawl.Start(ub.UbConfig.Unifiedbeat.Prospectors, ub.Spooler.Channel)

	// Publishes event to output
	go Publish(b, ub)

	// registrar records last acknowledged positions in all files.
	ub.registrar.Run()

	return nil
}

func (ub *Unifiedbeat) Cleanup(b *beat.Beat) error {
	if GeoIp2Reader != nil {
		GeoIp2Reader.Close()
		logp.Info("Cleanup: closed GeoIp2Reader.")
	}
	return nil
}

// Stop is called on exit for cleanup
func (ub *Unifiedbeat) Stop() {
	logp.Info("Stopping unifiedbeat")
	// Stop harvesters
	// Stop prospectors

	// Stopping spooler will flush items
	ub.Spooler.Stop()

	// Stopping registrar will write last state
	ub.registrar.Stop()

	// Close channels
	//close(ub.publisherChan)
}

func Publish(beat *beat.Beat, ub *Unifiedbeat) {
	logp.Info("Start sending events to output")

	// Receives events from spool during flush
	for events := range ub.publisherChan {
		pubEvents := make([]common.MapStr, 0, len(events))
		for _, event := range events {
			pubEvents = append(pubEvents, event.ToMapStr())
		}

		beat.Events.PublishEvents(pubEvents, publisher.Sync)
		logp.Info("Events sent: %d", len(events))

		// Tell the registrar that we've successfully sent these events
		ub.registrar.Channel <- events
	}
}
