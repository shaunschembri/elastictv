package elastictv

import (
	"fmt"
	"log"

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

	episode, err := estv.lookupEpisodeDetails(episodeQuery, episodeSearchItem)
	if err != nil {
		log.Printf("failed to lookup episode with IMDB ID [ %s ]. Will use details : %v", params.IMDbID, err)
	}

	// If episode was not found by IMDb ID lookup, try using the
	if episode == nil {
		return estv.lookupEpisodeFromDetails(params)
	}

	tvshow, score, err := estv.lookupTitle(
		NewQuery().WithTMDbID(episode.TVShowIDs.TMDb),
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
	episode := &Episode{}
	score, err := estv.getRecordWithScore(query, estv.Index.Episode, episode)
	if err != nil {
		return nil, fmt.Errorf("error querying for episode [ %s ] : %w", searchItem, err)
	}
	if score > 0 {
		return episode, nil
	}

	alreadySearched, err := estv.alreadySearched(searchItem)
	if err != nil {
		return nil, err
	}
	if alreadySearched {
		return nil, nil
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

	score, err = estv.getRecordWithScore(query, estv.Index.Episode, episode)
	if err != nil {
		return nil, fmt.Errorf("error querying for episode [ %s ] : %w", searchItem, err)
	}
	if score > 0 {
		return episode, nil
	}

	return nil, multierror.Append(errors, fmt.Errorf("episode [ %s ] was not found", searchItem))
}
