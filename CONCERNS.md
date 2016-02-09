# Unifiedbeat

### January 18, 2016

#### Sooner

1. ensure ```etc/unifiedbeat.template.json``` accounts for all known fields in [Unified2 File Format](http://manual.snort.org/node44.html)
  * or is it good enough to just rely on _mappings: _default_: _all: dynamic_templates:_ setting, after all:
    * most fields are the **long** datatype
    * the next most fields are the **string** datatype, which are the only fields that need **.raw**
    * some fields are the **boolean** datatype
    * a few fields are the **date** datatype, such as **@timestamp** and **indexed_at**
    * a few fields are the **geo_point** datatype, such as **src_location** and **dst_location**
    * a few fields are the **ip** datatype (_which is really a **long** but for IPv4_), such as **src_ip** and **dst_ip**
    * see [field datatypes](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-types.html)

1. in **etc/unifiedbeat.yml** allow a user to config an association between **sensor_id**:
  * and a **sensor_hostname**
  * and a **sensor_interface** (_e.g. eth0_)
  * or should this be done using ```unifiedbeat.prospectors.fields``` in **etc/unifiedbeat.yml**?
  * after all, searching for ```sensor_id: 0``` isn't very meaningful

1. improve geolocation (lat/lng) for both source and destination IPs
  * currently limited to IPv4 addresses when using [go-libGeoIP](https://github.com/nranchev/go-libGeoIP) as provided by libbeat
  * try [maxminddb-golang](https://github.com/oschwald/maxminddb-golang) for both IPv4 and IPv6 addresses which uses GeoLite2 and GeoIP2 databases
    * but what about memory mapping (**mmap**) the database as compared to loading the entire DB into memory
      * will this cause performance or other issues?
    * see also [geoip2-golang](https://github.com/oschwald/geoip2-golang) which is built using maxminddb-golang

1. create generic source and destination IP string fields:
  * src_ip can contain either src_ipv4 or src_ipv6
  * dst_ip can contain either dst_ipv4 or dst_ipv6
  * would this really make searching any easier?

1. ~~test logging at the INFO level to /var/log/unifiedbeat/unifiedbeat~~ [works]

1. ~~ensure all testing/debugging **fmt.Print**'s have been removed or changed to **logp.Info()**~~ [done]

***

#### Later

1. remove all of the _line oriented text file handling_ code
  * none of this code is needed, as unified2 files are binary
  * why are unified2 files are binary?
    * the binary format allows Snort/Suricata to quickly log alerts without missing any network traffic

1. handle **classification_id** (_maybe_)
  * classification_id _maps_ to a line number in the **classification.config** file (ignoring any comment lines)
  * this seems to be a very _error prone and loosey–goosey_ approach
  * given that the **classtype** is provided in most rules, is mapping the classification_id really useful ?

1. ensure that the only valid **output** for unifiedbeat is ElasticSearch
  * not Logstash, nor a file, nor the console
  * keep the _use case_ as simple as possible, at least initially
  * keep the focus on the end goal of using full-text search for security analysis
  * the combination of **unifiedbeat, elasticsearch, and kibana** can be a replacement for:
    * Squert, Sguil, Wireshark, and Snorby
      * well, for unified2 files, but Sguil and Wireshark are still useful for pcap files
    * ELSA
      * which involves Snort, Syslog-NG, MySQL, Sphinx Search and several truckloads of Perl
      * wow, that's a bit like _spinning plates on sticks_, and a lot of stuff to manage, but it's still in use
    * see Security Onion below

1. could [Kibi](http://siren.solutions/kibi) also be used for analysis?
  * not so much for it's SQL integrations
  * but for _joining documents_ based on **_type** and **event_id** fields
  * or is this already possible using a [terms-query](https://www.elastic.co/guide/en/elasticsearch/reference/2.1/query-dsl-terms-query.html)?

1. experiment with ElasticSearch's snapshot/restore to handle backups of unifiedbeat indices
  * for staging/removing indices during/after analysis ... why?
    * because IDS data can be voluminous
    * it's desirable to keep ElasticSearch lean and fast
  * are old snapshots guaranteed to work with new releases of ElasticSearch?

1. ~~reviewed and removed the  ```etc/fields.yml``` file, as this is not needed for unifiedbeat~~ [done]

1. ~~do relative and symlink paths from **etc/unifiedbeat.yml** work properly~~ [it works]

1. ~~what happens when a file that's being harvested gets deleted or renamed~~ [it still works]

***

#### Security Onion

1. check if Unifiedbeat, Java, ElasticSearch, and Kibana could be added to [Security Onion](https://security-onion-solutions.github.io/security-onion/) distributions
  * maybe as a replacement for Snorby, which is no longer maintained and being removed from Security Onion
  * install Security Onion on a VM and test unifiedbeat, java, elasticsearch, and kibana

1. in **etc/unifiedbeat.yml** add a user config field named **capme_url** for use on Security Onion
  * under the ```unifiedbeat.prospectors.fields``` setting
  * allows _pivoting_ from kibana to [CapME](https://github.com/Security-Onion-Solutions/security-onion/wiki/CapMeAuthentication) for full transcripts captured via [netsniff-ng](https://github.com/netsniff-ng/netsniff-ng)
  * see **Core Components** in the [Introduction](https://github.com/Security-Onion-Solutions/security-onion/wiki/IntroductionToSecurityOnion)

***

#### Code

##### Packages and Folders

* unifiedbeat's flow and processing is as follows:
  * ```config -> prospect -> harvest   -> spool``` _... logically speaking_, see [Overview](https://raw.githubusercontent.com/cleesmith/unifiedbeat/master/screenshots/unifiedbeat.png "overview of unifiedbeat processing")
  * yet the packages and folders are named:
  * ```config -> crawler  -> harvester -> beat```
    * is this a side effect of being based on libbeat?
    * _crawler_ kind of implies _web crawler_, and not file prospecting, at first glance
    * _beat_ conveys nothing as we know this is a beat
  * for unifiedbeat, most of the work is actually done in the ```input``` package
    * via function **ToMapStr()** in ```input/file.go```
    * which translates unified2 records into JSON documents for indexing into ElasticSearch
    * seems like everything in the ```input``` package should be in ```harvester```
* this _disconnect_ between logical flow and package names makes it difficult to:
  * read and understand the code, well, at a glance
  * perform any future bugfixes or refactoring without re-learning about this _disconnect_

***
***
