package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type webResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

type wikiSearchResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
}

type hnHit struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Points int    `json:"points"`
	ObjectID string `json:"objectID"`
}

type hnResponse struct {
	Hits []hnHit `json:"hits"`
}

type wttrResponse struct {
	CurrentCondition []struct {
				TempC       string `json:"temp_C"`
				FeelsLikeC  string `json:"FeelsLikeC"`
				Humidity    string `json:"humidity"`
				WeatherDesc []struct {
					Value string `json:"value"`
				} `json:"weatherDesc"`
				Winddir16Point string `json:"winddir16Point"`
				WindspeedKmph  string `json:"windspeedKmph"`
			} `json:"current_condition"`
	NearestArea []struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
		Region []struct {
			Value string `json:"value"`
		} `json:"region"`
		Country []struct {
			Value string `json:"value"`
		} `json:"country"`
	} `json:"nearest_area"`
	Weather []struct {
		Date      string `json:"date"`
		AvgtempC  string `json:"avgtempC"`
		MintempC  string `json:"mintempC"`
		MaxtempC  string `json:"maxtempC"`
		Hourly    []struct {
					TempC      string `json:"tempC"`
					FeelsLikeC string `json:"FeelsLikeC"`
					WeatherDesc []struct {
						Value string `json:"value"`
					} `json:"weatherDesc"`
				} `json:"hourly"`
	} `json:"weather"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func NewSearchTool() Tool {
	return Tool{
		Name:        "websearch",
		Description: "Search the web for information. Supports general queries via Wikipedia and Hacker News, weather queries (containing 'weather', 'temperature', 'forecast'), and date/time queries.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query. For weather: location name.",
				},
			},
			"required": []any{"query"},
		},
		Execute: func(args map[string]any) (string, error) {
			q, _ := args["query"].(string)
			if q == "" {
				return "", fmt.Errorf("query can't be empty")
			}

			ql := strings.ToLower(q)

			if isDateQuery(ql) {
				return fetchDate()
			}

			if isWeatherQuery(ql) {
				return fetchWeather(q)
			}

			return searchWeb(q)
		},
	}
}

func isDateQuery(q string) bool {
	triggers := []string{"today", "today's date", "current date", "today's", "what date", "what day", "date today", "current time", "time now"}
	for _, t := range triggers {
		if strings.Contains(q, t) {
			return true
		}
	}
	if q == "date" || q == "time" || q == "now" || q == "today" {
		return true
	}
	return false
}

func fetchDate() (string, error) {
	now := time.Now()
	weekday := now.Weekday().String()
	day := now.Day()
	month := now.Month().String()
	year := now.Year()

	dateStr := fmt.Sprintf("%s, %s %d, %d", weekday, month, day, year)
	timeStr := now.Format("3:04 PM")
	tz, _ := now.Zone()

	result := map[string]any{
		"date": dateStr,
		"time": timeStr,
		"timezone": tz,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return string(out), nil
}

func isWeatherQuery(q string) bool {
	for _, kw := range []string{"weather", "temperature", "forecast", "climate"} {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func searchWeb(query string) (string, error) {
	results := make([]webResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wikiURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&srlimit=5&srprop=snippet",
		url.QueryEscape(query),
	)
	hnURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?query=%s&hitsPerPage=5&tags=story",
		url.QueryEscape(query),
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		body, err := doGet(wikiURL)
		if err != nil {
			return
		}
		var resp wikiSearchResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return
		}
		if len(resp.Query.Search) == 0 {
			return
		}

		mu.Lock()
		topTitles := make([]string, 0, 2)
		for _, r := range resp.Query.Search {
			results = append(results, webResult{
				Title:   r.Title,
				URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(r.Title, " ", "_"))),
				Snippet: stripHTMLTags(r.Snippet),
				Source:  "Wikipedia",
			})
			if len(topTitles) < 2 {
				topTitles = append(topTitles, r.Title)
			}
		}
		mu.Unlock()

		for _, title := range topTitles {
			extractURL := fmt.Sprintf(
				"https://en.wikipedia.org/w/api.php?action=query&prop=extracts&exintro&explaintext&titles=%s&format=json",
				url.QueryEscape(title),
			)
			body2, err2 := doGet(extractURL)
			if err2 != nil {
				continue
			}
			var extractResp struct {
				Query struct {
					Pages map[string]struct {
						Extract string `json:"extract"`
					} `json:"pages"`
				} `json:"query"`
			}
			if err2 := json.Unmarshal(body2, &extractResp); err2 != nil {
				continue
			}
			for _, page := range extractResp.Query.Pages {
				if page.Extract == "" {
					continue
				}
				mu.Lock()
				results = append(results, webResult{
					Title:   title,
					URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(title, " ", "_"))),
					Snippet: page.Extract,
					Source:  "Wikipedia extract",
				})
				mu.Unlock()
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		body, err := doGet(hnURL)
		if err != nil {
			return
		}
		var resp hnResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return
		}
		mu.Lock()
		for _, h := range resp.Hits {
			itemURL := h.URL
			if itemURL == "" {
				itemURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", h.ObjectID)
			}
			snippet := fmt.Sprintf("Points: %d", h.Points)
			results = append(results, webResult{
				Title:   h.Title,
				URL:     itemURL,
				Snippet: snippet,
				Source:  "Hacker News",
			})
		}
		mu.Unlock()
	}()



	wg.Wait()

	if len(results) == 0 {
		return "[]", nil
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	return string(out), nil
}

func fetchWeather(raw string) (string, error) {
	location := cleanLocation(raw)
	apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.QueryEscape(location))
	body, err := doGet(apiURL)
	if err != nil {
		return "", fmt.Errorf("weather request failed: %w", err)
	}

	var w wttrResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return "", fmt.Errorf("parsing weather: %w", err)
	}

	if len(w.CurrentCondition) == 0 {
		return "[]", nil
	}

	cc := w.CurrentCondition[0]
	desc := "unknown"
	if len(cc.WeatherDesc) > 0 {
		desc = cc.WeatherDesc[0].Value
	}

	loc := location
	if len(w.NearestArea) > 0 {
		na := w.NearestArea[0]
		if len(na.AreaName) > 0 {
			loc = na.AreaName[0].Value
		}
		region := ""
		country := ""
		if len(na.Region) > 0 {
			region = na.Region[0].Value
		}
		if len(na.Country) > 0 {
			country = na.Country[0].Value
		}
		if region != "" && country != "" {
			loc = fmt.Sprintf("%s, %s, %s", loc, region, country)
		} else if country != "" {
			loc = fmt.Sprintf("%s, %s", loc, country)
		}
	}

	forecasts := make([]map[string]any, 0)
	for _, day := range w.Weather {
		noonDesc := ""
		if len(day.Hourly) > 4 {
			noonDesc = day.Hourly[4].WeatherDesc[0].Value
		}
		forecasts = append(forecasts, map[string]any{
			"date":      day.Date,
			"condition": noonDesc,
			"min_temp":  fmt.Sprintf("%s°C", day.MintempC),
			"max_temp":  fmt.Sprintf("%s°C", day.MaxtempC),
			"avg_temp":  fmt.Sprintf("%s°C", day.AvgtempC),
		})
	}

	result := map[string]any{
		"location": loc,
		"current": map[string]any{
			"condition":   desc,
			"temperature": fmt.Sprintf("%s°C", cc.TempC),
			"feels_like":  fmt.Sprintf("%s°C", cc.FeelsLikeC),
			"humidity":    fmt.Sprintf("%s%%", cc.Humidity),
			"wind":        fmt.Sprintf("%s %s", cc.Winddir16Point, cc.WindspeedKmph),
		},
		"forecast": forecasts,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return string(out), nil
}

func doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "goop/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func cleanLocation(raw string) string {
	words := strings.Fields(raw)
	stopWords := map[string]bool{
		"weather": true, "temperature": true, "forecast": true, "climate": true,
		"in": true, "at": true, "for": true, "the": true, "what": true,
		"is": true, "s": true, "today": true, "now": true, "current": true,
		"tell": true, "me": true, "can": true, "you": true, "get": true,
		"like": true, "how": true, "tomorrow": true,
	}
	var cleaned []string
	for _, w := range words {
		if !stopWords[w] {
			cleaned = append(cleaned, w)
		}
	}
	if len(cleaned) == 0 {
		return raw
	}
	return strings.Join(cleaned, " ")
}

func stripHTMLTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteRune(r)
		}
	}
	return out.String()
}
