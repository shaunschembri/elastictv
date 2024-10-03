package elastictv

import "encoding/json"

const (
	MovieType   = "movie"
	TVShowType  = "tv"
	EpisodeType = "episode"
)

type Title struct {
	Alias       []string    `json:"alias,omitempty"`
	Country     []string    `json:"country,omitempty"`
	Credits     Credits     `json:"credits"`
	Description Description `json:"description"`
	Genre       []string    `json:"genre,omitempty"`
	IDs         IDs         `json:"ids"`
	Image       string      `json:"image,omitempty"`
	Language    string      `json:"language,omitempty"`
	Rating      *Rating     `json:"rating,omitempty"`
	Timestamp   string      `json:"@timestamp"`
	Title       string      `json:"title"`
	Type        string      `json:"type"`
	Year        uint16      `json:"year,omitempty"`
	Tagline     string      `json:"tagline,omitempty"`
}

type Episode struct {
	AirDate     string      `json:"air_date,omitempty"`
	Description Description `json:"description"`
	IDs         IDs         `json:"ids"`
	Image       string      `json:"image,omitempty"`
	Rating      *Rating     `json:"rating,omitempty"`
	Timestamp   string      `json:"@timestamp"`
	Title       string      `json:"title"`
	TVShowIDs   IDs         `json:"tvshow_ids,omitempty"`
	EpisodeNo   uint16      `json:"episode"`
	SeasonNo    uint16      `json:"season"`
}

type Credits struct {
	Actor    []string `json:"actor,omitempty"`
	Director []string `json:"director,omitempty"`
	Other    []string `json:"other,omitempty"`
}

type Description struct {
	Source string `json:"source,omitempty"`
	Text   string `json:"text,omitempty"`
}

type IDs struct {
	IMDb string `json:"imdb,omitempty"`
	TMDb int    `json:"tmdb,omitempty"`
}

type Rating struct {
	Source string  `json:"source,omitempty"`
	Value  float32 `json:"value,omitempty"`
}

type esResult struct {
	Hits struct {
		Hits []struct {
			ID     string          `json:"_id"`
			Score  float64         `json:"_score"`
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
	} `json:"hits"`
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
}
