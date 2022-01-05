package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type AuthResponse struct {
	Result string `json:"result"`
	Token  struct {
		Session string `json:"session"`
		Refresh string `json:"refresh"`
	} `json:"token"`
}

type StatusResponse struct {
	Result   string                 `json:"result"`
	Statuses map[string]interface{} `json:"statuses"`
}

type MangaResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Title struct {
				En string `json:"en"`
				Jp string `json:"jp"`
			}
			Links                  interface{}   `json:"links"`
			OriginalLanguage       string        `json:"originalLanguage"`
			LastVolume             string        `json:"lastVolume"`
			LastChapter            string        `json:"lastChapter"`
			PublicationDemographic string        `json:"publicationDemographic"`
			Status                 string        `json:"status"`
			Year                   int           `json:"year"`
			ContentRating          string        `json:"contentRating"`
			tags                   []interface{} `json:"tags"`
			State                  string        `json:"state"`
			CreatedAt              string        `json:"createdAt"`
			UpdatedAt              string        `json:"updatedAt"`
			Version                int           `json:"version"`
		} `json:"attributes"`
		Relationships []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Description string `json:"description"`
				Volume      string `json:"volume"`
				FileName    string `json:"fileName"`
				CreatedAt   string `json:"createdAt"`
				UpdatedAt   string `json:"updatedAt"`
				Version     int    `json:"version"`
			}
		} `json:"relationships, omitempty"`
	} `json:"data"`
}

type ChapterResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Volume             string `json:"volume"`
			Chapter            string `json:"chapter"`
			Title              string `json:"title"`
			translatedLanguage string `json:"translatedLanguage"`
			hash               string `json:"hash"`
		}
	} `json:"data"`
}

var token string

func authorization() {
	// err := godotenv.Load(".env")

	// if err != nil {
	// 	log.Fatalf("Error loading .env file, err: %s \n", err)
	// }

	body := strings.NewReader(fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}", os.Getenv("MANGADEX_USERNAME"), os.Getenv("MANGADEX_PASSWORD")))
	res, err := http.Post("https://api.mangadex.org/auth/login", "application/json", body)

	if err != nil {
		log.Fatalf("Error authorizing, err: %s \n", err)
	}
	defer res.Body.Close()

	var authResponse AuthResponse

	err = json.NewDecoder(res.Body).Decode(&authResponse)

	if err != nil {
		log.Fatalf("Error parsing response body, err: %s \n", err)
	}

	token = authResponse.Token.Session
}

func getAllMangasIds() map[string]interface{} {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", "https://api.mangadex.org/manga/status", nil)
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)

	if res.StatusCode == 429 {
		authorization()

		time.Sleep(time.Second * 11)
	} else if err != nil || res.StatusCode != 200 {
		if res.StatusCode == 401 {
			authorization()

			return getAllMangasIds()
		} else {
			log.Fatalf("Error retrieving manga statuses from mangadex, err: %s \n", err)
		}
	}
	defer res.Body.Close()

	var statusReponse StatusResponse

	err = json.NewDecoder(res.Body).Decode(&statusReponse)

	if err != nil {
		log.Fatalf("Error parsing response body for https://api.mangadex.org/manga/status, err: %s \n", err)
	}

	m := make(map[string]interface{})

	for key, status := range statusReponse.Statuses {
		m[key] = status
	}

	return m
}

