package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Manga struct {
	ID                     string
	Type                   string
	Title                  string
	Link                   string
	Status                 []string
	CurrentProgress        float32
	LatestRelease          float32
	SeenLatestRelease      bool
	ReleaseSchedule        string
	LatestReleaseUpdatedAt string
	Rating                 float32
	Art                    string
}

type Type struct {
	Select Select `json:"select"`
}

type Select struct {
	Name string `json:"name"`
}

type CurrentProgress struct {
	Number float32 `json:"number"`
}

type Rating struct {
	Number float32 `json:"number"`
}

type Link struct {
	Url string `json:"url"`
}

type MultiSelect struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type Status struct {
	MultiSelect []MultiSelect `json:"multi_select"`
}

type Date struct {
	Start string `json:"start"`
}

type LatestReleaseUpdatedAt struct {
	Date Date `json:"date"`
}

type LatestRelease struct {
	Number float32 `json:"number"`
}

type SeenLatestRelease struct {
	Checkbox bool `json:"checkbox"`
}

type ReleaseSchedule struct {
	MultiSelect []MultiSelect `json:"multi_select"`
}

type Text struct {
	Content string `json:"content"`
}

type Titles struct {
	Text Text `json:"text"`
}

type Title struct {
	Title []Titles `json:"title"`
}

type NotionProperties struct {
	Type                   Type                   `json:"Type"`
	CurrentProgress        CurrentProgress        `json:"Current Progress"`
	Rating                 Rating                 `json:"Rating,omitempty"`
	Link                   Link                   `json:"Link"`
	Status                 Status                 `json:"Status"`
	LatestReleaseUpdatedAt LatestReleaseUpdatedAt `json:"Latest Release Updated At"`
	LatestRelease          LatestRelease          `json:"Latest Release"`
	SeenLatestRelease      SeenLatestRelease      `json:"Seen Latest Release"`
	ReleaseSchedule        *ReleaseSchedule       `json:"Release Schedule,omitempty"`
	Title                  Title                  `json:"Title"`
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
	Archived   bool             `json:"archived"`
	Url        string           `json:"url"`
	Properties NotionProperties `json:"properties"`
}

type NotionPagesResponse struct {
	Object     string                       `json:"object"`
	Results    []NotionPagesResponseResults `json:"results"`
	HasMore    bool                         `json:"has_more"`
	NextCursor string                       `json:"next_cursor"`
}

type NotionUpdateBody struct {
	Properties struct {
		LatestReleaseUpdatedAt LatestReleaseUpdatedAt `json:"Latest Release Updated At"`
		LatestRelease          LatestRelease          `json:"Latest Release"`
		SeenLatestRelease      SeenLatestRelease      `json:"Seen Latest Release"`
		Status                 *Status                `json:"Status,omitempty"`
	} `json:"properties"`
}

type Parent struct {
	DatabaseID string `json:"database_id"`
}

type External struct {
	Url string `json:"url"`
}

type Image struct {
	Type     string   `json:"type"`
	External External `json:"external"`
}

type Children struct {
	Object string `json:"object"`
	Type   string `json:"type"`
	Image  Image  `json:"image"`
}

type NotionCreateBody struct {
	Parent     Parent           `json:"parent"`
	Properties NotionProperties `json:"properties"`
	Children   []Children       `json:"children"`
}

const (
	Dropped         string = "Dropped"
	DoneAiring      string = "Done Airing"
	Completed       string = "Completed"
	PlanningToRead  string = "Planning to Read"
	PlanningToWatch string = "Planning to Watch"
	Watching        string = "Watching"
	Reading         string = "Reading"
	OnHold          string = "On Hold"
)

var notionSecret string
var notionDatabaseId string

func getColorForStatus(status string) string {
	switch status {
	case Dropped:
		return "brown"
	case DoneAiring:
		return "green"
	case Completed:
		return "pink"
	case PlanningToRead:
		return "purple"
	case PlanningToWatch:
		return "purple"
	case Watching:
		return "red"
	case Reading:
		return "red"
	case OnHold:
		return "blue"
	default:
		return ""
	}
}

var loc *time.Location

func Sync() {
	loc, _ = time.LoadLocation("Europe/Amsterdam")
	// err := godotenv.Load(".env")

	// if err != nil {
	// 	log.Printf("Error loading .env file, err: %s \n", err)
	// }

	notionSecret = os.Getenv("NOTION_SECRET")
	notionDatabaseId = os.Getenv("NOTION_DATABASE_ID")

	elapsedTime := time.Since(time.Now())
	log.Println("Starting sync \n")

	syncMangaDexWithNotion()
	syncNotionPagesWithIntegrations()

	log.Printf("Sync completed, time elapsed: %s \n", elapsedTime)
}

