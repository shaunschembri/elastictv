package elastictv

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
)

type LookupEpisodeParams struct {
	LookupCommonParams
	SeasonNo  uint16
	EpisodeNo uint16
}

func (estv ElasticTV) LookupEpisode(params LookupEpisodeParams) (*Title, *Episode, float64, error) {
	if params.IMDbID != "" {
		return estv.lookupEpisodeFromEpisodeIMDbID(params)
	}

	return estv.lookupEpisodeFromDetails(params)
}

func (estv ElasticTV) lookupEpisodeFromEpisodeIMDbID(params LookupEpisodeParams) (*Title, *Episode, float64, error) {
	episodeQuery := NewQuery().WithIMDbID(params.IMDbID)
	episodeSearchItem := NewSearchItem(EpisodeType, IMDbIDSearchAttribute, params.IMDbID)

	episode, _ := estv.lookupEpisodeDetails(episodeQuery, episodeSearchItem)
	// If episode was not found by IMDb ID lookup, lookup using details
	if episode == nil {
		return estv.lookupEpisodeFromDetails(params)
	}

	tvshow, score, err := estv.lookupTitle(
		NewQuery().WithTMDbID(episode.TVShowIDs.TMDb).WithType(TvShowType),
		SearchItems{NewSearchItem(TvShowType, TMDbIDSearchAttribute, episode.TVShowIDs.TMDb)},
		0, 0,
	)
	if err != nil {
		return nil, nil, score, err
	}

	return tvshow, episode, score, err
}

func (estv ElasticTV) lookupEpisodeFromDetails(params LookupEpisodeParams) (*Title, *Episode, float64, error) {
	query := params.LookupCommonParams.getCommonTitleQuery().
		WithType(TvShowType)

	minScore := viper.GetFloat64("elastictv.tvshow.min_score_credits")
	if !params.LookupCommonParams.hasCredits() {
		minScore = viper.GetFloat64("elastictv.tvshow.min_score_no_credits")
	}

	tvshow, score, err := estv.lookupTitle(
		query,
		params.LookupCommonParams.getSearchItemsFromDetails(TvShowType, 0),
		viper.GetFloat64("elastictv.movie.min_score_no_search"),
		minScore,
	)
	if err != nil {
		return nil, nil, 0, err
	}

	if params.SeasonNo == 0 || params.EpisodeNo == 0 {
		return tvshow, nil, score, err
	}

	query = NewQuery().WithTVShowTMDbID(tvshow.IDs.TMDb).
		WithSeasonNumber(params.SeasonNo).
		WithEpisodeNumber(params.EpisodeNo)

	searchParams := SearchItem{
		Attribute: TMDbIDSearchAttribute,
		Query:     tvshow.IDs.TMDb,
		SeasonNo:  params.SeasonNo,
		EpisodeNo: params.EpisodeNo,
		Type:      EpisodeType,
	}

	episode, err := estv.lookupEpisodeDetails(query, searchParams)

	return tvshow, episode, score, err
}

func (estv ElasticTV) lookupEpisodeDetails(query *Query, searchItem SearchItem) (*Episode, error) {
	episode, err := estv.getEpisode(query, searchItem)
	if episode != nil {
		return episode, nil
	}

	if !estv.RecordExpired(NewQuery().WithSearchItem(searchItem), estv.Index.Search) {
		return nil, err
	}

	var errors *multierror.Error
	for _, provider := range estv.Providers {
		if err := provider.SearchEpisode(searchItem); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	if err := estv.RefreshIndices(estv.Index.Episode); err != nil {
		errors = multierror.Append(errors, err)
	}

	if err := estv.indexSearchItem(searchItem); err != nil {
		errors = multierror.Append(errors, err)
	}

	episode, err = estv.getEpisode(query, searchItem)

	return episode, multierror.Append(errors, err).ErrorOrNil()
}

func (estv ElasticTV) getEpisode(query *Query, searchItem SearchItem) (*Episode, error) {
	episode := &Episode{}
	score, err := estv.getRecordWithScore(query, estv.Index.Episode, episode)
	if err != nil {
		return nil, fmt.Errorf("error querying for episode [ %s ] : %w", searchItem, err)
	}

	if score > 0 {
		return episode, nil
	}

	return nil, fmt.Errorf("episode not found [ %s ]", searchItem)
}
