package geoip

import (
	"log/slog"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoInfo contains geolocation data for an IP address
type GeoInfo struct {
	Country   string
	City      string
	Latitude  float64
	Longitude float64
}

// Service provides IP geolocation lookups using MaxMind GeoLite2
type Service struct {
	db   *geoip2.Reader
	mu   sync.RWMutex
	path string
}

// NewService creates a new GeoIP service
// dbPath should point to GeoLite2-City.mmdb file
// If the file doesn't exist, lookups will return empty results
func NewService(dbPath string) *Service {
	svc := &Service{path: dbPath}

	if dbPath != "" {
		db, err := geoip2.Open(dbPath)
		if err != nil {
			slog.Warn("GeoIP database not available, geo lookups disabled",
				slog.String("path", dbPath),
				slog.String("error", err.Error()),
			)
		} else {
			svc.db = db
			slog.Info("GeoIP database loaded", slog.String("path", dbPath))
		}
	}

	return svc
}

// Lookup returns geolocation data for an IP address
// Returns empty GeoInfo if lookup fails or database is not available
func (s *Service) Lookup(ipStr string) GeoInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return GeoInfo{}
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoInfo{}
	}

	record, err := s.db.City(ip)
	if err != nil {
		slog.Debug("GeoIP lookup failed",
			slog.String("ip", ipStr),
			slog.String("error", err.Error()),
		)
		return GeoInfo{}
	}

	// Get city name in English, fallback to first available
	city := ""
	if name, ok := record.City.Names["en"]; ok {
		city = name
	} else if name, ok := record.City.Names["ru"]; ok {
		city = name
	} else {
		for _, name := range record.City.Names {
			city = name
			break
		}
	}

	return GeoInfo{
		Country:   record.Country.IsoCode,
		City:      city,
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}
}

// IsAvailable returns true if the GeoIP database is loaded
func (s *Service) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db != nil
}

// Close closes the GeoIP database
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Reload reloads the GeoIP database from disk
// Useful for updating the database without restart
func (s *Service) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil
	}

	// Close old database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			slog.Warn("failed to close old GeoIP database", slog.String("error", err.Error()))
		}
	}

	// Open new database
	db, err := geoip2.Open(s.path)
	if err != nil {
		s.db = nil
		return err
	}

	s.db = db
	slog.Info("GeoIP database reloaded", slog.String("path", s.path))
	return nil
}