func updateNotionPage(pageID string, latestChapter float32, latestReleaseUpdatedAt string, status string) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	notionUpdateBody := NotionUpdateBody{}
	notionUpdateBody.Properties.LatestRelease.Number = latestChapter
	notionUpdateBody.Properties.SeenLatestRelease.Checkbox = false

	if latestReleaseUpdatedAt != "" {
		notionUpdateBody.Properties.LatestReleaseUpdatedAt.Date.Start = latestReleaseUpdatedAt
	} else {
		notionUpdateBody.Properties.LatestReleaseUpdatedAt.Date.Start = time.Now().In(loc).Format("2006-01-02 15:04:05")
	}

	if status != "" {
		multiselect := make([]MultiSelect, 1)
		multiselect[0].Color = getColorForStatus(status)
		multiselect[0].Name = status

		notionUpdateBody.Properties.Status = &Status{
			MultiSelect: multiselect,
		}
	}

	body, _ := json.Marshal(notionUpdateBody)

	req, _ := http.NewRequest("PATCH", "https://api.notion.com/v1/pages/"+pageID, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Bearer "+notionSecret)
	req.Header.Add("Notion-Version", "2021-08-16")
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil || res.StatusCode != 200 {
		log.Printf("Error updating notion page, pageID: %s err: %s , status code: %v \n", pageID, err, res.StatusCode)
	}
	defer res.Body.Close()
}

func createNotionPage(manga Manga) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	statusMultiSelect := make([]MultiSelect, len(manga.Status))

	for key, _ := range manga.Status {
		statusMultiSelect[key].Name = manga.Status[key]
		statusMultiSelect[key].Color = getColorForStatus(manga.Status[key])
	}

	titles := make([]Titles, 1)
	titles[0].Text.Content = manga.Title

	notionCreateBody := &NotionCreateBody{
		Parent: Parent{
			DatabaseID: notionDatabaseId,
		},
		Properties: NotionProperties{
			Type: Type{
				Select: Select{
					Name: manga.Type,
				},
			},
			CurrentProgress: CurrentProgress{
				Number: manga.CurrentProgress,
			},
			Link: Link{
				Url: manga.Link,
			},
			Status: Status{
				MultiSelect: statusMultiSelect,
			},
			LatestReleaseUpdatedAt: LatestReleaseUpdatedAt{
				Date: Date{
					Start: manga.LatestReleaseUpdatedAt,
				},
			},
			LatestRelease: LatestRelease{
				Number: manga.LatestRelease,
			},
			SeenLatestRelease: SeenLatestRelease{
				Checkbox: manga.SeenLatestRelease,
			},
			Title: Title{
				Title: titles,
			},
		},
	}

	notionCreateBody.Children = make([]Children, 1)
	notionCreateBody.Children[0] = Children{
		Object: "block",
		Type:   "image",
		Image: Image{
			Type: "external",
			External: External{
				Url: manga.Art,
			},
		},
	}

	if manga.ReleaseSchedule != "" {
		releaseScheduleMultiSelect := make([]MultiSelect, 1)
		releaseScheduleMultiSelect[0].Name = manga.ReleaseSchedule

		notionCreateBody.Properties.ReleaseSchedule = &ReleaseSchedule{
			MultiSelect: releaseScheduleMultiSelect,
		}
	}

	body, err := json.Marshal(notionCreateBody)

	if err != nil {
		log.Printf("Error creating body for creating new page, mangaId: %s err: %s \n", manga.ID, err)
	}

	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Bearer "+notionSecret)
	req.Header.Add("Notion-Version", "2021-08-16")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)

	if err != nil || res.StatusCode != 200 {
		log.Printf("Error creating notion page, mangaID: %s err: %s \n", manga.ID, err)
	}
	defer res.Body.Close()
}

