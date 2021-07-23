package elastictv

import "fmt"

const (
	TitleAttribute    = "title"
	DirectorAttribute = "director"
	ActorAttribute    = "actor"
)

type SearchableProvider interface {
	Name() string
	Init(estv *ElasticTV) (SearchableProvider, error)
	SearchTitles(SearchTitlesParams) error
	SearchEpisodes(*Title, uint16, uint16) error
}

func (estv *ElasticTV) AddProvider(p SearchableProvider) error {
	provider, err := p.Init(estv)
	if err != nil {
		return fmt.Errorf("failed to init provider %s: %w", p.Name(), err)
	}

	estv.Providers = append(estv.Providers, provider)
	return nil
}

type SearchTitlesParams struct {
	Query     string `json:"query,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Year      uint16 `json:"year,omitempty"`
	Type      string `json:"type,omitempty"`
	Timestamp string `json:"@timestamp,omitempty"`
}

type SearchEpisodesParams struct {
	IDs     IDs
	Episode uint16
	Season  uint16
}
