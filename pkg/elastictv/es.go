package elastictv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const timeFormat = "2006-01-02T15:04:05.0000000"

func (estv ElasticTV) queryES(query *Query, index string, doc interface{}) (string, float64, error) {
	buf, err := estv.encodeQuery(query)
	if err != nil {
		return "", 0, err
	}

	response, err := estv.Client.Search(
		estv.Client.Search.WithContext(context.Background()),
		estv.Client.Search.WithFrom(0),
		estv.Client.Search.WithSize(1),
		estv.Client.Search.WithIndex(index),
		estv.Client.Search.WithBody(buf),
	)
	if err != nil {
		buf, _ := estv.encodeQuery(query)

		return "", 0, fmt.Errorf("error querying elasticsearch: Error: %w Query: %s",
			err, buf.String())
	}
	defer response.Body.Close()

	esDoc := esResult{}
	if err := json.NewDecoder(response.Body).Decode(&esDoc); err != nil {
		return "", 0, fmt.Errorf("error parsing reply: %w", err)
	}

	if esDoc.Error.Reason != "" {
		buf, _ := estv.encodeQuery(query)

		return "", 0, fmt.Errorf("error of type [%s] return from elasticsearch: Error %s Query: %s",
			esDoc.Error.Type, esDoc.Error.Reason, buf.String())
	}

	if esDoc.Hits.Total.Value == 0 {
		return "", 0, nil
	}

	if doc != nil {
		if err := json.Unmarshal(esDoc.Hits.Hits[0].Source, doc); err != nil {
			return "", 0, fmt.Errorf("error parsing source: %w", err)
		}
	}

	return esDoc.Hits.Hits[0].ID, esDoc.Hits.Hits[0].Score, nil
}

func (estv ElasticTV) encodeQuery(query *Query) (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("error encoding query: %w", err)
	}

	return &buf, nil
}

func (estv ElasticTV) index(index, docID string, doc interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return fmt.Errorf("error encoding document: %w", err)
	}

	request := esapi.IndexRequest{
		Index:      index,
		DocumentID: docID,
		Body:       &buf,
		Refresh:    "false",
	}

	res, err := request.Do(context.Background(), estv.Client)
	if err != nil {
		return fmt.Errorf("error indexing document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		response, _ := io.ReadAll(res.Body)

		return fmt.Errorf("[%s] Error indexing document ID=%s: %s",
			res.Status(), request.DocumentID, string(response))
	}

	return nil
}

func (estv ElasticTV) RefreshIndices(indices ...string) error {
	if len(indices) == 0 {
		return errors.New("no indices to refresh")
	}

	request := esapi.IndicesRefreshRequest{
		Index: indices,
	}

	res, err := request.Do(context.Background(), estv.Client)
	if err != nil {
		return fmt.Errorf("error refreshing index: %w", err)
	}
	defer res.Body.Close()

	return nil
}

func (estv ElasticTV) GetRecordID(query *Query, index string) (string, error) {
	id, _, err := estv.queryES(query, index, nil)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (estv ElasticTV) getRecordWithScore(query *Query, index string, doc interface{}) (float64, error) {
	_, score, err := estv.queryES(query, index, doc)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (estv ElasticTV) UpsertTitle(title Title) error {
	query := NewQuery().
		WithTMDbID(title.IDs.TMDb).
		WithIMDbID(title.IDs.IMDb)

	recordID, err := estv.GetRecordID(query, estv.Index.Title)
	if err != nil {
		return err
	}

	title.Timestamp = time.Now().UTC().Format(timeFormat)

	return estv.index(estv.Index.Title, recordID, title)
}

func (estv ElasticTV) UpsertEpisode(episode Episode) error {
	query := NewQuery().
		WithTVShowTMDbID(episode.TVShowIDs.TMDb).
		WithEpisodeNumber(episode.EpisodeNo).
		WithSeasonNumber(episode.SeasonNo)

	recordID, err := estv.GetRecordID(query, estv.Index.Episode)
	if err != nil {
		return err
	}

	episode.Timestamp = time.Now().UTC().Format(timeFormat)

	return estv.index(estv.Index.Episode, recordID, episode)
}

func (estv ElasticTV) IsRecordExpired(query *Query, index string) bool {
	var docTimestamp Timestamp

	docID, _, err := estv.queryES(query, index, &docTimestamp)
	if err != nil || docID == "" {
		return true
	}

	return estv.RequiresUpdate(docTimestamp.Timestamp)
}

func (estv ElasticTV) RequiresUpdate(timestamp string) bool {
	lastUpdated, err := time.Parse(timeFormat, timestamp)
	if err != nil {
		return true
	}

	return lastUpdated.Before(estv.UpdateAfter)
}
