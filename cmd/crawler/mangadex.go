package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
	Result   string `json:"result"`
	Statuses struct {
		string
	} `json:"statuses"`
}

type ChapterResponse struct {
}

func authorization() string {
	err := godotenv.Load("../../.env")

	if err != nil {
		log.Fatalf("Error loading .env file, err: %s", err)
	}

	body := strings.NewReader(fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}", os.Getenv("NOTION_USERNAME"), os.Getenv("NOTION_PASSWORD")))
	res, err := http.Post("https://api.mangadex.org/auth/login", "application/json", body)

	if err != nil {
		log.Fatalf("Error authorizing, err: %s", err)
	}
	defer res.Body.Close()

	var authResponse AuthResponse

	err = json.NewDecoder(res.Body).Decode(&authResponse)

	if err != nil {
		log.Fatalf("Error parsing response body, err: %s", err)
	}

	return authResponse.Token.Session
}

func getAllMangasIds(token string) {
	client := http.Client{}
	req, _ := http.NewRequest("GET", "https://api.mangadex.org/manga/status", nil)
	req.Header = http.Header{
		"Authorization": []string{"Bearer ", token},
	}
	res, err := client.Do(req)

	if err != nil {
		log.Fatalf("Error retrieving manga statuses from mangadex, err: %s", err)
	}
	defer res.Body.Close()

	var statusReponse StatusResponse

	err = json.NewDecoder(res.Body).Decode(&statusReponse)

	if err != nil {
		log.Fatalf("Error parsing response body, err: %s", err)
	}

	log.Printf("%+v", statusReponse)

	// v := reflect.ValueOf(statusReponse.Statuses)
	// typeOfS := v.Type()

	// for i := 0; i < v.NumField(); i++ {

	// }
}

func getMangaDetail(mangaId string, token string) {

}

func getChapterForManga(token string) {

}

func SyncMangaDex() {
	token := authorization()

	getAllMangasIds(token)
}
