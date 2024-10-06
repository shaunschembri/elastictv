package elastictv

import (
	"fmt"
	"strings"
)

type Query struct {
	Query struct {
		Bool struct {
			Should []interface{} `json:"should,omitempty"`
			Must   []interface{} `json:"must,omitempty"`
			Filter []interface{} `json:"filter,omitempty"`
		} `json:"bool"`
	} `json:"query"`
}

type queryModels struct {
	MultiMatch *multiMatchQuery `json:"multi_match,omitempty"`
	Match      *matchQuery      `json:"match,omitempty"`
	Term       *termQuery       `json:"term,omitempty"`
	Range      *rangeQuery      `json:"range,omitempty"`
}

type multiMatchQuery struct {
	Query  string   `json:"query,omitempty"`
	Fields []string `json:"fields,omitempty"`
}

type termQuery struct {
	IMDbID       string          `json:"ids.imdb,omitempty"`
	Type         Type            `json:"type,omitempty"`
	Query        string          `json:"query,omitempty"`
	Attribute    SearchAttribute `json:"attribute,omitempty"`
	TMDbID       int             `json:"ids.tmdb,omitempty"`
	TVShowTMDbID int             `json:"tvshow_ids.tmdb,omitempty"`
	Year         uint16          `json:"year,omitempty"`
	SeasonNo     uint16          `json:"season,omitempty"`
	EpisodeNo    uint16          `json:"episode,omitempty"`
}

type matchQuery struct {
	Director string `json:"credits.director,omitempty"`
	Actor    string `json:"credits.actor,omitempty"`
	Other    string `json:"credits.other,omitempty"`
	Country  string `json:"country,omitempty"`
	Genre    string `json:"genre,omitempty"`
}

type rangeQueryParams struct {
	GTE uint16 `json:"gte"`
	LTE uint16 `json:"lte"`
}

type rangeQuery struct {
	Year rangeQueryParams `json:"year"`
}

func NewQuery() *Query {
	return &Query{}
}

func (q *Query) WithTitles(titles ...string) *Query {
	for _, title := range titles {
		if title == "" {
			continue
		}

		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			MultiMatch: &multiMatchQuery{
				Query:  title,
				Fields: []string{"title.keyword", "alias.keyword"},
			},
		}, queryModels{
			MultiMatch: &multiMatchQuery{
				Query:  title,
				Fields: []string{"title", "alias"},
			},
		})
	}

	return q
}

func (q *Query) WithIMDbID(imdbID string) *Query {
	if !strings.HasPrefix(imdbID, "tt") {
		return q
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Term: &termQuery{
			IMDbID: imdbID,
		},
	})

	return q
}

func (q *Query) WithTMDbID(tmdbID int) *Query {
	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Term: &termQuery{
			TMDbID: tmdbID,
		},
	})

	return q
}

func (q *Query) WithActors(names ...string) *Query {
	for _, name := range names {
		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			Match: &matchQuery{
				Actor: name,
			},
		})
	}

	return q
}

func (q *Query) WithDirectors(names ...string) *Query {
	for _, name := range names {
		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			Match: &matchQuery{
				Director: name,
			},
		})
	}

	return q
}

func (q *Query) WithOthers(names ...string) *Query {
	for _, name := range names {
		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			Match: &matchQuery{
				Other: name,
			},
		})
	}

	return q
}

func (q *Query) WithCountries(countries ...string) *Query {
	for _, country := range countries {
		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			Match: &matchQuery{
				Country: country,
			},
		})
	}

	return q
}

func (q *Query) WithYearRange(year, diff uint16) *Query {
	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Range: &rangeQuery{
			Year: rangeQueryParams{
				GTE: year - diff,
				LTE: year + diff,
			},
		},
	})

	return q
}

func (q *Query) WithType(docType Type) *Query {
	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Term: &termQuery{
			Type: docType,
		},
	})

	return q
}

func (q *Query) WithTVShowTMDbID(tmdbTvShowID int) *Query {
	q.Query.Bool.Filter = append(q.Query.Bool.Filter, queryModels{
		Term: &termQuery{
			TVShowTMDbID: tmdbTvShowID,
		},
	})

	return q
}

func (q *Query) WithSeasonNumber(season uint16) *Query {
	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Term: &termQuery{
			SeasonNo: season,
		},
	})

	return q
}

func (q *Query) WithEpisodeNumber(episode uint16) *Query {
	if episode == 0 {
		return q
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
		Term: &termQuery{
			EpisodeNo: episode,
		},
	})

	return q
}

func (q *Query) WithGenres(genres ...string) *Query {
	for _, genre := range genres {
		q.Query.Bool.Should = append(q.Query.Bool.Should, queryModels{
			Match: &matchQuery{
				Genre: genre,
			},
		})
	}

	return q
}

func (q *Query) WithSearchItem(searchItem SearchItem) *Query {
	q.Query.Bool.Must = append(q.Query.Bool.Must,
		queryModels{
			Term: &termQuery{
				Query: fmt.Sprintf("%v", searchItem.Query),
			},
		},
		queryModels{
			Term: &termQuery{
				Attribute: searchItem.Attribute,
			},
		},
		queryModels{
			Term: &termQuery{
				Type: searchItem.Type,
			},
		},
	)

	if searchItem.Year > 1900 {
		q.Query.Bool.Must = append(q.Query.Bool.Must, queryModels{
			Term: &termQuery{
				Year: searchItem.Year,
			},
		})
	}

	if searchItem.SeasonNo > 0 && searchItem.EpisodeNo > 0 {
		q.Query.Bool.Must = append(q.Query.Bool.Must,
			queryModels{
				Term: &termQuery{
					SeasonNo: searchItem.SeasonNo,
				},
			},
			queryModels{
				Term: &termQuery{
					EpisodeNo: searchItem.EpisodeNo,
				},
			},
		)
	}

	return q
}
