# Unifiedbeat change log

***

### v2.0.0 2016-02-18

#### Changes

* deleted all of the existing code
  * because it was based on a clone of filebeat (_which is great for syslogs, but not unified2 files_)
  * originally, cloning filebeat was a good choice
    * given a server might be used to monitor data from multiple sensors
    * however, it is **much simpler to just execute a unifiedbeat process** for each sensor
    * after all, each sensor:
      * may have a different set of rules
      * different file locations and prefix names
      * can be monitored and managed (_started, stopped, or archived_) individually
* designed and rewrote the entire project
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
* concerns/todo's:
  * ensure all ```fmt.Print```'s are changed to ```logp.Info```'s
  * how to install/upgrade Go
    * without using gvm, but manually
  * how to best handle dependencies:
    * **vendor** all dependencies (_lockdown project to the **known and working**_)
    * godep doesn't seem to work with gvm
    * use glide ?
  * it's unfortunate that installs, upgrades, and dependencies are still a pain (_just like in ruby, python, or whatever_)
  * don't run unifiedbeat on Security Onion unless you stop Snort first
    * to stop Snort do ```sudo nsm_sensor_ps-stop```
    * otherwise, snort triggers an alert for every request/response to/from ElasticSearch
    * otherwise, there is an endless loop of indexing and we can never catch up
    * or edit the rules to not trigger these alerts (_probably not a good idea_)
    * this is another good reason to keep elasticsearch on the same server with the sensor data
    * this may also be true if you are forwarding to Logstash (_this was not tested_)

***

### v1.3 2016-02-04

#### Changed

* after replaying several pcap's into snort the following was a bad idea:
  * ~~minimized ```unifiedbeat.template.json``` template (field mappings)~~
  * it does cause records to fail to index into ES
  * solution: **explicitly define all known fields** in ```unifiedbeat.template.json```
    * this is tedious and error prone
    * but once defined it should never need to be changed, unless the unified file format changes
* warn about unified2 files that are older than 24 hours ago
  * if they are to be indexed then do: ```touch /var/log/snort/*.log*``` or similar
* Filebeat now has a new config setting called ```close_older```
  * research how this may affect unifiedbeat
  * would this help to avoid **all** log files being opened and tailed
    * then again, someone should be _managing_ these files and archiving them appropriately
    * _big data_ doesn't mean unmanaged data growth, as there are limits after all
* enable the ```logging:``` section in ```etc/unifiedbeat.yml file```
  * logs are in ```/var/log/unifiedbeat/unifiedbeat```
  * the logs autorotate every 10MB and at most 3 are kept, but these settings can be changed
  * viewing the logs help to determine:
    * if harvesters were started, i.e. which unified2 files are being indexed
    * any errors, such as:
      * duplicate or invalid rules
      * indexing errors from ES
      * json related errors
* todo:
  * package unifiedbeat as a zip file that provides:
    * YAML and JSON files
    * GeoIP2 database file
    * sample test files:
      * unified2 file from snort
      * rules files

***

### v1.2 2016-01-26

#### Changed

* minimized ```unifiedbeat.template.json``` template (field mappings)
  * concern: records with bad data may cause incorrect mappings leading to ES errors
* support GeoIP2 database for geocoding both IPv4 and IPv6
  * only loads GeoIP or GeoIP2 databases, not both
* source IP (only for _"_type": "event"_):
  * src_ip (string)
  * src_ipv4 (ip)
  * src_ipv6 (string)
  * src_country_code (ISO 2 character string)
  * src_location (geo_point: "latitude, longitude")
* destination IP (only for _"_type": "event"_):
  * dst_ip (string)
  * dst_ipv4 (ip)
  * dst_ipv6 (string)
  * dst_country_code (ISO 2 character string)
  * dst_location (geo_point: "latitude, longitude")

***

### v1.1 2016-01-18

#### Added

* NIDS (Snort/Suricata) Rules
  * to lookup ```generator_id``` + ```signature_id```
  * index rule info into ElasticSearch for **event** type records
  * new fields: signature, rule_raw, rule_source_file, and rule_source_file_line_number
  * having the rule info included _as of the time the event happened_ avoids:
    * lookups at run time, which is unnecessary processing
    * the event becoming _out-of-sync_ with the rule that triggered it due to rule updates over time

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

> consider using the typical 3 digit version numbers, like 1.1.1, but unifiedbeat probably won't change that often

***

All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

***
***
