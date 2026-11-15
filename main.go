package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const mapaURL = "https://transonline.salvador.ba.gov.br/geral_sistema/web/index.php/mapa/mapa-porticos"

type Radar struct {
	Nome       string  `json:"nome"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Velocidade int     `json:"velocidade"`
}

var (
	markerRE = regexp.MustCompile(`(?s)google\.maps\.Marker\(\{.*?LatLng\((-?\d+\.\d+),\s*(-?\d+\.\d+)\).*?setContent\('((?:\\'|[^'])*)'\)`)
	speedRE  = regexp.MustCompile(`Velocidade:\s*(\d+)\s*Km`)
)

func main() {
	req, err := http.NewRequest(http.MethodGet, mapaURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "go-radartron/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	radares := parseRadares(string(body))
	out, err := json.MarshalIndent(radares, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(out))
}

func parseRadares(html string) []Radar {
	matches := markerRE.FindAllStringSubmatch(html, -1)
	radares := make([]Radar, 0, len(matches))
	for _, m := range matches {

		lat, errLat := strconv.ParseFloat(m[1], 64)
		lng, errLng := strconv.ParseFloat(m[2], 64)
		if errLat != nil || errLng != nil {
			continue
		}

		popup := strings.ReplaceAll(m[3], `\'`, "'")
		popup = strings.ReplaceAll(popup, "<br/>", "\n")
		popup = strings.ReplaceAll(popup, "<br>", "\n")
		popup = stripTags(popup)

		nome := firstLine(popup)
		velocidade := 0
		if sm := speedRE.FindStringSubmatch(m[3]); len(sm) == 2 {
			velocidade, _ = strconv.Atoi(sm[1])
		}

		radares = append(radares, Radar{
			Nome:       nome,
			Latitude:   lat,
			Longitude:  lng,
			Velocidade: velocidade,
		})
	}

	return radares
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
