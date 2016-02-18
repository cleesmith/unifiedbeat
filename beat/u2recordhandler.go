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
	"net"
	"path/filepath"
	"time"
	"unicode"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/cleesmith/go-unified2"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const minASCII = '\u001F' // 31

// FileEvent is sent to the output and must contain all relevant information
type FileEvent struct {
	ReadTime        time.Time
	Source          string
	InputType       string
	DocumentType    string
	Offset          int64
	U2Record        interface{}
	Fields          *map[string]string
	fieldsUnderRoot bool
}

// SetFieldsUnderRoot sets whether the fields should be added
// top level to the output documentation (fieldsUnderRoot = true) or
// under a fields dictionary.
func (f *FileEvent) SetFieldsUnderRoot(fieldsUnderRoot bool) {
	f.fieldsUnderRoot = fieldsUnderRoot
}

func (f *FileEvent) ToMapStr() common.MapStr {
	event := common.MapStr{
		"indexed_at":    common.Time(f.ReadTime),
		"source":        f.Source,
		"source_offset": f.Offset,
		"type":          f.DocumentType,
		"input_type":    f.InputType,
	}
	// handle unified2 record types, see record type structs in:
	//   ~/go/src/github.com/elastic/unifiedbeat/vendor/github.com/jasonish/go-unified2/unified2.go
	var es, ems, ps, pms uint32
	var ut time.Time
	switch f.U2Record.(type) {
	case *unified2.EventRecord:
		event["type"] = "event" // set document type to match unified2 record type
		event["record_type"] = "event"
		// must assert ".(*unified2.EventRecord)." coz record is an interface{}
		event["sensor_id"] = f.U2Record.(*unified2.EventRecord).SensorId
		event["event_id"] = f.U2Record.(*unified2.EventRecord).EventId
		es = f.U2Record.(*unified2.EventRecord).EventSecond
		event["event_second"] = es
		ems = f.U2Record.(*unified2.EventRecord).EventMicrosecond
		event["event_microsecond"] = ems
		ut = time.Unix(int64(es), int64(ems)*1000) // nanosecs = microsecs * 1000
		event["@timestamp"] = common.Time(ut)
		event["signature_revision"] = f.U2Record.(*unified2.EventRecord).SignatureRevision
		event["classification_id"] = f.U2Record.(*unified2.EventRecord).ClassificationId
		event["priority"] = f.U2Record.(*unified2.EventRecord).Priority

		event["generator_id"] = f.U2Record.(*unified2.EventRecord).GeneratorId // GeneratorId uint32
		event["signature_id"] = f.U2Record.(*unified2.EventRecord).SignatureId // SignatureId uint32
		// SourceFiles, Rules, and Rule are available coz "rules.go" is part of this package
		gs := fmt.Sprint(event["generator_id"]) + ":" + fmt.Sprint(event["signature_id"])
		aRule, ok := Rules[gs]
		if ok {
			absPath, err := filepath.Abs(SourceFiles[aRule.SourceFileIndex])
			if err != nil {
				absPath = SourceFiles[aRule.SourceFileIndex] // ok, just use it as-is
			}
			event["rule_source_file"] = absPath
			event["rule_source_file_line_number"] = aRule.SourceFileLineNum
			event["signature"] = aRule.Msg
			event["rule_raw"] = aRule.RuleRaw
		} else {
			logp.Info("ToMapStr: lookup gid+sid:%v failed to find rule\n", gs)
		}

		// handle src/dst IPs:
		//   src_ip,   dst_ip    string -- must ALWAYS have it's value set!
		//   src_ipv4, dst_ipv4  "type": "ip" (ES)
		//   src_ipv6, dst_ipv6  string
		event["src_ip"] = net.IP(f.U2Record.(*unified2.EventRecord).IpSource).String()
		ip4, _, ips := isIP(event["src_ip"].(string))
		if ip4 {
			event["src_ipv4"] = ips
		} else {
			event["src_ipv6"] = ips
		}
		event["dst_ip"] = net.IP(f.U2Record.(*unified2.EventRecord).IpDestination).String()
		// event["dst_ip"] = "::b110:c400" // IPv6 in BR Brazil
		ip4, _, ips = isIP(event["dst_ip"].(string))
		if ip4 {
			event["dst_ipv4"] = ips
		} else {
			event["dst_ipv6"] = ips
		}

		// handle geolocation for source/destination IPs
		// use GeoLite (ipv4 only) or GeoLite2 (both ipv4/6):
		//   - if both specified then use GeoIp2Reader
		//   - if neither specified then don't do anything
		if GeoIp2Reader != nil {
			loc := GetLocationByIP(event["src_ip"].(string)) // always returns a *geoip2.City struct
			if loc != nil && loc.Location.Latitude != 0 && loc.Location.Longitude != 0 {
				event["src_country_code"] = loc.Country.IsoCode
				latlng := fmt.Sprintf("%f, %f", loc.Location.Latitude, loc.Location.Longitude)
				event["src_location"] = latlng
			}
			loc = GetLocationByIP(event["dst_ip"].(string)) // always returns a *geoip2.City struct
			if loc != nil && loc.Location.Latitude != 0 && loc.Location.Longitude != 0 {
				event["dst_country_code"] = loc.Country.IsoCode
				latlng := fmt.Sprintf("%f, %f", loc.Location.Latitude, loc.Location.Longitude)
				event["dst_location"] = latlng
			}
			// feb 2016: only support GeoIP2 and not GeoIP, so
			// just ignore the message logged by libbeat:
			//   "GeoIP disabled: No paths were set under shipper.geoip.paths"
			// } else if publisher.Publisher.GeoLite != nil {
			// 	aIP, exists := event["src_ip"]
			// 	// limited to IPv4
			// 	if exists && len(aIP.(string)) > 0 {
			// 		loc := publisher.Publisher.GeoLite.GetLocationByIP(aIP.(string))
			// 		if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
			// 			event["src_country_code"] = loc.CountryCode
			// 			loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
			// 			// if latitude/longitude values are found then add the location field as GeoJSON array:
			// 			event["src_location"] = loc
			// 		}
			// 	}
			// 	aIP, exists = event["dst_ip"]
			// 	// limited to IPv4
			// 	if exists && len(aIP.(string)) > 0 {
			// 		loc := publisher.Publisher.GeoLite.GetLocationByIP(aIP.(string))
			// 		if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
			// 			event["dst_country_code"] = loc.CountryCode
			// 			loc := fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
			// 			// if latitude/longitude values are found then add the location field as GeoJSON array:
			// 			event["dst_location"] = loc
			// 		}
			// 	}
		}

		event["sport"] = f.U2Record.(*unified2.EventRecord).SportItype
		event["dport"] = f.U2Record.(*unified2.EventRecord).DportIcode
		// maybe: proto_map = {1: "ICMP", 6: "TCP", 17: "UDP"}
		event["protocol"] = f.U2Record.(*unified2.EventRecord).Protocol
		event["impact_flag"] = f.U2Record.(*unified2.EventRecord).ImpactFlag
		event["impact"] = f.U2Record.(*unified2.EventRecord).Impact
		event["blocked"] = f.U2Record.(*unified2.EventRecord).Blocked
		event["mpls_label"] = f.U2Record.(*unified2.EventRecord).MplsLabel
		event["vlan_id"] = f.U2Record.(*unified2.EventRecord).VlanId

	case *unified2.PacketRecord:
		event["type"] = "packet" // set document type to match unified2 record type
		event["record_type"] = "packet"
		// must assert ".(*unified2.PacketRecord)." coz record is an interface{}
		event["sensor_id"] = f.U2Record.(*unified2.PacketRecord).SensorId
		event["event_id"] = f.U2Record.(*unified2.PacketRecord).EventId
		event["event_second"] = f.U2Record.(*unified2.PacketRecord).EventSecond
		ps = f.U2Record.(*unified2.PacketRecord).PacketSecond
		event["packet_second"] = ps
		pms = f.U2Record.(*unified2.PacketRecord).PacketMicrosecond
		event["packet_microsecond"] = pms
		ut = time.Unix(int64(ps), int64(pms)*1000) // nanosecs = microsecs * 1000
		event["@timestamp"] = common.Time(ut)

		event["packet_link_type"] = f.U2Record.(*unified2.PacketRecord).LinkType
		event["packet_length"] = f.U2Record.(*unified2.PacketRecord).Length
		// how to make packet data readable ?
		// this has unicode/utf8/whatever in it:
		//  event["packet_data"] = string(f.U2Record.(*unified2.PacketRecord).Data)
		// this removes the unicode/utf8/whatever but still has unreadable characters:
		v := make([]rune, 0, len(f.U2Record.(*unified2.PacketRecord).Data))
		for _, r := range f.U2Record.(*unified2.PacketRecord).Data {
			if r > minASCII && r < unicode.MaxASCII && unicode.IsPrint(rune(r)) {
				v = append(v, rune(r))
			}
		}
		event["packet_data"] = fmt.Sprintf("%v", string(v))

		// maybe: create a copy of packet data as base64 ???
		// event["packet_data_base64"] = f.U2Record.(*unified2.PacketRecord).Data
		event["packet_data_hex"] = fmt.Sprintf("% x", f.U2Record.(*unified2.PacketRecord).Data)

		// re-create the packet based on the raw bytes of the "Data" from the "unified2.PacketRecord"
		// big assumption: we expect to see the usual ethernet+ip+tcp stuff
		aPacket :=
			gopacket.NewPacket(
				f.U2Record.(*unified2.PacketRecord).Data,
				layers.LayerTypeEthernet, // firstLayerDecoder
				gopacket.Default,         // options
			)
		// decode aPacket as if it was read from a pcap file, e.g. "tcpdump -s 1514 icmp -w test.pcap"
		gatherPacketLayersInfo(event, aPacket)

	case *unified2.ExtraDataRecord:
		event["type"] = "extradata" // set document type to match unified2 record type
		event["record_type"] = "extradata"
		// must assert ".(*unified2.ExtraDataRecord)." coz record is an interface{}
		event["sensor_id"] = f.U2Record.(*unified2.ExtraDataRecord).SensorId
		event["event_id"] = f.U2Record.(*unified2.ExtraDataRecord).EventId
		es = f.U2Record.(*unified2.ExtraDataRecord).EventSecond
		event["event_second"] = es
		ut = time.Unix(int64(es), 0)
		event["@timestamp"] = common.Time(ut)

		event["event_type"] = f.U2Record.(*unified2.ExtraDataRecord).EventType
		event["event_length"] = f.U2Record.(*unified2.ExtraDataRecord).EventLength
		event["extradata_type"] = f.U2Record.(*unified2.ExtraDataRecord).Type
		event["extradata_data_type"] = f.U2Record.(*unified2.ExtraDataRecord).DataType
		event["extradata_data_length"] = f.U2Record.(*unified2.ExtraDataRecord).DataLength
		event["extradata_data"] = f.U2Record.(*unified2.ExtraDataRecord).Data
	}

	// add any "optional additional fields" from unifiedbeat.yml:
	if f.Fields != nil {
		if f.fieldsUnderRoot {
			for key, value := range *f.Fields {
				// in case of conflicts, overwrite
				_, found := event[key]
				if found {
					logp.Warn("Overwriting %s key", key)
				}
				event[key] = value
			}
		} else {
			event["fields"] = f.Fields
		}
	}

	return event
}

