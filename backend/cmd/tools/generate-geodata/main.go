package main

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	countriesURL     = "https://download.geonames.org/export/dump/countryInfo.txt"
	citiesURL        = "https://download.geonames.org/export/dump/cities1000.zip"
	alternateNameURL = "https://download.geonames.org/export/dump/alternateNamesV2.zip"

	outputDir = "backend/internal/infrastructure/geodata"
)

type country struct {
	GeonameID int
	ISO2      string
	ISO3      string
	Name      string
	Phone     string
}

type city struct {
	GeonameID   int
	Name        string
	CountryCode string
	Admin1Code  string
	Latitude    float64
	Longitude   float64
}

func main() {
	tmpDir, err := os.MkdirTemp("", "geodata-gen-*")
	if err != nil {
		slog.Error("failed to create temp dir", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Error("failed to remove temp dir", "error", err)
		}
	}()

	slog.Info("downloading countries...")
	countries, err := downloadCountries()
	if err != nil {
		slog.Error("failed to download countries", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed countries", "count", len(countries))

	slog.Info("downloading cities...")
	cities, err := downloadCities(tmpDir)
	if err != nil {
		slog.Error("failed to download cities", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed cities", "count", len(cities))

	slog.Info("downloading Russian translations...")
	ruNames, err := downloadAlternateNames(tmpDir, "ru")
	if err != nil {
		slog.Error("failed to download alternate names", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed Russian translations", "count", len(ruNames))

	slog.Info("writing countries.csv.gz...")
	if err := writeCountries(countries, ruNames); err != nil {
		slog.Error("failed to write countries", "error", err)
		os.Exit(1)
	}

	slog.Info("writing cities.csv.gz...")
	if err := writeCities(cities, ruNames); err != nil {
		slog.Error("failed to write cities", "error", err)
		os.Exit(1)
	}

	slog.Info("done", "output", outputDir)
}

func writeCountries(countries []country, ruNames map[int]string) error {
	f, err := os.Create(filepath.Join(outputDir, "countries.csv.gz"))
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Error("failed to close file", "error", cerr)
		}
	}()

	gw := gzip.NewWriter(f)
	defer func() {
		if cerr := gw.Close(); cerr != nil {
			slog.Error("failed to close gzip writer", "error", cerr)
		}
	}()

	w := csv.NewWriter(gw)
	defer w.Flush()

	for _, c := range countries {
		nameRu := ruNames[c.GeonameID]
		if nameRu == "" {
			nameRu = c.Name
		}
		if err := w.Write([]string{
			strconv.Itoa(c.GeonameID),
			c.ISO2,
			c.ISO3,
			c.Name,
			c.Phone,
			nameRu,
		}); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

func writeCities(cities []city, ruNames map[int]string) error {
	f, err := os.Create(filepath.Join(outputDir, "cities.csv.gz"))
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Error("failed to close file", "error", cerr)
		}
	}()

	gw := gzip.NewWriter(f)
	defer func() {
		if cerr := gw.Close(); cerr != nil {
			slog.Error("failed to close gzip writer", "error", cerr)
		}
	}()

	w := csv.NewWriter(gw)
	defer w.Flush()

	for _, c := range cities {
		nameRu := ruNames[c.GeonameID]
		if nameRu == "" {
			nameRu = c.Name
		}
		if err := w.Write([]string{
			strconv.Itoa(c.GeonameID),
			c.Name,
			nameRu,
			c.CountryCode,
			c.Admin1Code,
			strconv.FormatFloat(c.Latitude, 'f', 6, 64),
			strconv.FormatFloat(c.Longitude, 'f', 6, 64),
		}); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

func downloadCountries() ([]country, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(countriesURL)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Error("failed to close response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result []country
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 17 {
			continue
		}
		geonameID, _ := strconv.Atoi(fields[16])
		if geonameID == 0 {
			continue
		}
		result = append(result, country{
			GeonameID: geonameID,
			ISO2:      fields[0],
			ISO3:      fields[1],
			Name:      fields[4],
			Phone:     fields[12],
		})
	}
	return result, scanner.Err()
}

func downloadCities(tmpDir string) ([]city, error) {
	zipPath := filepath.Join(tmpDir, "cities1000.zip")
	if err := downloadFile(citiesURL, zipPath); err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			slog.Error("failed to close zip reader", "error", cerr)
		}
	}()

	var result []city
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".txt") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open in zip: %w", err)
		}

		scanner := bufio.NewScanner(rc)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), "\t")
			if len(fields) < 19 || fields[6] != "P" {
				continue
			}
			geonameID, _ := strconv.Atoi(fields[0])
			lat, _ := strconv.ParseFloat(fields[4], 64)
			lon, _ := strconv.ParseFloat(fields[5], 64)

			result = append(result, city{
				GeonameID:   geonameID,
				Name:        fields[1],
				CountryCode: fields[8],
				Admin1Code:  fields[10],
				Latitude:    lat,
				Longitude:   lon,
			})
		}
		if cerr := rc.Close(); cerr != nil {
			slog.Error("failed to close zip entry", "error", cerr)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
	}
	return result, nil
}

func downloadAlternateNames(tmpDir string, language string) (map[int]string, error) {
	zipPath := filepath.Join(tmpDir, "alternateNamesV2.zip")
	if err := downloadFile(alternateNameURL, zipPath); err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			slog.Error("failed to close zip reader", "error", cerr)
		}
	}()

	names := make(map[int]string)
	preferred := make(map[int]bool)

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".txt") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open in zip: %w", err)
		}

		scanner := bufio.NewScanner(rc)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), "\t")
			if len(fields) < 4 || fields[2] != language {
				continue
			}
			geonameID, _ := strconv.Atoi(fields[1])
			name := fields[3]
			isPref := len(fields) > 4 && fields[4] == "1"

			if _, exists := names[geonameID]; !exists || (isPref && !preferred[geonameID]) {
				names[geonameID] = name
				preferred[geonameID] = isPref
			}
		}
		if cerr := rc.Close(); cerr != nil {
			slog.Error("failed to close zip entry", "error", cerr)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
	}
	return names, nil
}

func downloadFile(url string, destPath string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Error("failed to close response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Error("failed to close file", "error", cerr)
		}
	}()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	slog.Info("downloaded", "path", destPath, "size_mb", written/1024/1024)
	return nil
}
