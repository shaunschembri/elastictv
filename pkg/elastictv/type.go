package elastictv

import (
	"fmt"
)

type Type int

const (
	MovieType Type = iota + 1
	TvShowType
	EpisodeType
)

var typesList = [...]string{"movie", "tv", "episode"}

var typesMap = map[string]Type{
	"movie":   MovieType,
	"tv":      TvShowType,
	"episode": EpisodeType,
}

func (t Type) MarshalText() ([]byte, error) {
	return []byte(typesList[t-1]), nil
}

func (t *Type) UnmarshalText(text []byte) error {
	docType, ok := typesMap[string(text)]
	if !ok {
		return fmt.Errorf("type [%s] is not valid", string(text))
	}

	*t = docType

	return nil
}

func (t Type) String() string {
	return typesList[t-1]
}
