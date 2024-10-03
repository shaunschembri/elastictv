package elastictv

import (
	"fmt"
)

type SearchableProvider interface {
	Name() string
	Init(estv *ElasticTV) (SearchableProvider, error)
	SearchMovies(SearchItem) error
	SearchTvShows(SearchItem) error
	SearchEpisode(SearchItem) error
}

func (estv *ElasticTV) AddProvider(p SearchableProvider) error {
	provider, err := p.Init(estv)
	if err != nil {
		return fmt.Errorf("failed to init provider %s: %w", p.Name(), err)
	}

	estv.Providers = append(estv.Providers, provider)

	return nil
}
