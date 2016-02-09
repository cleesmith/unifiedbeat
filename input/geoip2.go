package input

import (
	"net"

	"github.com/elastic/libbeat/logp"
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
