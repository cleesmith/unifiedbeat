# Unifiedbeat change log

***

### v2.0.1 2016-02-28

#### Changes

* build and test using Go 1.6
* fix ```reflect+anonymous+unexported``` bug in ```gopacket```'s ```packet.go```
  * [gopacket issue](https://github.com/google/gopacket/issues/175)
* use godep to vendor dependencies
  * as Elastic Beats continues to change rapidly
  * fix for gopacket

***

### v2.0.0 2016-02-18

#### Changes

##### deleted all of the existing code
  * it was based on a clone of filebeat (_which is great for syslogs, but not unified2 files_)
  * originally, cloning filebeat was a good choice

##### designed and rewrote the entire project
* it is much simpler, more readable, and more appropriate for unified2 files
* the issue with excessive CPU usage (_70+% on all cores_) has disappeared
* followed the [Beats development guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html)
* dependencies:
  * 1. [libbeat](https://github.com/elastic/beats/tree/master/libbeat)
  * 2. [gopacket](https://github.com/google/gopacket) -- for the ```packet_dump``` field
  * 3. [geoip2-golang](https://github.com/oschwald/geoip2-golang) -- to geocode IP v4/6 addresses
  * 4. [go-unified2](https://github.com/cleesmith/go-unified2) -- to read and spool unified2 files
    * this is a fork of the original [go-unified2](https://github.com/jasonish/go-unified2)
    * with changes for the registrar feature

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
