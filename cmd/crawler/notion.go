package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Manga struct {
	ID                     string
	Type                   string
	Title                  string
	Link                   string
	Status                 string
	CurrentProgress        int
	LatestRelease          int
	LatestReleaseLink      string
	SeenLatestRelease      bool
	ReleaseSchedule        string
	LatestReleaseUpdatedAt string
	Rating                 int
}

type NotionPagesResponseResults struct {
	Object       string `json:"object"`
	ID           string `json:"id"`
	CreatedTime  string `json:"created_time"`
	LastEditedAt string `json:"last_edited_at"`
	Parent       struct {
		Type       string `json:"type"`
		DatabaseId string `json:"database_id"`
	} `json:"parent"`
	Archived   bool   `json:"archived"`
	Url        string `json:"url"`
	Properties struct {
		Type struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Select struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			}
		} `json:"Type"`
		CurrentProgress struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Number int    `json:"number"`
		} `json:"Current Progress"`
		Rating struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			// Check the omitempty here might not work
			Number int `json:"number"`
		} `json:"Rating"`
		Link struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Url  string `json:"url"`
		} `json:"Link"`
		Status struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			MultiSelect []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"multi_select"`
		} `json:"Status"`
		LatestRelease struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Number int    `json:"number"`
		} `json:"Latest Release"`
		SeenLatestRelease struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Checkbox bool   `json:"checkbox"`
		} `json:"Seen Latest Release"`
		ReleaseSchedule struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			MultiSelect []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"multi_select"`
		} `json:"Release Schedule"`
		LatestReleaseUpdatedAt struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Date struct {
				Start    string `json:"start"`
				End      string `json:"end"`
				TimeZone string `json:"time_zone"`
			} `json:"date"`
		} `json:"Latest Release Updated At"`
		Title struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Title []struct {
				Type string `json:"type"`
				Text struct {
					Content string `json:"content"`
					Link    string `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Underline     bool   `json:"underline"`
					Strikethrough bool   `json:"strikethrough"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string `json:"plain_text"`
				Href      string `json:"href"`
			} `json:"title"`
		} `json:"Title"`
		Url string `json:"url"`
	} `json:"properties"`
}

type NotionPagesResponse struct {
	Object     string                       `json:"object"`
	Results    []NotionPagesResponseResults `json:"results"`
	HasMore    bool                         `json:"has_more"`
	NextCursor string                       `json:"next_cursor"`
}

const (
	Dropped    string = "Dropped"
	DoneAiring        = "Done Airing"
	Completed         = "Completed"
)

var notionSecret string
var notionDatabaseId string

func Sync() {
	err := godotenv.Load("../../.env")

	if err != nil {
		log.Fatalf("Error loading .env file, err: %s", err)
	}

	notionSecret = os.Getenv("NOTION_SECRET")
	notionDatabaseId = os.Getenv("NOTION_DATABASE_ID")

	elapsedTime := time.Since(time.Now())
	log.Println("Starting sync")

	// go SyncNotionPagesWithIntegrations()
	SyncMangaDexWithNotion()

	log.Printf("Sync completed, time elapsed: %s", elapsedTime)
}

func updateNotionPageLatestRelease(pageID string, latestChapter int) {

}

func getNotionPages(includeMangaDex bool) []Manga {
	client := http.Client{}
	var nextCursor string
	var pages []NotionPagesResponseResults

	for hasMore := true; hasMore; hasMore = (hasMore == false) {
		body := strings.NewReader(fmt.Sprintf("{\"cursor\": \"%s\"}", nextCursor))
		if includeMangaDex {
			body = strings.NewReader(fmt.Sprintf("{\"filter\": {\"property\": \"Link\", \"text\": { \"contains\": \"mangadex\" }}, \"cursor\": \"%s\"}", nextCursor))
		}
		req, _ := http.NewRequest("POST", fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", notionDatabaseId), body)
		req.Header = http.Header{
			"Authorization":  []string{"Bearer ", notionSecret},
			"Notion-Version": []string{time.Now().Format("YYYY-MM-DD")},
			"Content-Type":   []string{"application/json"},
		}
		res, err := client.Do(req)

		if err != nil {
			log.Fatalf("Error retrieving database pages, err: %s", err)
		}
		defer res.Body.Close()

		var notionPagesResponse NotionPagesResponse

		err = json.NewDecoder(res.Body).Decode(&notionPagesResponse)

		if err != nil {
			log.Fatalf("Error parsing response body, err: %s", err)
		}

		hasMore = notionPagesResponse.HasMore
		nextCursor = notionPagesResponse.NextCursor
		pages = append(pages, notionPagesResponse.Results...)
	}

	var mangas []Manga

	for _, page := range pages {
		manga := Manga{
			ID:                     page.ID,
			Type:                   page.Properties.Type.Select.Name,
			Title:                  page.Properties.Title.Title[0].Text.Content,
			Link:                   page.Properties.Link.Url,
			Status:                 page.Properties.Status.MultiSelect[0].Name,
			CurrentProgress:        page.Properties.CurrentProgress.Number,
			LatestRelease:          page.Properties.LatestRelease.Number,
			LatestReleaseUpdatedAt: page.Properties.LatestReleaseUpdatedAt.Date.Start,
			LatestReleaseLink:      "",
			SeenLatestRelease:      page.Properties.SeenLatestRelease.Checkbox,
			ReleaseSchedule:        page.Properties.ReleaseSchedule.MultiSelect[0].Name,
			Rating:                 page.Properties.Rating.Number,
		}

		mangas = append(mangas, manga)
	}

	return mangas
}

func currentDay() string {
	day := time.Now().Day()

	switch day {
	case 1:
		return "Monday"
	case 2:
		return "Tuesday"
	case 3:
		return "Wednesday"
	case 4:
		return "Thursday"
	case 5:
		return "Friday"
	case 6:
		return "Saturday"
	case 7:
		return "Sunday"
	default:
		return ""
	}
}

func SyncNotionPagesWithIntegrations() {
	mangas := getNotionPages(false)

	if len(mangas) == 0 {
		log.Fatalln("No mangas found (this means something has gone wrong parsing notion to custom model)")
	}

	for _, manga := range mangas {
		if manga.ReleaseSchedule == "" || manga.ReleaseSchedule == currentDay() && manga.Status != Completed || manga.Status == Dropped || manga.Status == DoneAiring {
			if strings.Contains(manga.Link, "pahe.win") || strings.Contains(manga.Link, "animepahe.com") || strings.Contains(manga.Link, "toomics.com") {
				updateNotionPageLatestRelease(manga.ID, manga.LatestRelease+1)
			} else {
				latestChapter := CrawlManga(manga.Link, manga.LatestRelease)

				if latestChapter != 0 || latestChapter != manga.LatestRelease {
					updateNotionPageLatestRelease(manga.ID, latestChapter)
				}
			}
		}
	}
}

func SyncMangaDexWithNotion() {
	mangas := getNotionPages(true)

	if len(mangas) == 0 {
		log.Fatalln("No mangas found (this means something has gone wrong parsing notion to custom model)")
	}

}