func gatherPacketLayersInfo(event common.MapStr, packet gopacket.Packet) {
	// see https://godoc.org/github.com/google/gopacket#hdr-Pointers_To_Known_Layers
	//   Pointers To Known Layers:
	//   During decoding, certain layers are stored in the packet as well-known layer types.
	//   For example, IPv4 and IPv6 are both considered NetworkLayer layers,
	//   while TCP and UDP are both TransportLayer layers.
	//   We support 4 layers, corresponding to the 4 layers of the TCP/IP layering scheme
	//   (roughly anagalous to layers 2, 3, 4, and 7 of the OSI model).
	//   To access these, you can use the:
	//     packet.LinkLayer,
	//     packet.NetworkLayer,
	//     packet.TransportLayer, and
	//     packet.ApplicationLayer functions.
	//   Each of these functions returns a corresponding interface (gopacket.{Link,Network,Transport,Application}Layer).
	//   The first three provide methods for getting src/dst addresses for that particular layer,
	//   while the final layer provides a Payload function to get payload data.

	// fmt.Printf("packet.Data=%v=%T=%s\n", len(packet.Data()), packet.Data(), packet.Data())
	// fmt.Printf("packet.Dump=%T...:\n%s\n...\n", packet.Dump(), packet.Dump())

	// use "packet.Dump()" as a fail-safe to capture all available layers for a packet,
	// just in case we encounter something unexpected in the unified2 file or we
	// have neglected to handle a particular layer explicitly
	event["packet_dump"] = packet.Dump()
	// "packet.Dump()" is very verbose, i.e. a large amount of text, but that's ok

	// capture the name of the layers found
	var packet_layers []string
	for _, layer := range packet.Layers() {
		packet_layers = append(packet_layers, fmt.Sprintf("%v", layer.LayerType()))
	}
	event["packet_layers"] = packet_layers

	// Ethernet layer?
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethernetLayer != nil {
		ethernetPacket, _ := ethernetLayer.(*layers.Ethernet)
		event["ethernet_src_mac"] = fmt.Sprintf("%v", ethernetPacket.SrcMAC)
		event["ethernet_dst_mac"] = fmt.Sprintf("%v", ethernetPacket.DstMAC)
		// ethernet type is typically IPv4 but could be ARP or other
		event["ethernet_type"] = fmt.Sprintf("%v", ethernetPacket.EthernetType)
		// Length is only set if a length field exists within this header.  Ethernet
		// headers follow two different standards, one that uses an EthernetType, the
		// other which defines a length the follows with a LLC header (802.3).  If the
		// former is the case, we set EthernetType and Length stays 0.  In the latter
		// case, we set Length and EthernetType = EthernetTypeLLC.
		event["ethernet_length"] = fmt.Sprintf("%v", ethernetPacket.Length)
	}

	// IPv4 layer?
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		event["ip_version"] = ip.Version
		event["ip_ihl"] = ip.IHL
		event["ip_tos"] = ip.TOS
		event["ip_length"] = ip.Length
		event["ip_id"] = ip.Id
		event["ip_flags"] = ip.Flags
		event["ip_fragoffset"] = ip.FragOffset
		event["ip_ttl"] = ip.TTL
		event["ip_protocol"] = ip.Protocol
		event["ip_checksum"] = ip.Checksum
		event["ip_src_ip"] = ip.SrcIP
		event["ip_dst_ip"] = ip.DstIP
		event["ip_options"] = ip.Options // maybe? fmt.Sprintf("%v", ip.Options)
		event["ip_padding"] = ip.Padding
	}

	// IPv6 layer?
	ip6Layer := packet.Layer(layers.LayerTypeIPv6)
	if ip6Layer != nil {
		ip6, _ := ip6Layer.(*layers.IPv6)
		event["ip6_version"] = ip6.Version
		event["ip6_trafficclass"] = ip6.TrafficClass
		event["ip6_flowlabel"] = ip6.FlowLabel
		event["ip6_length"] = ip6.Length
		event["ip6_nextheader"] = ip6.NextHeader
		event["ip6_hoplimit"] = ip6.HopLimit
		event["ip6_src_ip"] = ip6.SrcIP
		event["ip6_dst_ip"] = ip6.DstIP
		event["ip6_hopbyhop"] = ip6.HopByHop
	}

	// see: gopacket/layers folder ... what layers are needed for Snort/Suricata alerts?
	// ICMPv4 layer?
	// ICMPv6 layer?
	// ARP layer?

	// UDP layer?
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		event["udp_src_port"] = udp.SrcPort
		event["udp_dst_port"] = udp.DstPort
		event["udp_length"] = udp.Length
		event["udp_checksum"] = udp.Checksum
	}

	// TCP layer?
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		event["tcp_src_port"] = tcp.SrcPort
		event["tcp_dst_port"] = tcp.DstPort
		event["tcp_seq"] = tcp.Seq
		event["tcp_ack"] = tcp.Ack
		event["tcp_data_offset"] = tcp.DataOffset
		event["tcp_fin"] = tcp.FIN
		event["tcp_syn"] = tcp.SYN
		event["tcp_rst"] = tcp.RST
		event["tcp_psh"] = tcp.PSH
		event["tcp_ack"] = tcp.ACK
		event["tcp_urg"] = tcp.URG
		event["tcp_ece"] = tcp.ECE
		event["tcp_cwr"] = tcp.CWR
		event["tcp_ns"] = tcp.NS
		event["tcp_window"] = tcp.Window
		event["tcp_checksum"] = tcp.Checksum
		event["tcp_urgent"] = tcp.Urgent
		event["tcp_options"] = tcp.Options // maybe? fmt.Sprintf("%v", tcp.Options)
		event["tcp_padding"] = tcp.Padding
	}

	// note: the Payload layer is the same as this applicationLayer
	// also, we can get payloads for all packets regardless of their underlying data type:
	// application layer? (aka packet payload)
	applicationLayer := packet.ApplicationLayer()
	if applicationLayer != nil {
		event["packet_payload"] = fmt.Sprintf("%s", applicationLayer.Payload())
	}

	// errors?
	if err := packet.ErrorLayer(); err != nil {
		event["packet_error"] = fmt.Sprintf("Packet decoding error: %v", err)
	}
}

func isIP(s string) (ip4 bool, ip6 bool, ips string) {
	ip := net.ParseIP(s)
	if ip.To4() == nil {
		// it's not IPv4, is it IPv6:
		if ip.To16() == nil {
			return false, false, ""
		} else {
			return false, true, ip.String()
		}
	} else {
		return true, false, ip.String()
	}
}
