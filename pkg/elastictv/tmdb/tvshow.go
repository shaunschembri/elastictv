package tmdb

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/shaunschembri/go-tmdb"

	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

func (t TMDb) searchTVShow(params elastictv.SearchTitlesParams) error {
	switch params.Attribute {
	case elastictv.TitleAttribute:
		return t.searchTVShowByTitle(params.Query)
	case elastictv.DirectorAttribute:
		return t.searchTVShowByDirector(params.Query)
	case elastictv.ActorAttribute:
		return t.searchTVShowByActor(params.Query)
	default:
		return nil
	}
}

func (t TMDb) searchTVShowByTitle(tvshowTitle string) error {
	log.Printf("%s: Searching for tvshow by title [ %s ]",
		t.Name(), tvshowTitle)

	tvshows, err := t.tmdb.SearchTv(tvshowTitle, t.getDefaultOptions())
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

func (t TMDb) searchTVShowByDirector(director string) error {
	log.Printf("%s: Searching for tvshow director credits [ %s ]",
		t.Name(), director)

	persons, err := t.tmdb.SearchPerson(director, t.getDefaultOptions())
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

func (t TMDb) searchTVShowByActor(actor string) error {
	log.Printf("%s: Searching for tvshow actor credits [ %s ]",
		t.Name(), actor)

	persons, err := t.tmdb.SearchPerson(actor, t.getDefaultOptions())
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

func (t TMDb) getTVShowDetails(id int, originalLanguage string) error {
	if t.hasBeenIndexed(id, elastictv.TVShowType) {
		return nil
	}

	options := t.getDefaultOptions()
	options["append_to_response"] = "translations,alternative_titles,credits,external_ids"
	options["language"] = t.getDetailsLanguage(originalLanguage)

	details, err := t.tmdb.GetTvInfo(id, options)
	if err != nil {
		return fmt.Errorf("%s: error getting details for ID %d: %w", t.Name(), id, err)
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
		Type:     elastictv.TVShowType,
	}

	if err := t.estv.UpsertTitle(tvshow); err != nil {
		return fmt.Errorf("error indexing tvshow: %w", err)
	}

	return nil
}

func (t TMDb) getTVAliases(tr tmdb.TvTranslations, at tmdb.TvAlternativeTitles, name string) []string {
	aliases := make([]string, 0)

	for _, translation := range tr.Translations {
		aliases = t.addAlias(aliases, translation.Iso3166_1, translation.Data.Name, name)
	}

	for _, altTitle := range at.Results {
		aliases = t.addAlias(aliases, altTitle.Iso3166_1, altTitle.Title, name)
	}

	return aliases
}
