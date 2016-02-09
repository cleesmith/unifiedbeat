# Unifiedbeat

### January 18, 2016

> ### Protect the box
>
> * disk space and memory are cheap
> * ElasticSearch is fast and easy to install (no more SQL database)
> * Unifiedbeat indexes **all** unified2 data to assist with **incident** analysis (intrusion detective work)
> * ElasticSearch simplifies managing both historical and current incident data via clusters/nodes/indices
> * your **networks** and **data** are at risk

#### Features

* Each Unified2 record type (IDS Event, Packet, Extra Data) is indexed as a separate document "**_type**"
  * this avoids issues caused by trying to coalesce records into an alert/event as a single document/record
  * record types may be linked together using the **event_id** field to _recreate_, if possible, the alert/event
  * there is no guaranteed order to the records in an unified2 file ... events and packets may not be in sequence
  * on a busy IDS sensor the **event_id** field may **wrap**, it's only 32 bits, further complicating "alert" recreation
  * so, recreating an "alert" can be tricky, see this [discussion](http://seclists.org/snort/2011/q2/619)
  * of course, _joins and lookups_ are tricky in elasticsearch, see [terms-query](https://www.elastic.co/guide/en/elasticsearch/reference/2.1/query-dsl-terms-query.html)
    * this may complicate analysis when using Kibana
    * but _bouncing_ between the Discover, Visualize, and Dashboard tabs isn't too bad

* Along with the alert/event details, the **triggering Snort rule/signature** is indexed
  * this captures the rule in use at the time the alert/event was triggered
  * this avoids an alert/event becoming out-of-sync with it's triggering rule due to rule changes/updates
  * the details of each rule are indexed in every document of **type: "event"**:
    * generator ID (gid)
    * signature ID (sid)
    * message (signature)
    * path to the source file containing the rule
    * line number of the rule within the source file
    * the actual **raw** rule text as read from the source file
  * **multi-line** rules are detected and **ignored** (_for now_)
  * **duplicate** (gid+sid) rules are detected and **ignored** (_for now, the first rule seen is the winner_)

* Everything regarding an alert/event is now available for full-text searching and visualizing, including:
  * timestamp as **seconds** and **microseconds** since the epoch of when the alert/event was generated
  * complete network packet(s) data/payload, as well as human readable detailed packet dumps
  * geolocation (lat/lng) for both source and destination IPs, currently limited to IPv4 addresses
  * rule/signature information for each alert/event (no more _out-of-sync rule lookups_)
  * no attempt is made at combining events and packets to reconstruct an "alert" during indexing
  * all records in the unified2 files are simply indexed "as-is" into ElasticSearch
    * what the IDS captured is what appears in ElasticSearch, even if the network data is improper/invalid

* Unifiedbeat has **no additional dependencies**
  * just unified2 files, rules files, Java, ElasticSearch, and the unifiedbeat binary file
  * no C libraries, perl, python, or ruby installations or compilations are needed

* Sample files are provided in the **sample_data** folder
  * these files may be used to verify that unifiedbeat is working as expected
  * a unified2 file is provided in **sample_data/snort.log.1452978988**
  * **sample_data/newdat3.log** is the pcap file that was read by Snort to create snort.log.1452978988
    * how to details [here](http://manual.snort.org/node8.html)
  * the rules triggered by Snort reading **sample_data/newdat3.log** are in the **sample_data/rules** folder
  * the following screenshots are based on the sample files provided

* While there exist other indexing approaches like unifiedbeat, unfortunately, all of them:
  * involve additional and non-typical configuration for Snort/Suricata to obtain JSON output
  * require multiple steps to convert unified2 files to JSON and then index into ElasticSearch
  * use additional storage space for intermediate JSON/other files

***
***
