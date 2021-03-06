package tmdb

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/shaunschembri/go-tmdb"

	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

func (t TMDb) searchMovie(params elastictv.SearchTitlesParams) error {
	switch params.Attribute {
	case elastictv.TitleAttribute:
		return t.searchMovieByTitle(params.Query, params.Year)
	case elastictv.DirectorAttribute:
		return t.searchMovieByDirector(params.Query, params.Year)
	case elastictv.ActorAttribute:
		return t.searchMovieByActor(params.Query, params.Year)
	default:
		return nil
	}
}

func (t TMDb) searchMovieByTitle(movieTitle string, year uint16) error {
	log.Printf("%s: Searching for movie by title [ %s | Year: %d ]",
		t.Name(), movieTitle, year)

	movies, err := t.tmdb.SearchMovie(movieTitle, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching movie title [ %s ]: %w",
			t.Name(), movieTitle, err)
	}

	var errors *multierror.Error
	for _, movie := range movies.Results {
		if t.getYear(movie.ReleaseDate) != year {
			continue
		}

		if err := t.getMovieDetails(movie.ID, movie.OriginalLanguage); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) searchMovieByDirector(director string, year uint16) error {
	log.Printf("%s: Searching for movie director credits [ %s | Year: %d ]",
		t.Name(), director, year)

	persons, err := t.tmdb.SearchPerson(director, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching for person [ %s ]: %w",
			t.Name(), director, err)
	}

	var errors *multierror.Error
	for _, person := range persons.Results {
		credits, err := t.tmdb.GetPersonMovieCredits(person.ID, t.getDefaultOptions())
		if err != nil {
			errors = multierror.Append(errors,
				fmt.Errorf("%s: error searching movie credits for person [ %s ]: %w",
					t.Name(), person.Name, err))
		}

		for _, credit := range credits.Crew {
			if credit.Job != directorJob {
				continue
			}

			if t.getYear(credit.ReleaseDate) != year {
				continue
			}

			if err := t.getMovieDetails(credit.ID, credit.OriginalLanguage); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) searchMovieByActor(actor string, year uint16) error {
	log.Printf("%s: Searching for movie actor credits [ %s | Year: %d ]",
		t.Name(), actor, year)

	persons, err := t.tmdb.SearchPerson(actor, t.getDefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: error searching for person [%s]: %w",
			t.Name(), actor, err)
	}

	var errors *multierror.Error
	for _, person := range persons.Results {
		credits, err := t.tmdb.GetPersonMovieCredits(person.ID, t.getDefaultOptions())
		if err != nil {
			errors = multierror.Append(errors,
				fmt.Errorf("%s: error searching credits for person [ %s ]: %w",
					t.Name(), person.Name, err))
		}

		for _, credit := range credits.Cast {
			if t.getYear(credit.ReleaseDate) != year {
				continue
			}

			if err := t.getMovieDetails(credit.ID, credit.OriginalLanguage); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	return errors.ErrorOrNil()
}

func (t TMDb) getMovieDetails(id int, originalLanguage string) error {
	if t.hasBeenIndexed(id, elastictv.MovieType) {
		return nil
	}

	options := t.getDefaultOptions()
	options["append_to_response"] = "translations,alternative_titles,credits"
	options["language"] = t.getDetailsLanguage(originalLanguage)

	details, err := t.tmdb.GetMovieInfo(id, options)
	if err != nil {
		return fmt.Errorf("%s: error getting details for ID %d: %w", t.Name(), id, err)
	}
	year := t.getYear(details.ReleaseDate)
	log.Printf("%s: Got details for movie [ %s | Year: %d ]",
		t.Name(), details.Title, year)

	movie := elastictv.Title{
		Title: details.Title,
		Genre: t.getGenres(details.Genres),
		IDs: elastictv.IDs{
			TMDb: details.ID,
			IMDb: details.ImdbID,
		},
		Rating: t.getRating(details.VoteAverage),
		Image:  t.getImage(details.PosterPath),
		Description: elastictv.Description{
			Text:   details.Overview,
			Source: t.Name(),
		},
		Year:     year,
		Tagline:  details.Tagline,
		Country:  t.getCountries(details.ProductionCountries),
		Language: t.getLanguage(details.SpokenLanguages),
		Credits:  t.getCredits(details.Credits.Cast, details.Credits.Crew),
		Alias:    t.getMovieAliases(*details.Translations, *details.AlternativeTitles, details.Title),
		Type:     elastictv.MovieType,
	}

	if err := t.estv.UpsertTitle(movie); err != nil {
		return fmt.Errorf("error indexing movie: %w", err)
	}

	return nil
}

func (t TMDb) getMovieAliases(tr tmdb.MovieTranslations, at tmdb.MovieAlternativeTitles, title string) []string {
	aliases := make([]string, 0)

	for _, translation := range tr.Translations {
		aliases = t.addAlias(aliases, translation.Iso3166_1, translation.Data.Title, title)
	}

	for _, altTitle := range at.Titles {
		aliases = t.addAlias(aliases, altTitle.Iso3166_1, altTitle.Title, title)
	}

	return aliases
}
