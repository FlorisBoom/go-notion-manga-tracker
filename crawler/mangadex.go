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

	"github.com/joho/godotenv"
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
			}
			altTitles []struct {
				En string `json:"en"`
			} `json:"alt_titles"`
			Description struct {
				En string `json:"en"`
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
		} `json:"relationships"`
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
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file, err: %s \n", err)
	}

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

	if err != nil || res.StatusCode != 200 {
		if res.StatusCode == 401 {
			authorization()

			// return getAllMangasIds()
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

	if err != nil || res.StatusCode != 200 {
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
		Title:                  mangaResponse.Data.Attributes.Title.En,
		Link:                   "https://mangadex.org/title/" + mangaResponse.Data.ID,
		CurrentProgress:        0,
		SeenLatestRelease:      false,
		ReleaseSchedule:        "",
		LatestReleaseUpdatedAt: mangaResponse.Data.Attributes.UpdatedAt,
		Art:                    fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.512.jpg", mangaId, mangaResponse.Data.Relationships[len(mangaResponse.Data.Relationships)-1].Attributes.FileName),
	}

	switch status {
	case "plan_to_read":
		manga.Status = PlanningToRead
		break
	case "reading":
		manga.Status = Reading
		break
	case "re_reading":
		manga.Status = Reading
		break
	case "completed":
		manga.Status = Completed
		break
	case "on_hold":
		manga.Status = OnHold
		break
	case "dropped":
		manga.Status = Dropped
		break
	default:
		manga.Status = PlanningToRead
		break
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

	if err != nil || res.StatusCode != 200 {
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
		log.Fatalf("Error parsing response body for manga detail, err: %s \n", err)
	}

	i, _ := strconv.ParseFloat(chapterResponse.Data[0].Attributes.Chapter, 32)

	return float32(i)
}

func SyncMangaDex() []Manga {
	authorization()

	idsAndStatusesMap := getAllMangasIds()

	var mangas []Manga

	for id, status := range idsAndStatusesMap {
		manga := getManga(id, fmt.Sprintf("%s", status))
		mangas = append(mangas, manga)
	}

	return mangas
}
