package github

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/0x7461/github-trending/bot"
)

// TrendingSource fetches trending repositories from GitHub.
type TrendingSource struct {
	Period string // "daily", "weekly", "monthly" — defaults to "weekly"
}

func (s *TrendingSource) Fetch() ([]bot.Item, error) {
	period := s.Period
	if period == "" {
		period = "weekly"
	}

	resp, err := http.Get(fmt.Sprintf("https://github.com/trending?since=%s", period))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var items []bot.Item
	doc.Find("article.Box-row").Each(func(i int, sel *goquery.Selection) {
		h2 := sel.Find("h2 a")
		name := strings.Join(strings.Fields(h2.Text()), " ")
		href, _ := h2.Attr("href")

		items = append(items, bot.Item{
			Title:       name,
			URL:         "https://github.com" + href,
			Description: strings.TrimSpace(sel.Find("p.col-9").Text()),
			Meta: map[string]string{
				"language": strings.TrimSpace(sel.Find("span[itemprop='programmingLanguage']").Text()),
				"stars":    strings.TrimSpace(sel.Find("span.d-inline-block.float-sm-right").Text()),
			},
		})
	})

	return items, nil
}
