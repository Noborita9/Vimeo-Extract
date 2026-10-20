package vimeo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var videoURIIDRegex = regexp.MustCompile(`/videos/(\d+)`)

type ShowcaseVideo struct {
	ID    string
	Title string
	Link  string
}

func (c *Client) FetchShowcaseVideos(showcaseID string) ([]ShowcaseVideo, error) {
	info, err := c.FetchViewerInfo()
	if err != nil {
		return nil, fmt.Errorf("error al obtener viewer token para showcase: %w", err)
	}

	var allVideos []ShowcaseVideo
	page := 1
	perPage := 50

	for {
		apiURL := fmt.Sprintf("https://api.vimeo.com/albums/%s/videos?page=%d&per_page=%d&fields=uri,link,name", showcaseID, page, perPage)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Authorization", "jwt "+info.JWT)
		req.Header.Set("Accept", "application/json")
		if c.referer != "" {
			req.Header.Set("Referer", c.referer)
		} else {
			req.Header.Set("Referer", fmt.Sprintf("https://vimeo.com/showcase/%s", showcaseID))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error al consultar videos del showcase: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("showcase videos API devolvió status %d: %s", resp.StatusCode, string(body))
		}

		var res AlbumVideosResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("error al parsear JSON del showcase: %w", err)
		}

		for _, item := range res.Data {
			m := videoURIIDRegex.FindStringSubmatch(item.URI)
			if len(m) > 1 {
				allVideos = append(allVideos, ShowcaseVideo{
					ID:    m[1],
					Title: item.Name,
					Link:  item.Link,
				})
			}
		}

		if len(allVideos) >= res.Total || len(res.Data) == 0 {
			break
		}
		page++
	}

	return allVideos, nil
}
