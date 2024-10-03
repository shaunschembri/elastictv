package tmdb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shaunschembri/go-tmdb"
	"github.com/spf13/viper"

	"github.com/shaunschembri/elastictv/pkg/elastictv"
)

const (
	maxActors            = 10
	maxOtherCredits      = 5
	directorJob          = "Director"
	producerJob          = "Producer"
	executiveProducerJob = "Executive Producer"
)

type TMDb struct {
	estv     *elastictv.ElasticTV
	tmdb     *tmdb.TMDb
	language string
	genre    map[int]string
	// List of country codes to use titles from
	aliasCountryCodes []string
	// List of language codes to use title and description in original language.
	originalLanguageCodes []string
	// Use original spoken language name (ex Italiano for Italian). If set to false the English language name is used.
	useOriginalSpokenLanguage bool
}

func (t TMDb) Name() string {
	return "TMDb"
}

func (t TMDb) Init(estv *elastictv.ElasticTV) (elastictv.SearchableProvider, error) {
	t.tmdb = tmdb.Init(tmdb.Config{
		APIKey: viper.GetString("elastictv.provider.tmdb.api_key"),
	})
	t.language = viper.GetString("elastictv.provider.tmdb.language")
	t.aliasCountryCodes = viper.GetStringSlice("elastictv.provider.tmdb.alias_countries")
	t.originalLanguageCodes = viper.GetStringSlice("elastictv.provider.tmdb.keep_original_title_desc")
	t.useOriginalSpokenLanguage = viper.GetBool("elastictv.provider.tmdb.use_original_spoken_language")
	t.estv = estv

	genres, err := t.tmdb.GetMovieGenres(t.getDefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("error getting genres from TMDb: %w", err)
	}

	t.genre = make(map[int]string)
	for _, g := range genres.Genres {
		t.genre[g.ID] = g.Name
	}

	return t, nil
}

func (t TMDb) getDefaultOptions() map[string]string {
	return map[string]string{
		"language": t.language,
	}
}

func (t TMDb) getYear(releaseDate string) uint16 {
	if len(releaseDate) < 4 {
		return 0
	}

	year, _ := strconv.Atoi(releaseDate[:4])
	return uint16(year)
}

func (t TMDb) getImage(image string) string {
	if len(image) == 0 {
		return ""
	}

	return image[1:]
}

func (t TMDb) getCountries(list tmdb.ProductionCountries) []string {
	result := make([]string, 0)

	for _, n := range list {
		result = append(result, n.Name)
	}

	return result
}

func (t TMDb) getLanguage(languages tmdb.SpokenLanguages) string {
	if len(languages) == 0 {
		return ""
	}

	if t.useOriginalSpokenLanguage {
		return languages[0].Name
	}
	return languages[0].EnglishName
}

func (t TMDb) getGenres(list tmdb.Genres) []string {
	genres := make([]string, 0)
	for _, genre := range list {
		if genre, ok := t.genre[genre.ID]; ok {
			genres = append(genres, genre)
		}
	}

	return genres
}

func (t TMDb) addAlias(aliases []string, isoCode, name, title string) []string {
	if name == "" {
		return aliases
	}
	// Check if alias country is in the required list
	if !t.isAliasRequired(isoCode) {
		return aliases
	}
	// Check if name is not the title
	if strings.EqualFold(title, name) {
		return aliases
	}
	// Check if name is not already in aliases
	if t.contains(aliases, name) {
		return aliases
	}
	aliases = append(aliases, name)

	return aliases
}

func (t TMDb) isAliasRequired(isoCode string) bool {
	for _, n := range t.aliasCountryCodes {
		if isoCode == n {
			return true
		}
	}

	return false
}

func (t TMDb) getDetailsLanguage(lang string) string {
	if t.contains(t.originalLanguageCodes, lang) {
		return lang
	}

	return t.language
}

func (t TMDb) contains(stringSlice []string, value string) bool {
	for _, n := range stringSlice {
		if strings.EqualFold(n, value) {
			return true
		}
	}

	return false
}

func (t TMDb) getCredits(cast tmdb.CastCredit, crew tmdb.CrewCredit) elastictv.Credits {
	credits := elastictv.Credits{}

	for _, actor := range cast {
		if actor.Order < maxActors {
			credits.Actor = append(credits.Actor, actor.Name)
		}
	}

	for _, person := range crew {
		if person.Job == directorJob {
			credits.Director = append(credits.Director, person.Name)
		}

		if len(credits.Other) < maxOtherCredits && (person.Job == producerJob || person.Job == executiveProducerJob) {
			credits.Other = append(credits.Other, person.Name)
		}
	}

	return credits
}

func (t TMDb) hasBeenIndexed(id int, titleType string) bool {
	query := elastictv.NewQuery().
		WithTMDbID(id).
		WithType(titleType)

	docID, err := t.estv.GetRecordID(query, t.estv.Index.Title)
	if err != nil || docID == "" {
		return false
	}

	return true
}

func (t TMDb) getRating(rating float32) *elastictv.Rating {
	if rating > 0 {
		return &elastictv.Rating{
			Value:  rating,
			Source: t.Name(),
		}
	}

	return nil
}
