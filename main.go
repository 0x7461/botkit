package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"

	"github.com/0x7461/github-trending/utils"
)

// fetchTrending scrapes GitHub trending page and returns repos
func fetchTrending() ([]utils.Repo, error) {
	// Fetch the HTML page (weekly trending)
	resp, err := http.Get("https://github.com/trending?since=weekly")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract repos
	var repos []utils.Repo

	// Find each repository article
	doc.Find("article.Box-row").Each(func(i int, s *goquery.Selection) {
		repo := utils.Repo{}

		// Get repo name and URL
		h2 := s.Find("h2 a")
		rawName := h2.Text()

		// Clean up name: Fields splits on whitespace, Join combines with single space
		repo.Name = strings.Join(strings.Fields(rawName), " ")
		href, _ := h2.Attr("href")
		repo.URL = "https://github.com" + href

		// Get description
		repo.Description = strings.TrimSpace(s.Find("p.col-9").Text())

		// Get language
		repo.Language = strings.TrimSpace(s.Find("span[itemprop='programmingLanguage']").Text())

		// Get stars this week
		repo.Stars = strings.TrimSpace(s.Find("span.d-inline-block.float-sm-right").Text())

		repos = append(repos, repo)
	})

	return repos, nil
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found - copy .env.example to .env and fill in your credentials")
	}

	fmt.Println("Fetching GitHub trending repos...")
	repos, err := fetchTrending()
	if err != nil {
		log.Fatalf("Error fetching trending: %v", err)
	}

	fmt.Printf("Found %d trending repos\n\n", len(repos))

	// Send to Telegram if enabled
	enableTelegram := os.Getenv("ENABLE_TELEGRAM") == "true"
	if enableTelegram {
		botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")

		if botToken == "" || chatIDStr == "" {
			log.Println("[WARN] ENABLE_TELEGRAM=true but missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID")
		} else {
			// Convert chatID string to int64
			var chatID int64
			fmt.Sscanf(chatIDStr, "%d", &chatID)

			fmt.Println("[SEND] Sending to Telegram...")
			// Use Telegram-specific formatter
			message := utils.FormatForTelegram(repos)
			err := utils.SendTelegramMessage(botToken, chatID, message)
			if err != nil {
				log.Printf("[ERROR] Failed to send Telegram message: %v\n", err)
			} else {
				fmt.Println("[OK] Message sent to Telegram successfully!")
			}
		}
	} else {
		fmt.Println("[INFO] Telegram disabled (set ENABLE_TELEGRAM=true in .env to enable)")
	}

	fmt.Println("\n[DONE] Finished processing weekly trending repos")
}
