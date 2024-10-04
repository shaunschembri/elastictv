package elastictv

import (
	"fmt"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
)

const defaultUpdateAfterDays = 30

type ElasticTV struct {
	Client      *elasticsearch.Client
	Providers   []SearchableProvider
	UpdateAfter time.Time
	Index       index
}

type index struct {
	Title   string
	Episode string
	Search  string
}

func New() (*ElasticTV, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:     viper.GetStringSlice("elastictv.elasticsearch.address"),
		Username:      viper.GetString("elastictv.elasticsearch.username"),
		Password:      viper.GetString("elastictv.elasticsearch.password"),
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to init elastictv: %w", err)
	}

	updateAfterDays := viper.GetInt("elastictv.update_after_days")
	if updateAfterDays == 0 {
		updateAfterDays = defaultUpdateAfterDays
	}

	return &ElasticTV{
		Client:      client,
		Providers:   make([]SearchableProvider, 0),
		UpdateAfter: time.Now().AddDate(0, 0, -updateAfterDays),
		Index: index{
			Title:   viper.GetString("elastictv.elasticsearch.index.title"),
			Episode: viper.GetString("elastictv.elasticsearch.index.episode"),
			Search:  viper.GetString("elastictv.elasticsearch.index.search"),
		},
	}, nil
}
