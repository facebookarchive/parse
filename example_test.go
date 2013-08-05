package parse_test

import (
	"fmt"
	"net/url"
	"os"

	"github.com/daaku/go.parse"
)

func Example() {
	// Clients can be used concurrently by multiple goroutines.
	client := &parse.Client{

		// Credentials will automatically be included in every request.
		Credentials: &parse.Credentials{
			ApplicationID: "spAVcBmdREXEk9IiDwXzlwe0p4pO7t18KFsHyk7j",
			RestApiKey:    "t6ON64DfTrTL4QJC322HpWbhN6fzGYo8cnjVttap",
		},

		// The relative URLs used below are resolved against this base URL.
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "api.parse.com",
			Path:   "/1/classes/GameScore/",
		},
	}

	// Our GameScore Object Type.
	type GameScore struct {
		parse.Object
		Score      int    `json:"score,omitempty"`
		PlayerName string `json:"playerName,omitempty"`
		CheatMode  bool   `json:"cheatMode,omitempty"`
	}

	// Data for a new instance.
	postObject := GameScore{
		Score:      1337,
		PlayerName: "Sean Plott",
		CheatMode:  false,
	}

	// The response from creating the object - will contain the ID.
	var postResponse parse.Object

	// The HTTP response is being ignored, but is available in case you want to
	// rely on the status code/headers alone.
	_, err := client.Post(nil, &postObject, &postResponse)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(postResponse)

	// We fetch the same object again using it's ID.
	var getResponse GameScore
	_, err = client.Get(&url.URL{Path: string(postResponse.ID)}, &getResponse)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(getResponse)
}