func getNotionPages(includeMangaDex bool) []Manga {
	client := &http.Client{
		Timeout: time.Second * 30,
	}
	var nextCursor string
	var pages []NotionPagesResponseResults

	for {
		var body *strings.Reader

		if nextCursor != "" && !includeMangaDex {
			body = strings.NewReader(fmt.Sprintf("{\"start_cursor\": \"%s\", \"filter\": {\"property\": \"Link\", \"url\": { \"does_not_contain\": \"mangadex\" }}}", nextCursor))
		} else if includeMangaDex && nextCursor == "" {
			body = strings.NewReader("{\"filter\": {\"property\": \"Link\", \"url\": { \"contains\": \"mangadex\" }}}")
		} else if includeMangaDex && nextCursor != "" {
			body = strings.NewReader(fmt.Sprintf("{\"filter\": {\"property\": \"Link\", \"url\": { \"contains\": \"mangadex\" }}, \"start_cursor\": \"%s\"}", nextCursor))
		} else if !includeMangaDex && nextCursor == "" {
			body = strings.NewReader("{\"filter\": {\"property\": \"Link\", \"url\": { \"does_not_contain\": \"mangadex\" }}}")
		}

		req, _ := http.NewRequest("POST", fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", notionDatabaseId), body)

		req.Header.Add("Authorization", "Bearer "+notionSecret)
		req.Header.Add("Notion-Version", "2021-08-16")
		req.Header.Add("Content-Type", "application/json")
		res, err := client.Do(req)

		if err != nil || res.StatusCode != 200 {
			log.Printf("Error retrieving database pages, err: %s statuscode: %v \n", err, res.StatusCode)
		}

		defer res.Body.Close()

		var notionPagesResponse NotionPagesResponse

		err = json.NewDecoder(res.Body).Decode(&notionPagesResponse)

		if err != nil {
			log.Printf("Error parsing response body, err: %s \n", err)
		}

		nextCursor = notionPagesResponse.NextCursor
		pages = append(pages, notionPagesResponse.Results...)
		if !notionPagesResponse.HasMore {
			break
		}
	}

	var mangas []Manga

	for _, page := range pages {
		manga := Manga{
			ID:                     page.ID,
			Type:                   page.Properties.Type.Select.Name,
			Title:                  page.Properties.Title.Title[0].Text.Content,
			Link:                   page.Properties.Link.Url,
			CurrentProgress:        page.Properties.CurrentProgress.Number,
			LatestRelease:          page.Properties.LatestRelease.Number,
			LatestReleaseUpdatedAt: page.Properties.LatestReleaseUpdatedAt.Date.Start,
			SeenLatestRelease:      page.Properties.SeenLatestRelease.Checkbox,
			ReleaseSchedule:        "",
			Rating:                 page.Properties.Rating.Number,
		}

		if len(page.Properties.ReleaseSchedule.MultiSelect) > 0 {
			manga.ReleaseSchedule = page.Properties.ReleaseSchedule.MultiSelect[0].Name
		}

		var statusses []string
		for _, status := range page.Properties.Status.MultiSelect {
			statusses = append(statusses, status.Name)
		}

		manga.Status = statusses

		mangas = append(mangas, manga)
	}

	return mangas
}

func currentDay() string {
	day := time.Now().In(loc).Weekday()
	return day.String()
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func syncNotionPagesWithIntegrations() {
	mangas := getNotionPages(false)

	if len(mangas) > 0 {
		for _, manga := range mangas {
			if manga.ReleaseSchedule == "" || manga.ReleaseSchedule == currentDay() {
				if !(contains(manga.Status, Completed) || contains(manga.Status, Dropped) || contains(manga.Status, DoneAiring)) {
					if strings.Contains(manga.Link, "pahe.win") || strings.Contains(manga.Link, "animepahe.com") {
						t, _ := time.Parse("2006-01-02T15:04:05.000+03:00", manga.LatestReleaseUpdatedAt)
						if t.Format("2006-01-02") != time.Now().In(loc).Format("2006-01-02") {
							go updateNotionPage(manga.ID, manga.LatestRelease+1, "", "")
						}
					} else {
						latestChapter := CrawlManga(manga.Link, manga.LatestRelease)

						if latestChapter != 0 && latestChapter > manga.LatestRelease {
							go updateNotionPage(manga.ID, latestChapter, "", "")
						}
					}
				}
			}
		}
	}
}

func syncMangaDexWithNotion() {
	notionMangas := getNotionPages(true)
	mangas := SyncMangaDex()

	if len(notionMangas) > 0 && len(mangas) > 0 {
		for _, manga := range mangas {
			if !(contains(manga.Status, Completed) || contains(manga.Status, Dropped) || contains(manga.Status, DoneAiring)) {
				for key, notionManga := range notionMangas {
					// Manga exists in notion and should be updated
					if manga.Link == notionManga.Link || manga.Title == notionManga.Title {
						log.Printf("Syncing %s \n", manga.Link)

						if manga.LatestRelease > notionManga.LatestRelease {
							log.Printf("Updating manga with link: %s \n", manga.Link)

							go updateNotionPage(notionManga.ID, manga.LatestRelease, manga.LatestReleaseUpdatedAt, manga.Status[0])
						}

						break
					} else if key+1 == len(notionMangas) {
						// Manga doesn't exist in notion and should be added
						log.Printf("Creating new notion page for %s \n", manga.Link)

						go createNotionPage(manga)
					}
				}
			}
		}
	} else if len(mangas) > 0 && len(notionMangas) == 0 {
		for _, manga := range mangas {
			log.Printf("Creating new notion page for %s \n", manga.Link)
			go createNotionPage(manga)
		}
	}
}
