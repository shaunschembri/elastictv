package tmdb

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

func (t TMDb) SearchEpisode(searchItem elastictv.SearchItem) error {
	if searchItem.Attribute == elastictv.IMDbIDSearchAttribute {
		return t.searchEpisodeFromIMDbID(searchItem)
	}

	return t.searchEpisodeFromDetails(searchItem)
}

func (t TMDb) searchEpisodeFromDetails(searchItem elastictv.SearchItem) error {
	if searchItem.Attribute != elastictv.TMDbIDSearchAttribute &&
		searchItem.Type != elastictv.EpisodeType &&
		searchItem.SeasonNo > 0 &&
		searchItem.EpisodeNo > 0 {
		return nil
	}

	tmdbID, ok := searchItem.Query.(int)
	if !ok {
		return fmt.Errorf("%s: cannot convert query item [ %s ] to TMDb ID", t.Name(), searchItem.Query)
	}

	log.Printf("%s: Getting details for episode [ %s ]", t.Name(), searchItem)

	options := t.getDefaultOptions()
	options["append_to_response"] = "external_ids"

	episode, err := t.tmdb.GetTvEpisodeInfo(tmdbID, int(searchItem.SeasonNo), int(searchItem.EpisodeNo), options)
	if err != nil {
		return fmt.Errorf("%s: error getting details for episode [ %s ] : %w", t.Name(), searchItem, err)
	}

	details := elastictv.Episode{
		AirDate: episode.AirDate,
		TVShowIDs: elastictv.IDs{
			TMDb: tmdbID,
		},
		Description: elastictv.Description{
			Text:   episode.Overview,
			Source: t.Name(),
		},
		EpisodeNo: uint16(episode.EpisodeNumber),
		SeasonNo:  uint16(episode.SeasonNumber),
		Image:     t.getImage(episode.StillPath),
		IDs: elastictv.IDs{
			TMDb: episode.ID,
			IMDb: episode.ExternalIDs.ImdbID,
		},
		Rating: t.getRating(episode.VoteAverage),
		Title:  episode.Name,
	}

	if err := t.estv.UpsertEpisode(details); err != nil {
		return fmt.Errorf("%s: error indexing episode [] %s ] : %w", t.Name(), searchItem, err)
	}

	return nil
}

func (t TMDb) searchEpisodeFromIMDbID(searchItem elastictv.SearchItem) error {
	imdbID, ok := searchItem.Query.(string)
	if !ok {
		return fmt.Errorf("%s: cannot convert query item [ %s ] to IMDb ID", t.Name(), searchItem.Query)
	}

	log.Printf("%s: Searching for episode [ %s ]", t.Name(), searchItem)

	findResults, err := t.tmdb.GetFind(imdbID, "imdb_id", nil)
	if err != nil {
		return fmt.Errorf("%s: error searching for episode [ %s ] : %w", t.Name(), searchItem, err)
	}

	var errors *multierror.Error
	for _, episode := range findResults.TvEpisodeResults {
		episodeSearchItem := elastictv.SearchItem{
			Query:     episode.ShowID,
			Attribute: elastictv.TMDbIDSearchAttribute,
			SeasonNo:  uint16(episode.SeasonNumber),
			EpisodeNo: uint16(episode.EpisodeNumber),
			Type:      elastictv.EpisodeType,
		}

		if err := t.searchEpisodeFromDetails(episodeSearchItem); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}
