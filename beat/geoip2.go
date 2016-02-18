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
	"net"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/oschwald/geoip2-golang"
)

var GeoIp2Reader *geoip2.Reader

func OpenGeoIp2DB(db string) error {
	var err error
	GeoIp2Reader, err = geoip2.Open(db) // avoid ":=" so no shadowing of GeoIp2Reader variable
	if err != nil {
		logp.Critical("OpenGeoIp2DB: unable to open GeoIP2 database '%v' error: %v!", db, err)
		return err
	}
	return nil
}

func GetLocationByIP(ip string) *geoip2.City {
	if ip == "" {
		return nil
	}
	nip := net.ParseIP(ip) // invalid returns nil
	if nip == nil {
		return nil
	}
	location, err := GeoIp2Reader.City(nip)
	if err != nil {
		return nil
	}
	return location
}
