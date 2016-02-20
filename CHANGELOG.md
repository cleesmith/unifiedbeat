# Unifiedbeat change log

***

### v2.0.0 2016-02-18

#### Changes

##### deleted all of the existing code
  * it was based on a clone of filebeat (_which is great for syslogs, but not unified2 files_)
  * originally, cloning filebeat was a good choice
    * given a server might be used to monitor data from multiple sensors
    * however, it is **much simpler to just execute a unifiedbeat process** for each sensor
    * after all, each sensor:
      * may have a different set of rules
      * different file locations and prefix names
      * can be monitored and managed (_started, stopped, or archived_) individually

##### designed and rewrote the entire project
* it is much simpler, more readable, and more appropriate for unified2 files
* the issue with excessive CPU usage (_70+% on all cores_) has disappeared
* followed the [Beats development guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html)
* upgraded to Go 1.5.3 (_sheesh! 1.6 was just released_)
* dependencies:
  * 1. [libbeat](https://github.com/elastic/beats/tree/master/libbeat) -- _to beat, or not to beat ..._
  * 2. [gopacket](https://github.com/google/gopacket) -- for the ```packet_dump``` field with it's thorough packet details
  * 3. [geoip2-golang](https://github.com/oschwald/geoip2-golang) -- to geocode IP v4/6 addresses
  * 4. [go-unified2](https://github.com/cleesmith/go-unified2) -- to read and spool unified2 files
    * this is a fork of the original [go-unified2](https://github.com/jasonish/go-unified2)
    * with changes for the registrar feature
      * to update a **bookmark** file -- **.unifiedbeat**
      * which tracks the **offset** into the unified2 file that's currently being tailed
      * the bookmark file is only written to disk upon _graceful_ program termination
        * otherwise the offset is kept in memory, which avoids constantly writing to disk
        * so don't _yank the plug_ on the server running unifiedbeat and expect to resume properly
* todos:
  * ensure all ```fmt.Print```'s are changed to ```logp.Info```'s
  * remove **quit** channel code and replace with **isAlive** boolean
    * how to wait for spool/publish during _graceful termination_ ?
  * ~~create a Linux 64bit binary release file~~ [_done Feb 19, 2016_]
* concerns:
  * deleting the unified2 being tailed causes unifiedbeat to exit immediately
    * why or how would this ever occur?
    * occurs in ```spoolrecordreader.go```'s ```Name()``` func
      * error message is _Next: unexpected error reading from..._
  * don't ```wget https://github.com/cleesmith/unifiedbeat/blob/master/var/GeoIP/GeoLite2-City.mmdb```
    * instead download **GeoLite2 City** database from [MaxMind](http://dev.maxmind.com/geoip/geoip2/geolite2/)
  * how to install/upgrade Go
    * without using gvm, but manually
  * how to best handle dependencies:
    * **vendor** all dependencies (_lockdown project to the **known and working**_)
    * godep doesn't seem to work with gvm
    * use glide ?
  * it's unfortunate that installs, upgrades, and dependencies are still a pain (_just like in ruby, python, or whatever_)
  * don't run unifiedbeat on Security Onion (SO) unless you stop Snort first
    * to stop Snort do ```sudo nsm_sensor_ps-stop```
    * otherwise, snort triggers an alert for every request/response to/from a remote ElasticSearch (ES)
    * otherwise, there is an endless loop of indexing and it can never catch up
    * after all, a sensor is watching inbound/outbound network traffic
    * or edit the rules to not trigger these alerts (_probably not a good idea_)
      * see: [Managing Alerts](https://github.com/Security-Onion-Solutions/security-onion/wiki/ManagingAlerts#suppressions)
    * yet another reason to run ElasticSearch on the same server with the sensor data
      * install ES on SO at ```127.0.0.1:9200``` then test
    * this is probably true when forwarding to Logstash (_this was not tested_)

***

### v1.0 2016-01-13

#### Initial Release

* this was a ```git clone``` of Filebeat as of 2015-11-25 with these changes:
  * remove _line-oriented text_ file reading
  * add unified2 file (binary) format reading via [go-unified2](https://github.com/jasonish/go-unified2)
  * index separate document ```_type```'s for each unified2 record: **event**, **packet**, **extradata**
  * use [gopacket](https://github.com/google/gopacket) for:
    * packet layers
    * a human readable ```packet_dump``` that is indexed
  * add geolocation for source/destination IPs via [go-libGeoIP](github.com/nranchev/go-libGeoIP) (included with libbeat)

***

All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

***
***
