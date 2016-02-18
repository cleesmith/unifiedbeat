/* Copyright (c) 2016 Chris Smith
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED ``AS IS'' AND ANY EXPRESS OR IMPLIED
 * WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY DIRECT,
 * INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
 * STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING
 * IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package unifiedbeat

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

// The "quit" channel is used to tell
// U2SpoolAndPublish to stop gracefully. This
// ensures the registry file is up-to-date.
var quit chan bool

// Indicates if U2SpoolAndPublish is running,
// which is needed to avoid blocking/panics
// concerning the "quit" channel.
var spoolingAndPublishingIsRunning bool

// Contains all objects needed to run the beat
type Unifiedbeat struct {
	UbConfig  ConfigSettings
	registrar *Registrar
	events    publisher.Client
}

func New() *Unifiedbeat {
	return &Unifiedbeat{}
}

func (ub *Unifiedbeat) Config(b *beat.Beat) error {
	// just load the unifiedbeat.yml config file
	err := cfgfile.Read(&ub.UbConfig, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}
	return nil
}

func (ub *Unifiedbeat) Setup(b *beat.Beat) error {
	// Go overboard checking stuff . . .
	// Also, instead of forcing the user to always find/look in the log file,
	// just log and print out critical errors and immediately exit.

	// It is possible for the Unified2Path to contain no files, as
	// there may have been no sensor alerts/events--as yet, so we
	// can not verify Unified2Prefix only that Unified2Path is valid.

	u2PathPrefixSettings := path.Join(ub.UbConfig.Sensor.Unified2Path, ub.UbConfig.Sensor.Unified2Prefix)
	// disallow filename globbing (remove all trailing *'s):
	u2PathPrefixSettings = strings.TrimRight(u2PathPrefixSettings, "*")
	// make path absolute (as it may be relative in unifiedbeat.yml):
	absPath, err := filepath.Abs(u2PathPrefixSettings)
	if err != nil {
		// this is not really an error, but it should NOT happen, so log a note:
		logp.Info("Setup: failed to set the absolute path for unified2 files: '%s'", u2PathPrefixSettings)
		absPath = u2PathPrefixSettings // whatever, just use it as-is
	}
	// ensure folder exists:
	ub.UbConfig.Sensor.Spooler.Folder = path.Dir(absPath)
	_, err = os.Stat(ub.UbConfig.Sensor.Spooler.Folder)
	if err != nil {
		// unable to find the unified2 files folder:
		fmt.Printf("Setup: call to 'os.Stat' failed with error: '%v'\n", err)
		fmt.Println("Setup: 'ERROR: unified2_path' is an invalid path; correct the YAML config file!")
		logp.Critical("Setup: ERROR: 'unified2_path' is an invalid path; correct the YAML config file!")
		os.Exit(1)
	}
	ub.UbConfig.Sensor.Spooler.FilePrefix = path.Base(absPath)

	if len(ub.UbConfig.Sensor.Rules.GenMsgMapPath) == 0 {
		fmt.Println("Setup: ERROR: required path to 'gen_msg_map_path' not specified in YAML config file!")
		logp.Critical("Setup: ERROR: required path to 'gen_msg_map_path' not specified in YAML config file!")
		os.Exit(1)
	}
	if len(ub.UbConfig.Sensor.Rules.Paths) == 0 {
		fmt.Println("Setup: ERROR: required path(s) to Rule files not specified in YAML config file!")
		logp.Critical("Setup: ERROR: required path(s) to Rule files not specified in YAML config file!")
		os.Exit(1)
	}

	if ub.UbConfig.Sensor.Geoip2Path == "" {
		logp.Info("Setup: 'geoip2_path:' not specified in YAML config file.")
	} else {
		// prefer to use GeoIP2 databases for geocoding both IPv4/6 addresses:
		err := OpenGeoIp2DB(ub.UbConfig.Sensor.Geoip2Path)
		if err != nil {
			fmt.Printf("Setup: failed opening 'GeoIp2' database; error: %v\n", err)
			logp.Critical("Setup: failed opening 'GeoIp2' database; error: %v", err)
			os.Exit(1)
		}
		logp.Info("Setup: activated 'GeoIP2' database for IP v4 and v6 geolocating.")
	}

	// load Rules and SourceFiles:
	multipleLineWarnings, duplicateRuleWarnings, err := LoadRules(ub.UbConfig.Sensor.Rules.GenMsgMapPath, ub.UbConfig.Sensor.Rules.Paths)
	if err != nil {
		fmt.Printf("Setup: loading Rules error: %v\n", err)
		logp.Critical("Setup: loading Rules error: %v", err)
		os.Exit(1)
	}
	logp.Info("Setup: Rules warnings: %v multiple line rules rejected, %v duplicate rules rejected", multipleLineWarnings, duplicateRuleWarnings)
	logp.Info("Setup: Rules stats: %v rule files read, %v rules created", len(SourceFiles), len(Rules))

	ub.events = b.Events

	// registry file is created in the current working directory:
	ub.registrar, err = NewRegistrar(".unifiedbeat")
	if err != nil {
		fmt.Printf("Setup: unable to set registry file error: %v\n", err)
		logp.Critical("Setup: unable to set registry file error: %v", err)
		os.Exit(1)
	}
	ub.registrar.LoadState()
	logp.Info("Setup: registrar: registry file: %#v", ub.registrar.registryFile)
	logp.Info("Setup: registrar: file source: %#v", ub.registrar.State.Source)
	logp.Info("Setup: registrar: file offset: %#v", ub.registrar.State.Offset)

	return nil
}

func (ub *Unifiedbeat) Run(b *beat.Beat) error {
	logp.Info("Run: start spooling and publishing...")
	// use a channel to gracefully shutdown "U2SpoolAndPublish":
	quit = make(chan bool)
	spoolingAndPublishingIsRunning = true
	ub.U2SpoolAndPublish()
	// just in case "U2SpoolAndPublish" returns unexpectedly:
	spoolingAndPublishingIsRunning = false
	// returning always calls Stop and Cleanup, and in that order
	return nil // return to "main.go" after Stop() and Cleanup()
}

// Stop is called on exit before Cleanup (unclear flow naming?)
func (ub *Unifiedbeat) Stop() {
	logp.Info("Stop: is spooling and publishing running? '%v'", spoolingAndPublishingIsRunning)
	if spoolingAndPublishingIsRunning {
		logp.Info("Stop: waiting for spooling and publishing to shutdown.")

		// tell "U2SpoolAndPublish" to shutdown:
		quit <- true

		// block/wait for "U2SpoolAndPublish" to close the quit channel:
		<-quit

		err := ub.registrar.WriteRegistry()
		if err != nil {
			logp.Info("Stop: failed to update registry file; error: %v", err)
		}
		logp.Info("Stop: updated registry file.")
	}
}

func (ub *Unifiedbeat) Cleanup(b *beat.Beat) error {
	logp.Info("Cleanup: is spooling and publishing running? '%v'", spoolingAndPublishingIsRunning)
	// see "beat/geoip2.go":
	if GeoIp2Reader != nil {
		GeoIp2Reader.Close()
		logp.Info("Cleanup: closed GeoIp2Reader.")
	}
	return nil
}
