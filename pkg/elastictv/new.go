package elastictv

import (
	"fmt"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
)

type ElasticTV struct {
	Client    *elasticsearch.Client
	Providers []SearchableProvider
	Index     index
}

type index struct {
	Title   string
	Episode string
	Search  string
}

func New() (*ElasticTV, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:     viper.GetStringSlice("elastictv.elasticsearch.address"),
		Username:      viper.GetString("elastictv.elasticsearch.username"),
		Password:      viper.GetString("elastictv.elasticsearch.password"),
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to init elastictv: %w", err)
	}

	return &ElasticTV{
		Client:    es,
		Providers: make([]SearchableProvider, 0),
		Index: index{
			Title:   viper.GetString("elastictv.elasticsearch.index.title"),
			Episode: viper.GetString("elastictv.elasticsearch.index.episode"),
			Search:  viper.GetString("elastictv.elasticsearch.index.search"),
		},
	}, nil
}
