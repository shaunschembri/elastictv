package elastictv

import (
	"fmt"
	"strings"
	"time"
)

const (
	TitleAttribute    = "title"
	DirectorAttribute = "director"
	ActorAttribute    = "actor"
	IMDbIDAttribute   = "imdbID"
	TMDbIDAttribute   = "tmdbID"
)

type SearchItem struct {
	Query     any    `json:"query,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Year      uint16 `json:"year,omitempty"`
	SeasonNo  uint16 `json:"season,omitempty"`
	EpisodeNo uint16 `json:"episode,omitempty"`
	Type      string `json:"type,omitempty"`
	Timestamp string `json:"@timestamp,omitempty"`
}

type SearchItems []SearchItem

func NewSearchItem(titleType, attribute string, query any) SearchItem {
	params := SearchItem{
		Attribute: attribute,
		Query:     query,
		Type:      titleType,
	}

	return params
}

func (s SearchItem) WithYear(year uint16) SearchItem {
	if year > 0 {
		s.Year = year
	}

	return s
}

func (s SearchItem) WithSeasonNo(seasonNo uint16) SearchItem {
	if seasonNo > 0 {
		s.SeasonNo = seasonNo
	}

	return s
}

func (s SearchItem) WithEpisodeNo(episodeNo uint16) SearchItem {
	if episodeNo > 0 {
		s.EpisodeNo = episodeNo
	}

	return s
}

func (s SearchItem) String() string {
	text := fmt.Sprintf("%v (%s %s)", s.Query, s.Attribute, s.Type)

	if s.SeasonNo > 0 && s.EpisodeNo > 0 {
		text += fmt.Sprintf(" S%02dE%02d", s.SeasonNo, s.EpisodeNo)
	}

	return strings.TrimSpace(text)
}

func (estv ElasticTV) alreadySearched(item SearchItem) (bool, error) {
	query := NewQuery().WithSearchItem(item)
	id, err := estv.GetRecordID(query, estv.Index.Search)
	if err != nil {
		return false, fmt.Errorf("failed to fetch search item : %w", err)
	}

	return id != "", nil
}

func (estv ElasticTV) indexSearchItem(item SearchItem) error {
	item.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.0000000")
	if err := estv.index(estv.Index.Search, "", item); err != nil {
		return fmt.Errorf("failed to index search item : %w", err)
	}

	if err := estv.RefreshIndices(estv.Index.Search); err != nil {
		return fmt.Errorf("failed to refresh search index : %w", err)
	}

	return nil
}
