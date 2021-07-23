package tmdb

import (
	"fmt"
	"log"

	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

func (t TMDb) SearchEpisodes(title *elastictv.Title, seasonNo, episodeNo uint16) error {
	log.Printf("%s: Getting episode details for S%02d for tvshow [ %s ]",
		t.Name(), seasonNo, title.Title)

	season, err := t.tmdb.GetTvSeasonInfo(title.IDs.TMDb, int(seasonNo), t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error getting details for season [%d] for tvshow [%s]: %w",
			t.Name(), seasonNo, title.Title, err)
	}

	for _, episode := range season.Episodes {
		details := elastictv.Episode{
			AirDate:  episode.AirDate,
			TVShowID: title.IDs.TMDb,
			Description: elastictv.Description{
				Text:   episode.Overview,
				Source: t.Name(),
			},
			Episode: uint16(episode.EpisodeNumber),
			Season:  uint16(episode.SeasonNumber),
			Image:   t.getImage(episode.StillPath),
			IDs: elastictv.IDs{
				TMDb: episode.ID,
			},
			Rating: t.getRating(episode.VoteAverage),
			Title:  episode.Name,
		}

		if err := t.estv.UpsertEpisode(details); err != nil {
			return fmt.Errorf("error indexing episode: %w", err)
		}
	}

	return nil
}
