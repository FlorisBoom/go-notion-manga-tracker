package crawler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	} `json:"token,omitempty"`
	Errors []struct {
		ID      string `json:"id"`
		Status  int    `json:"status"`
		Title   string `json:"title"`
		Detail  string `json:"detail"`
		Context string `json:"context"`
	} `json:"errors,omitempty"`
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
				En string `json:"en,omitempty"`
				Jp string `json:"jp,omitempty"`
			}
			LastVolume  string `json:"lastVolume"`
			LastChapter string `json:"lastChapter"`
			Status      string `json:"status"`
			CreatedAt   string `json:"createdAt"`
			UpdatedAt   string `json:"updatedAt"`
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
			} `json:"attributes,omitempty"`
		} `json:"relationships,omitempty"`
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
			UpdatedAt          string `json:"updatedAt"`
		}
	} `json:"data"`
}

var token string

func authorization() {
	// err := godotenv.Load(".env")

	// if err != nil {
	// 	log.Printf("Error loading .env file, err: %s \n", err)
	// }

	log.Print("Authorizing... \n")

	body := strings.NewReader(fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}", os.Getenv("MANGADEX_USERNAME"), os.Getenv("MANGADEX_PASSWORD")))
	res, err := http.Post("https://api.mangadex.org/auth/login", "application/json", body)

	if err != nil {
		log.Printf("Error authorizing, err: %s \n", err)
	}
	defer res.Body.Close()

	var authResponse AuthResponse

	err = json.NewDecoder(res.Body).Decode(&authResponse)

	if err != nil {
		log.Printf("Error parsing response body authorization, err: %s \n", err)
	}

	if authResponse.Errors != nil {
		log.Printf("Error getting new auth token, err: %s \n", authResponse.Errors[0].Detail)

		if authResponse.Errors[0].Status == 429 {
			log.Printf("Too many requests, sleeping for 20 minutes")

			time.Sleep(time.Second * 60 * 20)

			authorization()
		}
	}

	token = authResponse.Token.Session
}

func refreshToken() {
	log.Print("Refreshing Token... \n")

	body := strings.NewReader(fmt.Sprintf("{\"token\": \"%s\"}", token))
	res, err := http.Post("https://api.mangadex.org/auth/refresh", "application/json", body)

	if err != nil {
		log.Printf("Error refreshing token, err: %s \n", err)
	}
	defer res.Body.Close()

	var authResponse AuthResponse

	err = json.NewDecoder(res.Body).Decode(&authResponse)

	if err != nil {
		log.Printf("Error parsing response body authorization, err: %s \n", err)
	}

	if authResponse.Errors != nil {
		log.Printf("Error getting new auth token, err: %s \n", authResponse.Errors[0].Detail)

		if authResponse.Errors[0].Status == 429 {
			log.Printf("Too many requests, sleeping for 20 minutes")

			time.Sleep(time.Second * 60 * 20)

			authorization()
		}
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
			refreshToken()

			return getAllMangasIds()
		} else {
			log.Printf("Error retrieving manga statuses from mangadex, err: %s \n", err)
		}
	}
	defer res.Body.Close()

	var statusReponse StatusResponse

	err = json.NewDecoder(res.Body).Decode(&statusReponse)

	if err != nil {
		resBody, _ := ioutil.ReadAll(res.Body)
		fmt.Printf("res %s\n", string(resBody))

		log.Printf("Error parsing response body for https://api.mangadex.org/manga/status, err: %s \n", err)
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
			refreshToken()

			return getManga(mangaId, status)
		} else {
			log.Printf("Error retrieving manga detail from mangadex, err: %s \n", err)
		}
	}

	defer res.Body.Close()

	var mangaResponse MangaResponse

	err = json.NewDecoder(res.Body).Decode(&mangaResponse)

	if err != nil {
		log.Printf("Error parsing response body for manga detail, mangaId: %s err: %s \n", mangaId, err)
	}

	manga := Manga{
		Type:              "Manga",
		Link:              "https://mangadex.org/title/" + mangaResponse.Data.ID,
		CurrentProgress:   0,
		SeenLatestRelease: false,
		ReleaseSchedule:   "",
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

	latestRelease, updatedAt := getChapterForManga(mangaId)
	manga.LatestRelease = latestRelease
	manga.LatestReleaseUpdatedAt = updatedAt

	return manga
}

func getChapterForManga(mangaId string) (float32, string) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.mangadex.org/chapter?manga=%s&order[chapter]=desc&translatedLanguage[]=en", mangaId), nil)
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)

	if err != nil || res.StatusCode != 200 {
		if res.StatusCode == 401 {
			refreshToken()

			return getChapterForManga(mangaId)
		} else {
			log.Printf("Error retrieving manga chapters from mangadex mangaId: %s, err: %s \n", mangaId, err)
		}
	}
	defer res.Body.Close()

	var chapterResponse ChapterResponse

	err = json.NewDecoder(res.Body).Decode(&chapterResponse)

	if err != nil {
		log.Printf("Error parsing response body for manga chapters, mangaId: %s err: %s \n", mangaId, err)
	}

	i, _ := strconv.ParseFloat(chapterResponse.Data[0].Attributes.Chapter, 32)

	return float32(i), chapterResponse.Data[0].Attributes.UpdatedAt
}

func SyncMangaDex() []Manga {
	authorization()

	idsAndStatusesMap := getAllMangasIds()

	var mangas []Manga

	for id, status := range idsAndStatusesMap {
		manga := getManga(id, fmt.Sprintf("%s", status))
		mangas = append(mangas, manga)
		delete(idsAndStatusesMap, id)
		time.Sleep(time.Second * 1)
	}

	return mangas
}
