package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/u2u-labs/go-layerg-common/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApiResponse struct {
	Users      []api.User `json:"users"`
	TotalCount int        `json:"total_count"`
	NextCursor string     `json:"next_cursor"`
}

func fetchUsersFromAPI(cursor string) (*ApiResponse, error) {
	apiUrl := "http://localhost:8351/v2/console/account?tombstones=false"
	reqUrl, err := url.Parse(apiUrl)
	if err != nil {
		return nil, err
	}

	query := reqUrl.Query()
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	reqUrl.RawQuery = query.Encode()

	// Create a new HTTP request
	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	encodedServerKey := base64.StdEncoding.EncodeToString([]byte("admin:password"))

	// Add authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodedServerKey))

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Print the response body as a string (for debugging)
	fmt.Println("Response Body:", string(body))

	// Unmarshal into a map to handle specific fields
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Process the users array
	users := data["users"].([]interface{})
	for _, user := range users {
		userMap := user.(map[string]interface{})
		if createTime, ok := userMap["create_time"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, createTime); err == nil {
				userMap["create_time"] = timestamppb.New(parsedTime)
			} else {
				return nil, err
			}
		}
		if updateTime, ok := userMap["update_time"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, updateTime); err == nil {
				userMap["update_time"] = timestamppb.New(parsedTime)
			} else {
				return nil, err
			}
		}
	}

	// Marshal back to JSON and then unmarshal into the final ApiResponse struct
	finalData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var apiResponse ApiResponse
	if err := json.Unmarshal(finalData, &apiResponse); err != nil {
		return nil, err
	}

	return &apiResponse, nil
}

func insertOrUpdateUser(ctx context.Context, db *sql.DB, user *api.User) error {
	facebookId := sql.NullString{String: user.FacebookId, Valid: user.FacebookId != ""}
	googleId := sql.NullString{String: user.GoogleId, Valid: user.GoogleId != ""}
	gamecenterId := sql.NullString{String: user.GamecenterId, Valid: user.GamecenterId != ""}
	steamId := sql.NullString{String: user.SteamId, Valid: user.SteamId != ""}
	query := `
        INSERT INTO users (
            id, username, display_name, avatar_url, lang_tag, 
            location, timezone, metadata, facebook_id, google_id, 
            gamecenter_id, steam_id, edge_count, create_time, update_time
        ) VALUES (
            $1, $2, $3, $4, $5, 
            $6, $7, $8, $9, $10, 
            $11, $12, $13, $14, $15
        ) ON CONFLICT (id) DO UPDATE SET
            username = EXCLUDED.username,
            display_name = EXCLUDED.display_name,
            avatar_url = EXCLUDED.avatar_url,
            lang_tag = EXCLUDED.lang_tag,
            location = EXCLUDED.location,
            timezone = EXCLUDED.timezone,
            metadata = EXCLUDED.metadata,
            facebook_id = EXCLUDED.facebook_id,
            google_id = EXCLUDED.google_id,
            gamecenter_id = EXCLUDED.gamecenter_id,
            steam_id = EXCLUDED.steam_id,
            edge_count = EXCLUDED.edge_count,
            update_time = EXCLUDED.update_time;`

	_, err := db.ExecContext(ctx, query, user.Id, user.Username, user.DisplayName, user.AvatarUrl,
		user.LangTag, user.Location, user.Timezone, user.Metadata, facebookId,
		googleId, gamecenterId, steamId, user.EdgeCount, user.CreateTime.AsTime(), user.UpdateTime.AsTime())

	return err
}

func SyncUsers(ctx context.Context, db *sql.DB) {
	cursor := ""
	for {
		apiResponse, err := fetchUsersFromAPI(cursor)
		if err != nil {
			fmt.Println("Error fetching users:", err)
			break
		}

		for i := range apiResponse.Users {
			if err := insertOrUpdateUser(ctx, db, &apiResponse.Users[i]); err != nil {
				fmt.Println("Error inserting/updating user:", err)
			}
		}

		// Check if there is a next cursor for pagination
		if apiResponse.NextCursor == "" {
			break
		}
		cursor = apiResponse.NextCursor
	}

	fmt.Println("User sync complete")
}
