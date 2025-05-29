package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Holiday struct {
	Date       string `json:"date"`
	Name       string `json:"name"`
	IsNational uint8  `json:"is_national"`
}

func scrapeTanggalan(year int) ([]Holiday, error) {
	url := fmt.Sprintf("https://www.tanggalan.com/%d", year)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	monthMap := map[string]string{
		"januari":   "01",
		"februari":  "02",
		"maret":     "03",
		"april":     "04",
		"mei":       "05",
		"juni":      "06",
		"juli":      "07",
		"agustus":   "08",
		"september": "09",
		"oktober":   "10",
		"november":  "11",
		"desember":  "12",
	}

	var holidays []Holiday

	doc.Find("article ul").Each(func(_ int, list *goquery.Selection) {
		monthText := strings.TrimSpace(list.Find("li a").First().Text())
		monthText = strings.ToLower(strings.TrimRight(strings.TrimLeft(monthText, "0123456789"), "0123456789"))
		month := monthMap[monthText]

		list.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
			dateText := strings.TrimSpace(row.Find("td").First().Text())
			description := strings.TrimSpace(row.Find("td").Eq(1).Text())

			if strings.Contains(dateText, "-") {
				dates := strings.Split(dateText, "-")
				start, _ := strconv.Atoi(strings.TrimSpace(dates[0]))
				end, _ := strconv.Atoi(strings.TrimSpace(dates[1]))
				for day := start; day <= end; day++ {
					holidays = append(holidays, Holiday{
						Date:       fmt.Sprintf("%d-%s-%02d", year, month, day),
						Name:       description,
						IsNational: 1,
					})
				}
			} else {
				day, _ := strconv.Atoi(dateText)
				holidays = append(holidays, Holiday{
					Date:       fmt.Sprintf("%d-%s-%02d", year, month, day),
					Name:       description,
					IsNational: 1,
				})
			}
		})
	})

	return holidays, nil
}

func fetchGoogleHolidays(year int, apiKey string) ([]Holiday, error) {
	ctx := context.Background()

	srv, err := calendar.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %v", err)
	}

	timeMin := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	timeMax := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC).Format(time.RFC3339)

	events, err := srv.Events.List("id.indonesian#holiday@group.v.calendar.google.com").
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve holidays: %v", err)
	}

	var holidays []Holiday
	for _, event := range events.Items {
		if event.Start == nil || event.Start.Date == "" {
			continue
		}

		var isNational uint8
		isNational = 0
		if event.Description != "" && strings.Contains(event.Description, "Hari libur nasional") {
			isNational = 1
		}

		holidays = append(holidays, Holiday{
			Date:       event.Start.Date,
			Name:       event.Summary,
			IsNational: isNational,
		})
	}

	return holidays, nil
}

type ByDate []Holiday

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].Date < a[j].Date }

func main() {
	var outDir string
	var gkey string
	flag.StringVar(&outDir, "out-dir", ".", "Output directory for JSON file")
	flag.StringVar(&gkey, "gkey", ".", "Google Calendar API Key")
	flag.Parse()

	if gkey == "" {
		gkey = os.Getenv("GOOGLE_CALENDAR_API_KEY")
		if gkey == "" {
			log.Fatal("API key is required (use --gkey or set GOOGLE_CALENDAR_API_KEY)")
			return
		}
	}

	var year int
	if flag.NArg() > 0 {
		year, _ = strconv.Atoi(flag.Arg(0))
	} else {
		year = time.Now().Year()
	}

	holidays, err := scrapeTanggalan(year)
	if err != nil {
		return
	}

	googleHolidays, err := fetchGoogleHolidays(year, gkey)
	if err != nil {
		log.Printf("Warning: could not fetch Google holidays: %v", err)
	}

	dateMap := make(map[string]bool)
	for _, h := range holidays {
		dateMap[h.Date] = true
	}

	for _, gh := range googleHolidays {
		if !dateMap[gh.Date] {
			holidays = append(holidays, gh)
		}
	}

	outputPath := filepath.Join(outDir, fmt.Sprintf("%d.json", year))
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	sort.Sort(ByDate(holidays))

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(holidays); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Data for year %d has been saved to %s\n", year, outputPath)
}
