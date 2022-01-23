package crawler

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

func CrawlManga(url string, latestRelease float32) float32 {
	log.Printf("Syncing %s", url)

	c := colly.NewCollector()
	var latestChapter string
	re := regexp.MustCompile(`[0-9]+`)

	switch true {
	case strings.Contains(url, "mangakakalot.com"):
		c.OnHTML(".chapter-list", func(e *colly.HTMLElement) {
			latestChapter = e.ChildText("div:first-child span:first-child a")
			latestChapter = re.FindString(latestChapter)
		})
		break
	case strings.Contains(url, "readmanganato.com") || strings.Contains(url, "manganato.com"):
		c.OnHTML(".row-content-chapter", func(e *colly.HTMLElement) {
			latestChapter = e.ChildText("li:first-child a")
			latestChapter = re.FindString(latestChapter)
		})
		break
	case strings.Contains(url, "mangabuddy.com"):
		c.OnHTML("#chapter-list", func(e *colly.HTMLElement) {
			latestChapter = e.ChildText("li:first-child a:first-child div:first-child strong")
			latestChapter = re.FindString(latestChapter)
		})
		break
	case strings.Contains(url, "mangaweeaboo.com"):
		c.OnHTML(".version-chap", func(e *colly.HTMLElement) {
			latestChapter = e.ChildText("li:first-child a")
			latestChapter = re.FindString(latestChapter)
		})
	case strings.Contains(url, "toomics.com"):
		c.OnHTML(".list-ep", func(e *colly.HTMLElement) {
			latestChapter = e.ChildText(".normal_ep:last-child a .cell-num span")
			latestChapter = re.FindString(latestChapter)
		})
		break
	default:
		break
	}

	c.Visit(url)

	if latestChapter == "" {
		return latestRelease
	}

	i, err := strconv.ParseFloat(latestChapter, 32)

	if err != nil {
		return latestRelease
	}

	return float32(i)
}
