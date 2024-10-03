package elastictv

import (
	"fmt"
	"strings"
	"time"
)

type SearchAttribute int

var searchAttributesList = [...]string{"title", "director", "actor", "imdb_id", "tmdb_id"}

const (
	TitleSearchAttribute SearchAttribute = iota + 1
	DirectorSearchAttribute
	ActorSearchAttribute
	IMDbIDSearchAttribute
	TMDbIDSearchAttribute
)

func (id SearchAttribute) MarshalText() ([]byte, error) {
	return []byte(searchAttributesList[id-1]), nil
}

func (id SearchAttribute) String() string {
	return searchAttributesList[id-1]
}

type SearchItem struct {
	Query     any             `json:"query,omitempty"`
	Attribute SearchAttribute `json:"attribute,omitempty"`
	Year      uint16          `json:"year,omitempty"`
	SeasonNo  uint16          `json:"season,omitempty"`
	EpisodeNo uint16          `json:"episode,omitempty"`
	Type      Type            `json:"type,omitempty"`
	Timestamp string          `json:"@timestamp,omitempty"`
}

type SearchItems []SearchItem

func NewSearchItem(docType Type, attribute SearchAttribute, query any) SearchItem {
	params := SearchItem{
		Type:      docType,
		Attribute: attribute,
		Query:     query,
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
