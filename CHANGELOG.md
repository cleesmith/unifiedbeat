# Unifiedbeat change log

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