package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

func parseKalenderkuHTML(r io.Reader, year int) ([]Holiday, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
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

	doc.Find("h2").Each(func(_ int, h2 *goquery.Selection) {
		h2Text := strings.TrimSpace(h2.Text())
		if strings.Contains(h2Text, "Daftar Hari Libur") && !strings.Contains(h2Text, "Libur Panjang") {
			container := h2.Closest("section")
			if container.Length() == 0 {
				container = h2.Parent()
			}
			ul := container.Find("ul.columns-1, ul")

			ul.Find("li").Each(func(_ int, li *goquery.Selection) {
				srOnly := strings.TrimSpace(li.Find(".sr-only").Text())
				if srOnly != "" {
					parts := strings.Split(srOnly, " - ")
					if len(parts) >= 2 {
						datePart := strings.TrimSpace(parts[0])
						dateTokens := strings.Fields(datePart)
						if len(dateTokens) >= 2 {
							dayStr := dateTokens[len(dateTokens)-2]
							monthStr := strings.ToLower(dateTokens[len(dateTokens)-1])
							day, errDay := strconv.Atoi(dayStr)
							monthCode, ok := monthMap[monthStr]
							if errDay == nil && ok {
								name := strings.TrimSpace(parts[1])
								holidays = append(holidays, Holiday{
									Date:       fmt.Sprintf("%d-%s-%02d", year, monthCode, day),
									Name:       name,
									IsNational: 1,
								})
								return
							}
						}
					}
				}

				dateText := strings.TrimSpace(li.Find(".text-sm.font-medium").Text())
				name := strings.TrimSpace(li.Find("span.truncate").Text())
				if dateText != "" && name != "" {
					tokens := strings.Fields(dateText)
					if len(tokens) >= 2 {
						day, errDay := strconv.Atoi(tokens[0])
						monthCode, ok := monthMap[strings.ToLower(tokens[1])]
						if errDay == nil && ok {
							holidays = append(holidays, Holiday{
								Date:       fmt.Sprintf("%d-%s-%02d", year, monthCode, day),
								Name:       name,
								IsNational: 1,
							})
						}
					}
				}
			})
		}
	})

	return holidays, nil
}

func scrapeKalenderku(year int) ([]Holiday, error) {
	url := fmt.Sprintf("https://kalenderku.id/%d", year)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	return parseKalenderkuHTML(resp.Body, year)
}

func scrapeTanggalan(year int) ([]Holiday, error) {
	url := fmt.Sprintf("https://www.tanggalan.com/%d", year)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
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
	flag.StringVar(&gkey, "gkey", "", "Google Calendar API Key")
	flag.Parse()

	if gkey == "" || gkey == "." {
		gkey = os.Getenv("GOOGLE_CALENDAR_API_KEY")
	}

	var year int
	if flag.NArg() > 0 {
		year, _ = strconv.Atoi(flag.Arg(0))
	} else {
		year = time.Now().Year()
	}

	var holidays []Holiday
	kalenderkuHolidays, err := scrapeKalenderku(year)
	if err != nil || len(kalenderkuHolidays) == 0 {
		log.Printf("Warning: could not fetch holidays from kalenderku.id: %v, falling back to tanggalan.com", err)
		tHolidays, tErr := scrapeTanggalan(year)
		if tErr != nil {
			log.Printf("Warning: could not fetch holidays from tanggalan.com: %v", tErr)
		} else {
			holidays = tHolidays
		}
	} else {
		holidays = kalenderkuHolidays
	}

	excludedDates := map[string]bool{
		"2027-01-06": true,
		"2027-06-07": true,
	}

	if gkey != "" && gkey != "." {
		googleHolidays, err := fetchGoogleHolidays(year, gkey)
		if err != nil {
			log.Printf("Warning: could not fetch Google holidays: %v", err)
		} else {
			dateMap := make(map[string]bool)
			for _, h := range holidays {
				dateMap[h.Date] = true
			}

			for _, gh := range googleHolidays {
				if !dateMap[gh.Date] && !excludedDates[gh.Date] {
					holidays = append(holidays, gh)
				}
			}
		}
	} else {
		log.Println("Notice: Google Calendar API Key not provided, skipping Google Calendar sync")
	}

	var filtered []Holiday
	for _, h := range holidays {
		if !excludedDates[h.Date] {
			filtered = append(filtered, h)
		}
	}
	holidays = filtered

	if len(holidays) == 0 {
		log.Fatalf("No holidays found for year %d from any source", year)
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
