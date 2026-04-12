package geodata

import (
	"compress/gzip"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

//go:embed countries.csv.gz cities.csv.gz
var data embed.FS

type Country struct {
	GeonameID int
	ISO2      string
	ISO3      string
	Name      string
	Phone     string
	NameRu    string
}

type City struct {
	GeonameID   int
	Name        string
	NameRu      string
	CountryCode string
	Admin1Code  string
	Latitude    float64
	Longitude   float64
}

func LoadCountries() ([]Country, error) {
	f, err := data.Open("countries.csv.gz")
	if err != nil {
		return nil, fmt.Errorf("open embedded countries: %w", err)
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	r := csv.NewReader(gr)
	var result []Country

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv: %w", err)
		}
		if len(record) < 6 {
			continue
		}
		geonameID, _ := strconv.Atoi(record[0])
		result = append(result, Country{
			GeonameID: geonameID,
			ISO2:      record[1],
			ISO3:      record[2],
			Name:      record[3],
			Phone:     record[4],
			NameRu:    record[5],
		})
	}
	return result, nil
}

func LoadCities() ([]City, error) {
	f, err := data.Open("cities.csv.gz")
	if err != nil {
		return nil, fmt.Errorf("open embedded cities: %w", err)
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	r := csv.NewReader(gr)
	var result []City

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv: %w", err)
		}
		if len(record) < 7 {
			continue
		}
		geonameID, _ := strconv.Atoi(record[0])
		lat, _ := strconv.ParseFloat(record[5], 64)
		lon, _ := strconv.ParseFloat(record[6], 64)
		result = append(result, City{
			GeonameID:   geonameID,
			Name:        record[1],
			NameRu:      record[2],
			CountryCode: record[3],
			Admin1Code:  record[4],
			Latitude:    lat,
			Longitude:   lon,
		})
	}
	return result, nil
}
