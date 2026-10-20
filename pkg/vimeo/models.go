package vimeo

type ViewerInfo struct {
	JWT   string `json:"jwt"`
	XSRFT string `json:"xsrft"`
	VUID  string `json:"vuid"`
}

type PlayerConfig struct {
	Video   VideoInfo     `json:"video"`
	Request PlayerRequest `json:"request"`
}

type VideoInfo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type PlayerRequest struct {
	TextTracks []TextTrack `json:"text_tracks"`
}

type TextTrack struct {
	ID         int64  `json:"id"`
	Lang       string `json:"lang"`
	URL        string `json:"url"`
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Provenance string `json:"provenance"`
	Default    bool   `json:"default"`
}

type AlbumVideosResponse struct {
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
	Data     []AlbumVideo `json:"data"`
}

type AlbumVideo struct {
	URI  string `json:"uri"`
	Link string `json:"link"`
	Name string `json:"name"`
}

type AlbumSingleVideoResponse struct {
	Name            string `json:"name"`
	PlayerEmbedURL  string `json:"player_embed_url"`
}
