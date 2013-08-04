package parse_test

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/daaku/go.parse"
)

func Example() {
	client := &parse.Client{
		Credentials: &parse.Credentials{
			ApplicationID: "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j",
			RestApiKey:    "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap",
		},
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "api.parse.com",
			Path:   "/1/classes/GameScore/",
		},
	}

	type GameScore struct {
		ID         string     `json:"objectId,omitempty"`
		CreatedAt  *time.Time `json:"createdAt,omitempty"`
		Score      int        `json:"score,omitempty"`
		PlayerName string     `json:"playerName,omitempty"`
		CheatMode  bool       `json:"cheatMode,omitempty"`
	}

	type PostResponse struct {
		ID        string     `json:"objectId,omitempty"`
		CreatedAt *time.Time `json:"createdAt,omitempty"`
	}

	postObject := GameScore{
		Score:      1337,
		PlayerName: "Sean Plott",
		CheatMode:  false,
	}
	var postResponse PostResponse
	_, err := client.Post(nil, &postObject, &postResponse)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(postResponse)

	var getResponse GameScore
	_, err = client.Get(&url.URL{Path: postResponse.ID}, &getResponse)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(getResponse)
}
