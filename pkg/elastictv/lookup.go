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
	IMDBID   string
}

func (c LookupCommonParams) hasCredits() bool {
	return len(c.Director) > 0 || len(c.Actor) > 0 || len(c.Other) > 0
}

func (c LookupCommonParams) getCommonTitleQuery() *Query {
	return NewQuery().
		WithIMDbID(c.IMDBID).
		WithTitles(c.hasCredits(), c.Title...).
		WithGenres(c.Genre...).
		WithDirectors(c.Director...).
		WithActors(c.Actor...).
		WithOthers(c.Other...).
		WithCountries(c.Country...)
}

func (c LookupCommonParams) getSearchItems() []SearchTitlesParams {
	items := []SearchTitlesParams{}

	for _, title := range c.Title {
		items = append(items, SearchTitlesParams{
			Attribute: TitleAttribute,
			Query:     title,
		})
	}
	for _, director := range c.Director {
		items = append(items, SearchTitlesParams{
			Attribute: DirectorAttribute,
			Query:     director,
		})
	}
	for _, actor := range c.Actor {
		items = append(items, SearchTitlesParams{
			Attribute: ActorAttribute,
			Query:     actor,
		})
	}

	return items
}

func (estv ElasticTV) lookupTitle(query *Query, searchItems []SearchTitlesParams, minScoreNoSearch, minScore float64) (*Title, float64, error) {
	title := &Title{}
	score, err := estv.getRecordWithScore(query, estv.Index.Title, title)
	if err != nil {
		return nil, 0, fmt.Errorf("error looking for title: %w", err)
	}

	if score > minScoreNoSearch {
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

func (estv ElasticTV) searchTitles(searchTitles []SearchTitlesParams) *multierror.Error {
	var errors *multierror.Error

	for _, item := range searchTitles {
		query := NewQuery().WithSearchItem(item)
		id, err := estv.GetRecordID(query, estv.Index.Search)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("error looking search history: %w", err))
			continue
		}
		if id != "" {
			continue
		}

		var providerErrors *multierror.Error
		for _, provider := range estv.Providers {
			if err := provider.SearchTitles(item); err != nil {
				providerErrors = multierror.Append(errors, err)
			}
		}

		if providerErrors.ErrorOrNil() != nil {
			errors = multierror.Append(errors, providerErrors)
			continue
		}

		if err := estv.IndexSearchTitle(item); err != nil {
			errors = multierror.Append(errors, fmt.Errorf("error adding search item: %w", err))
		}
	}

	if err := estv.RefreshIndices(); err != nil {
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
		WithType(MovieType)

	searchItems := []SearchTitlesParams{}
	for _, item := range params.LookupCommonParams.getSearchItems() {
		item.Type = MovieType
		item.Year = params.Year
		searchItems = append(searchItems, item)
	}

	return estv.lookupTitle(
		query,
		searchItems,
		viper.GetFloat64("elastictv.movie.min_score_no_search"),
		viper.GetFloat64("elastictv.movie.min_score"),
	)
}

type LookupEpisodeParams struct {
	LookupCommonParams
	EpisodeTitle string
	Season       uint16
	Episode      uint16
}

func (estv ElasticTV) LookupEpisode(params LookupEpisodeParams) (*Title, *Episode, float64, error) {
	query := params.LookupCommonParams.getCommonTitleQuery().
		WithType(TVShowType)

	searchItems := []SearchTitlesParams{}
	for _, item := range params.LookupCommonParams.getSearchItems() {
		item.Type = TVShowType
		searchItems = append(searchItems, item)
	}

	minScore := viper.GetFloat64("elastictv.tvshow.min_score_credits")
	if !params.LookupCommonParams.hasCredits() {
		minScore = viper.GetFloat64("elastictv.tvshow.min_score_no_credits")
	}

	tvshow, score, err := estv.lookupTitle(
		query,
		searchItems,
		viper.GetFloat64("elastictv.movie.min_score_no_search"),
		minScore,
	)
	if err != nil {
		return nil, nil, 0, err
	}

	episode, err := estv.lookupEpisodeDetails(tvshow, params)
	return tvshow, episode, score, err
}

func (estv ElasticTV) lookupEpisodeDetails(tvshow *Title, params LookupEpisodeParams) (*Episode, error) {
	query := NewQuery().
		WithTVShowID(tvshow.IDs.TMDb).
		WithSeasonNumber(params.Season).
		WithEpisodeNumber(params.Episode).
		WithTitles(true, params.EpisodeTitle)

	episode := Episode{}
	score, err := estv.getRecordWithScore(query, estv.Index.Episode, &episode)
	if err != nil {
		return nil, fmt.Errorf("error querying for episode S%02dE%02d for tvshow [ %s ]: %w",
			params.Season, params.Episode, tvshow.Title, err)
	}
	if score > 0 {
		return &episode, nil
	}

	var errors *multierror.Error
	for _, provider := range estv.Providers {
		if err := provider.SearchEpisodes(tvshow, params.Season, params.Episode); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	if err := estv.RefreshIndices(); err != nil {
		errors = multierror.Append(errors, err)
	}

	score, err = estv.getRecordWithScore(query, estv.Index.Episode, &episode)
	if err != nil {
		errors = multierror.Append(errors,
			fmt.Errorf("error querying for episode S%02dE%02d for tvshow [ %s ]: %w",
				params.Season, params.Episode, tvshow.Title, err))
		return nil, errors
	}
	if score > 0 {
		return &episode, nil
	}

	// Episode not found after search at providers, index an empty record to prevent
	// search for the same episode again
	episode = Episode{
		TVShowID: tvshow.IDs.TMDb,
		Season:   params.Season,
		Episode:  params.Episode,
	}

	if err := estv.UpsertEpisode(episode); err != nil {
		errors = multierror.Append(errors,
			fmt.Errorf("error indexing episode S%02dE%02d for tvshow [ %s ]: %w",
				params.Season, params.Episode, tvshow.Title, err))
		return nil, errors
	}

	return nil, multierror.Append(errors,
		fmt.Errorf("episode S%02dE%02d for tvshow [ %s ] not found",
			params.Season, params.Episode, tvshow.Title))
}
