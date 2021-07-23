# ElasticTV

ElasticTV sources information a movie or a TV show and caches them in an Elasticsearch to speed up subsequent queries to the same title.  The project is under active development and currently can only source data from [The Movie Database (TMDb)](https://www.themoviedb.org/) and aims to be compatible to the schema used by [ElasticTMDB](https://github.com/shaunschembri/ElasticTMDB), however it has been designed to support other providers other then (TMDb).

## Planned features
- Update schema to support multiple providers.
- Automatically create indexes from the [index mappings](configs).
- Support the import of the [IMDb datasets](https://datasets.imdbws.com/) which can be used to populate an empty index.
- Support other searchable providers like [Trakt](https://trakt.tv/) and [TVMaze](https://www.tvmaze.com).