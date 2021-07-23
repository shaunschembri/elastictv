package elastictv

import (
	"regexp"
)

type Query struct {
	Query struct {
		Bool struct {
			Should []interface{} `json:"should,omitempty"`
			Must   []interface{} `json:"must,omitempty"`
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
	IMDbID    string `json:"ids.imdb,omitempty"`
	Type      string `json:"type,omitempty"`
	Query     string `json:"query,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	TMDbID    int    `json:"ids.tmdb,omitempty"`
	TVShowID  int    `json:"tvshow_id,omitempty"`
	Year      uint16 `json:"year,omitempty"`
	Season    uint16 `json:"season,omitempty"`
	Episode   uint16 `json:"episode,omitempty"`
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

func (q *Query) WithTitles(fuzzyMatch bool, titles ...string) *Query {
	fields := []string{"title", "alias"}
	if !fuzzyMatch {
		fields = []string{"title.keyword", "alias.keyword"}
	}

	for _, title := range titles {
		if title == "" {
			continue
		}

		term := queryModels{
			MultiMatch: &multiMatchQuery{
				Query:  title,
				Fields: fields,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithIMDbID(imdbURL string) *Query {
	re := regexp.MustCompile(`/(tt[0-9]*)`)
	id := re.FindStringSubmatch(imdbURL)
	if len(id) != 2 {
		return q
	}

	term := queryModels{
		Term: &termQuery{
			IMDbID: id[1],
		},
	}
	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithTMDbID(tmdbID int) *Query {
	term := queryModels{
		Term: &termQuery{
			TMDbID: tmdbID,
		},
	}
	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithActors(names ...string) *Query {
	for _, name := range names {
		term := queryModels{
			Match: &matchQuery{
				Actor: name,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithDirectors(names ...string) *Query {
	for _, name := range names {
		term := queryModels{
			Match: &matchQuery{
				Director: name,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithOthers(names ...string) *Query {
	for _, name := range names {
		term := queryModels{
			Match: &matchQuery{
				Other: name,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithCountries(countries ...string) *Query {
	for _, country := range countries {
		term := queryModels{
			Match: &matchQuery{
				Country: country,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithYearRange(year, diff uint16) *Query {
	params := rangeQueryParams{
		GTE: year - diff,
		LTE: year + diff,
	}
	term := queryModels{
		Range: &rangeQuery{
			Year: params,
		},
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithType(itemType string) *Query {
	term := queryModels{
		Term: &termQuery{
			Type: itemType,
		},
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithTVShowID(tvshowID int) *Query {
	term := queryModels{
		Term: &termQuery{
			TVShowID: tvshowID,
		},
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithSeasonNumber(season uint16) *Query {
	if season == 0 {
		return q
	}

	term := queryModels{
		Term: &termQuery{
			Season: season,
		},
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithEpisodeNumber(episode uint16) *Query {
	if episode == 0 {
		return q
	}

	term := queryModels{
		Term: &termQuery{
			Episode: episode,
		},
	}

	q.Query.Bool.Must = append(q.Query.Bool.Must, term)
	return q
}

func (q *Query) WithGenres(genres ...string) *Query {
	for _, genre := range genres {
		term := queryModels{
			Match: &matchQuery{
				Genre: genre,
			},
		}
		q.Query.Bool.Should = append(q.Query.Bool.Should, term)
	}

	return q
}

func (q *Query) WithSearchItem(searchItem SearchTitlesParams) *Query {
	queryTerm := queryModels{
		Term: &termQuery{
			Query: searchItem.Query,
		},
	}
	attributeTerm := queryModels{
		Term: &termQuery{
			Attribute: searchItem.Attribute,
		},
	}
	typeTerm := queryModels{
		Term: &termQuery{
			Type: searchItem.Type,
		},
	}
	q.Query.Bool.Must = append(q.Query.Bool.Must, queryTerm, attributeTerm, typeTerm)

	if searchItem.Year > 1900 {
		yearTerm := queryModels{
			Term: &termQuery{
				Year: searchItem.Year,
			},
		}

		q.Query.Bool.Must = append(q.Query.Bool.Must, yearTerm)
	}

	return q
}
