package geominder

import (
	"fmt"
	"net"

	"github.com/oschwald/maxminddb-golang"
)

// LookupDB essentially wraps a `maxminddb.Reader` to query for and retrieve our
// minimal data structure. By querying for less, lookups are faster.
//
// Additionally, this allows us to abstract and separate the DB lookup logic from
// the HTTP handlers.
type LookupDB struct {
	reader *maxminddb.Reader
}

// LookupResult is a minimal set of location information that is queried for and
// returned from our lookups.
type LookupResult struct {
	Country  country  `maxminddb:"country" json:"country"`
	Location location `maxminddb:"location" json:"location"`
}

// DEVS: For possible fields, see https://dev.maxmind.com/geoip/geoip2/web-services/
// TODO: maybe make same as https://github.com/bluesmoon/node-geoip?

type country struct {
	// A two-character ISO 3166-1 country code for the country associated with
	// the IP address.
	ISOCode string `maxminddb:"iso_code" json:"iso_code"`
}

type location struct {
	// The approximate latitude of the postal code, city, subdivision or country
	// associated with the IP address.
	Latitude float64 `maxminddb:"latitude" json:"latitude"`
	// The approximate longitude of the postal code, city, subdivision or
	// country associated with the IP address.
	Longitude float64 `maxminddb:"longitude" json:"longitude"`
	// The approximate accuracy radius, in kilometers, around the
	// latitude and longitude for the geographical entity (country,
	// subdivision, city or postal code) associated with the IP address.
	// We have a 67% confidence that the location of the end-user falls
	// within the area defined by the accuracy radius and the latitude
	// and longitude coordinates.
	Accuracy int `maxminddb:"accuracy_radius" json:"accuracy_radius"`
	// The time zone associated with location, as specified by the IANA
	// Time Zone Database, e.g., “America/New_York”.
	// Timezone string `maxminddb:"time_zone"`
}

// NewLookupDB open a new DB reader.
//
// dbPath must be the path to a valid maxmindDB file containing city precision.
func NewLookupDB(dbPath string) (*LookupDB, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &LookupDB{reader: db}, nil
}

// Close closes the underlying database and returns resources to the system.
//
// For current implemetnation, see maxminddb.Reader.Close()
func (l *LookupDB) Close() error {
	return l.reader.Close()
}

// Lookup returns the results for a given IP address, or an error if results can
// not be obtained for some reason.
func (l *LookupDB) Lookup(ip net.IP) (*LookupResult, error) {
	var r LookupResult
	err := l.lookup(ip, &r)
	return &r, err
}

// FastLookup is a version of Lookup() that avoids memory allocations by taking
// a pointer to a pre-allocated LookupResult to decode into.
//
// You probably don't need to use this unless you are tuning for ludicrous speed
// in combination with a sync.Pool, etc.
//
// TODO: benchmark this in more detail to see if saving that one allocation
// really makes a big enough difference, if not consider removal.
func (l *LookupDB) FastLookup(ip net.IP, r *LookupResult) error {
	return l.lookup(ip, r)
}

// oschwald/maxminddb-golang does not generate an error on a failed lookup,
// see: https://github.com/oschwald/maxminddb-golang/issues/41
//
// to work around this, we don't use their Lookup(), but rather check
// LookupOffset() first, and throw our own error if nothing was found, before
// using the offset for a manual Decode().
func (l *LookupDB) lookup(ip net.IP, r *LookupResult) error {
	offset, err := l.reader.LookupOffset(ip)
	if err != nil {
		return err
	}
	if offset == maxminddb.NotFound {
		return fmt.Errorf("no match for %v found in database", ip)
	}
	return l.reader.Decode(offset, r)
}