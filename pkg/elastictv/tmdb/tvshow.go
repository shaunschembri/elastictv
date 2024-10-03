package tmdb

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/shaunschembri/go-tmdb"

	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

func (t TMDb) SearchTvShows(params elastictv.SearchItem) error {
	switch params.Attribute {
	case elastictv.TitleSearchAttribute:
		return t.searchTVShowByTitle(params.Query)
	case elastictv.DirectorSearchAttribute:
		return t.searchTVShowByDirector(params.Query)
	case elastictv.ActorSearchAttribute:
		return t.searchTVShowByActor(params.Query)
	case elastictv.TMDbIDSearchAttribute:
		return t.getTVShowDetails(params.Query)
	default:
		return nil
	}
}

func (t TMDb) searchTVShowByTitle(tvshowTitle any) error {
	title, ok := tvshowTitle.(string)
	if !ok {
		return fmt.Errorf("%s: cannot convert query item [ %s ] to tv show title", t.Name(), tvshowTitle)
	}

	log.Printf("%s: Searching for tvshow by title [ %s ]", t.Name(), title)

	tvshows, err := t.tmdb.SearchTv(title, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching tvshow title [%s]: %w",
			t.Name(), tvshowTitle, err)
	}

	var errors *multierror.Error
	for _, tvshow := range tvshows.Results {
		if err := t.getTVShowDetails(tvshow.ID, tvshow.OriginalLanguage); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) searchTVShowByDirector(director any) error {
	name, ok := director.(string)
	if !ok {
		return fmt.Errorf("%s: cannot convert query item [ %s ] to director name", t.Name(), director)
	}

	log.Printf("%s: Searching for tvshow director credits [ %s ]", t.Name(), name)

	persons, err := t.tmdb.SearchPerson(name, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching for person [%s]: %w",
			t.Name(), director, err)
	}

	var errors *multierror.Error
	for _, person := range persons.Results {
		credits, err := t.tmdb.GetPersonTvCredits(person.ID, t.getDefaultOptions())
		if err != nil {
			errors = multierror.Append(errors,
				fmt.Errorf("%s: error searching tvshow credits for person [%s]: %w",
					t.Name(), person.Name, err))
		}

		for _, credit := range credits.Crew {
			if credit.Job != directorJob {
				continue
			}

			if err := t.getTVShowDetails(credit.ID, credit.OriginalLanguage); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) searchTVShowByActor(actor any) error {
	name, ok := actor.(string)
	if !ok {
		return fmt.Errorf("%s: cannot convert query item [ %s ] to director name", t.Name(), actor)
	}

	log.Printf("%s: Searching for tvshow actor credits [ %s ]", t.Name(), name)

	persons, err := t.tmdb.SearchPerson(name, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching for person [%s]: %w",
			t.Name(), actor, err)
	}

	var errors *multierror.Error
	for _, person := range persons.Results {
		credits, err := t.tmdb.GetPersonTvCredits(person.ID, t.getDefaultOptions())
		if err != nil {
			errors = multierror.Append(errors,
				fmt.Errorf("%s: error searching credits for person [%s]: %w",
					t.Name(), person.Name, err))
		}

		for _, credit := range credits.Cast {
			if err := t.getTVShowDetails(credit.ID, credit.OriginalLanguage); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) getTVShowDetails(tvShowID any, originalLanguage ...string) error {
	tmdbID, ok := tvShowID.(int)
	if !ok {
		return fmt.Errorf("%s: cannot convert id [ %s ] TMDb", t.Name(), tvShowID)
	}

	if t.hasBeenIndexed(tmdbID, elastictv.TvShowType) {
		return nil
	}

	options := t.getDefaultOptions()
	options["append_to_response"] = "translations,alternative_titles,credits,external_ids"

	if len(originalLanguage) > 0 {
		options["language"] = t.getDetailsLanguage(originalLanguage[0])
	}

	details, err := t.tmdb.GetTvInfo(tmdbID, options)
	if err != nil {
		return fmt.Errorf("%s: error getting details for ID %d: %w", t.Name(), tmdbID, err)
	}

	// If original language was not known before the above request then check if the original
	// language is one that we want to keep and if so make the request again
	if len(originalLanguage) == 0 && t.getDetailsLanguage(details.OriginalLanguage) != options["language"] {
		options["language"] = t.getDetailsLanguage(details.OriginalLanguage)

		details, err = t.tmdb.GetTvInfo(tmdbID, options)
		if err != nil {
			return fmt.Errorf("%s: error getting details for ID %d: %w", t.Name(), tmdbID, err)
		}
	}

	log.Printf("%s: Got details for tvshow [ %s ]", t.Name(), details.Name)

	tvshow := elastictv.Title{
		Title: details.Name,
		Genre: t.getGenres(details.Genres),
		IDs: elastictv.IDs{
			TMDb: details.ID,
			IMDb: details.ExternalIDs.ImdbID,
		},
		Rating: t.getRating(details.VoteAverage),
		Image:  t.getImage(details.PosterPath),
		Description: elastictv.Description{
			Text:   details.Overview,
			Source: t.Name(),
		},
		Year:     t.getYear(details.FirstAirDate),
		Country:  t.getCountries(details.ProductionCountries),
		Language: t.getLanguage(details.SpokenLanguages),
		Credits:  t.getCredits(details.Credits.Cast, details.Credits.Crew),
		Alias:    t.getTVAliases(*details.Translations, *details.AlternativeTitles, details.Name),
		Type:     elastictv.TvShowType,
	}

	if err := t.estv.UpsertTitle(tvshow); err != nil {
		return fmt.Errorf("error indexing tvshow: %w", err)
	}

	return nil
}

func (t TMDb) getTVAliases(tr tmdb.TvTranslations, altTitles tmdb.TvAlternativeTitles, name string) []string {
	aliases := make([]string, 0)

	for _, translation := range tr.Translations {
		aliases = t.addAlias(aliases, translation.Iso3166_1, translation.Data.Name, name)
	}

	for _, altTitle := range altTitles.Results {
		aliases = t.addAlias(aliases, altTitle.Iso3166_1, altTitle.Title, name)
	}

	return aliases
}
