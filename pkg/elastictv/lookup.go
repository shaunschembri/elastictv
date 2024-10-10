package elastictv

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
)

type LookupCommonParams struct {
	Title    []string
	Director []string
	Actor    []string
	Other    []string
	Country  []string
	Genre    []string
	IMDbID   string
}

func (c LookupCommonParams) hasCredits() bool {
	return len(c.Director) > 0 || len(c.Actor) > 0 || len(c.Other) > 0
}

func (c LookupCommonParams) getCommonTitleQuery() *Query {
	return NewQuery().
		WithTitles(c.Title...).
		WithGenres(c.Genre...).
		WithDirectors(c.Director...).
		WithActors(c.Actor...).
		WithOthers(c.Other...).
		WithCountries(c.Country...)
}

func (c LookupCommonParams) getSearchItemsFromDetails(docType Type, year uint16) SearchItems {
	items := make(SearchItems, 0)
	for _, title := range c.Title {
		items = append(items, NewSearchItem(docType, TitleSearchAttribute, title).WithYear(year))
	}

	for _, director := range c.Director {
		items = append(items, NewSearchItem(docType, DirectorSearchAttribute, director).WithYear(year))
	}

	for _, actor := range c.Actor {
		items = append(items, NewSearchItem(docType, ActorSearchAttribute, actor).WithYear(year))
	}

	return items
}

func (estv ElasticTV) lookupTitle(query *Query, searchItems SearchItems, minScoreNoSearch, minScore float64) (*Title, float64, error) {
	title := &Title{}
	score, err := estv.getRecordWithScore(query, estv.Index.Title, title)
	if err != nil {
		return nil, 0, fmt.Errorf("error looking for title: %w", err)
	}

	if score > minScoreNoSearch && !estv.RequiresUpdate(title.Timestamp) {
		return title, score, nil
	}

	errors := estv.searchTitles(searchItems)
	score, err = estv.getRecordWithScore(query, estv.Index.Title, title)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("error looking for title: %w", err))

		return nil, 0, errors
	}

	if score < minScore {
		errors = multierror.Append(errors,
			fmt.Errorf("found title [%s] has too low score %3.1f (min %3.1f)", title.Title, score, minScore))

		return nil, score, errors
	}

	return title, score, errors.ErrorOrNil()
}

func (estv ElasticTV) searchTitles(searchTitles SearchItems) *multierror.Error {
	var errors *multierror.Error

	for _, item := range searchTitles {
		for _, provider := range estv.Providers {
			var err error

			switch item.Type {
			case MovieType:
				err = provider.SearchMovies(item)
			case TvShowType:
				err = provider.SearchTvShows(item)
			default:
				return nil
			}

			if err != nil {
				errors = multierror.Append(errors, err)
			}
		}

		if err := estv.indexSearchItem(item); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	if err := estv.RefreshIndices(estv.Index.Title); err != nil {
		errors = multierror.Append(errors, err)
	}

	return errors
}

type LookupMovieParams struct {
	LookupCommonParams
	Year uint16
}

func (estv ElasticTV) LookupMovie(params LookupMovieParams) (*Title, float64, error) {
	query := params.LookupCommonParams.getCommonTitleQuery().
		WithYearRange(params.Year, 1).
		WithType(MovieType).
		WithIMDbID(params.IMDbID)

	minScore := viper.GetFloat64("elastictv.movie.min_score")
	if params.IMDbID != "" {
		minScore = 0
	}

	return estv.lookupTitle(
		query,
		estv.getSearchItemsForMovieLookup(params),
		viper.GetFloat64("elastictv.movie.min_score_no_search"),
		minScore,
	)
}

func (estv ElasticTV) getSearchItemsForMovieLookup(params LookupMovieParams) SearchItems {
	if params.IMDbID != "" {
		return SearchItems{
			NewSearchItem(MovieType, IMDbIDSearchAttribute, params.IMDbID),
		}
	}

	return params.LookupCommonParams.getSearchItemsFromDetails(MovieType, params.Year)
}