func getManga(mangaId string, status string) Manga {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.mangadex.org/manga/%s?includes[]=cover_art", mangaId), nil)
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)

	if res.StatusCode == 429 {
		authorization()

		time.Sleep(time.Second * 11)
	} else if err != nil || res.StatusCode != 200 {
		if res.StatusCode == 401 {
			authorization()

			return getManga(mangaId, status)
		} else {
			log.Fatalf("Error retrieving manga detail from mangadex, err: %s \n", err)
		}
	}
	defer res.Body.Close()

	var mangaResponse MangaResponse

	err = json.NewDecoder(res.Body).Decode(&mangaResponse)

	if err != nil {
		log.Fatalf("Error parsing response body for manga detail, mangaId: %s err: %s \n", mangaId, err)
	}

	manga := Manga{
		Type:                   "Manga",
		Link:                   "https://mangadex.org/title/" + mangaResponse.Data.ID,
		CurrentProgress:        0,
		SeenLatestRelease:      false,
		ReleaseSchedule:        "",
		LatestReleaseUpdatedAt: mangaResponse.Data.Attributes.UpdatedAt,
	}

	if mangaResponse.Data.Attributes.Title.Jp != "" {
		manga.Title = mangaResponse.Data.Attributes.Title.Jp
	} else if mangaResponse.Data.Attributes.Title.En != "" {
		manga.Title = mangaResponse.Data.Attributes.Title.En
	}

	var coverArt string
	for _, relation := range mangaResponse.Data.Relationships {
		if relation.Type == "cover_art" {
			coverArt = relation.Attributes.FileName
		}
	}

	// Check if cover image exists
	_, err = http.Head(fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", mangaId, coverArt))
	if err != nil {
		manga.Art = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", mangaId, coverArt)
	} else {
		manga.Art = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.512.jpg", mangaId, coverArt)
	}

	var statusses []string
	switch status {
	case "plan_to_read":
		statusses = append(statusses, PlanningToRead)
		manga.Status = statusses
		break
	case "reading":
		statusses = append(statusses, Reading)
		manga.Status = statusses
		break
	case "re_reading":
		statusses = append(statusses, Reading)
		manga.Status = statusses
		break
	case "completed":
		statusses = append(statusses, Completed)
		manga.Status = statusses
		break
	case "on_hold":
		statusses = append(statusses, OnHold)
		manga.Status = statusses
		break
	case "dropped":
		statusses = append(statusses, Dropped)
		manga.Status = statusses
		break
	default:
		statusses = append(statusses, PlanningToRead)
		manga.Status = statusses
		break
	}

	if mangaResponse.Data.Attributes.Status == "completed" {
		statusses = append(statusses, DoneAiring)
	}

	if mangaResponse.Data.Attributes.LastChapter != "" {
		i, _ := strconv.ParseFloat(mangaResponse.Data.Attributes.LastChapter, 32)
		manga.LatestRelease = float32(i)
	} else {
		manga.LatestRelease = getChapterForManga(mangaId)
	}

	return manga
}

func getChapterForManga(mangaId string) float32 {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.mangadex.org/chapter?manga=%s&order[chapter]=desc", mangaId), nil)
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)

	if res.StatusCode == 429 {
		authorization()

		time.Sleep(time.Second * 11)
	} else if err != nil || res.StatusCode != 200 {
		if res.StatusCode == 401 {
			authorization()

			return getChapterForManga(mangaId)
		} else {
			log.Fatalf("Error retrieving manga chapters from mangadex mangaId: %s, err: %s \n", mangaId, err)
		}
	}
	defer res.Body.Close()

	var chapterResponse ChapterResponse

	err = json.NewDecoder(res.Body).Decode(&chapterResponse)

	if err != nil {
		log.Fatalf("Error parsing response body for manga chapters, mangaId: %s err: %s \n", mangaId, err)
	}

	i, _ := strconv.ParseFloat(chapterResponse.Data[0].Attributes.Chapter, 32)

	return float32(i)
}

func SyncMangaDex() []Manga {
	authorization()

	idsAndStatusesMap := getAllMangasIds()

	var mangas []Manga
	batchRequestCount := 60
	loopCount := 0
	i := 0

	for {
		for id, status := range idsAndStatusesMap {
			i++

			manga := getManga(id, fmt.Sprintf("%s", status))
			mangas = append(mangas, manga)
			delete(idsAndStatusesMap, id)

			if i == batchRequestCount {
				loopCount++
				i = 0
				break
			}
		}

		time.Sleep(time.Second * 10)

		if loopCount >= len(idsAndStatusesMap) {
			break
		}
	}

	return mangas
}
