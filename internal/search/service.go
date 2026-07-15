package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"
)

type Service struct {
	repo   *Repository
	base   string
	index  string
	client *http.Client
}

type homestayDocument struct {
	ID                  int64     `json:"id"`
	Version             int64     `json:"version"`
	Title               string    `json:"title"`
	SubTitle            string    `json:"subTitle"`
	Banner              string    `json:"banner"`
	Info                string    `json:"info"`
	City                string    `json:"city"`
	Tags                []string  `json:"tags"`
	Star                float64   `json:"star"`
	Location            *geoPoint `json:"location,omitempty"`
	PeopleNum           int64     `json:"peopleNum"`
	HomestayBusinessID  int64     `json:"homestayBusinessId"`
	UserID              int64     `json:"userId"`
	RowState            int64     `json:"rowState"`
	RowType             int64     `json:"rowType"`
	FoodInfo            string    `json:"foodInfo"`
	FoodPrice           int64     `json:"foodPrice"`
	HomestayPrice       int64     `json:"homestayPrice"`
	MarketHomestayPrice int64     `json:"marketHomestayPrice"`
}

type geoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func NewService(repo *Repository, cfg shared.Config) *Service {
	return &Service{repo: repo, base: strings.TrimRight(cfg.ElasticsearchURL, "/"), index: cfg.SearchIndex, client: &http.Client{Timeout: 8 * time.Second}}
}

func (s *Service) EnsureIndex(ctx context.Context) error {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.base+"/"+url.PathEscape(s.index), nil)
		if err == nil {
			resp, doErr := s.client.Do(req)
			if doErr == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
				if resp.StatusCode == http.StatusNotFound {
					return s.createIndex(ctx)
				}
				lastErr = fmt.Errorf("elasticsearch HEAD index status %d", resp.StatusCode)
			} else {
				lastErr = doErr
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("elasticsearch unavailable: %w", lastErr)
}

func (s *Service) createIndex(ctx context.Context) error {
	body := map[string]any{"mappings": map[string]any{"properties": map[string]any{
		"id": map[string]any{"type": "long"}, "version": map[string]any{"type": "long"},
		"title":    map[string]any{"type": "text", "fields": map[string]any{"keyword": map[string]any{"type": "keyword"}}},
		"subTitle": map[string]any{"type": "text"}, "info": map[string]any{"type": "text"},
		"city": map[string]any{"type": "keyword"}, "tags": map[string]any{"type": "keyword"},
		"star": map[string]any{"type": "float"}, "location": map[string]any{"type": "geo_point"},
		"peopleNum": map[string]any{"type": "integer"}, "homestayBusinessId": map[string]any{"type": "long"},
		"userId": map[string]any{"type": "long"}, "rowState": map[string]any{"type": "byte"}, "rowType": map[string]any{"type": "byte"},
		"foodPrice": map[string]any{"type": "long"}, "homestayPrice": map[string]any{"type": "long"}, "marketHomestayPrice": map[string]any{"type": "long"},
	}}}
	err := s.doJSON(ctx, http.MethodPut, "/"+url.PathEscape(s.index), body, nil)
	if err != nil && strings.Contains(err.Error(), "resource_already_exists_exception") {
		return nil
	}
	return err
}

func toDocument(v *travel.Homestay) homestayDocument {
	doc := homestayDocument{ID: v.ID, Version: v.Version, Title: v.Title, SubTitle: v.SubTitle, Banner: v.Banner, Info: v.Info, City: v.City, Tags: splitTags(v.Tags), Star: v.Star, PeopleNum: v.PeopleNum, HomestayBusinessID: v.HomestayBusinessID, UserID: v.UserID, RowState: v.RowState, RowType: v.RowType, FoodInfo: v.FoodInfo, FoodPrice: v.FoodPrice, HomestayPrice: v.HomestayPrice, MarketHomestayPrice: v.MarketHomestayPrice}
	if v.Latitude >= -90 && v.Latitude <= 90 && v.Longitude >= -180 && v.Longitude <= 180 && (v.Latitude != 0 || v.Longitude != 0) {
		doc.Location = &geoPoint{Lat: v.Latitude, Lon: v.Longitude}
	}
	return doc
}

func splitTags(tags string) []string {
	// TODO(practice-07): 同时支持中英文逗号，去除空白并忽略空标签。
	return nil
}

func (s *Service) IndexHomestay(ctx context.Context, v *travel.Homestay) error {
	path := "/" + url.PathEscape(s.index) + "/_doc/" + strconv.FormatInt(v.ID, 10) + "?refresh=wait_for"
	return s.doJSON(ctx, http.MethodPut, path, toDocument(v), nil)
}

func (s *Service) DeleteHomestay(ctx context.Context, id int64) error {
	path := "/" + url.PathEscape(s.index) + "/_doc/" + strconv.FormatInt(id, 10) + "?refresh=wait_for"
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return elasticsearchError(resp)
}

func (s *Service) Search(ctx context.Context, q Query) (*SearchResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 10
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	filters := []any{map[string]any{"term": map[string]any{"rowState": 1}}}
	if q.City != "" {
		filters = append(filters, map[string]any{"term": map[string]any{"city": q.City}})
	}
	if q.MinPrice > 0 || q.MaxPrice > 0 {
		rangeValue := map[string]any{}
		if q.MinPrice > 0 {
			rangeValue["gte"] = q.MinPrice
		}
		if q.MaxPrice > 0 {
			rangeValue["lte"] = q.MaxPrice
		}
		filters = append(filters, map[string]any{"range": map[string]any{"homestayPrice": rangeValue}})
	}
	if q.MinStar > 0 {
		filters = append(filters, map[string]any{"range": map[string]any{"star": map[string]any{"gte": q.MinStar}}})
	}
	if len(q.Tags) > 0 {
		filters = append(filters, map[string]any{"terms": map[string]any{"tags": q.Tags}})
	}
	validGeo := q.DistanceKM > 0 && q.Latitude >= -90 && q.Latitude <= 90 && q.Longitude >= -180 && q.Longitude <= 180
	if validGeo {
		filters = append(filters, map[string]any{"geo_distance": map[string]any{"distance": fmt.Sprintf("%gkm", q.DistanceKM), "location": map[string]any{"lat": q.Latitude, "lon": q.Longitude}}})
	}
	boolQuery := map[string]any{"filter": filters}
	if strings.TrimSpace(q.Keyword) != "" {
		boolQuery["must"] = []any{map[string]any{"multi_match": map[string]any{"query": strings.TrimSpace(q.Keyword), "fields": []string{"title^3", "subTitle^2", "info", "tags^2"}}}}
	}
	body := map[string]any{"from": (q.Page - 1) * q.PageSize, "size": q.PageSize, "track_total_hits": true, "query": map[string]any{"bool": boolQuery}, "sort": searchSort(q.SortBy, validGeo, q.Latitude, q.Longitude)}
	var raw struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source homestayDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := s.doJSON(ctx, http.MethodPost, "/"+url.PathEscape(s.index)+"/_search", body, &raw); err != nil {
		return nil, shared.E(shared.CodeCommon, "搜索服务繁忙,请稍后再试", err)
	}
	result := &SearchResult{Total: raw.Hits.Total.Value, Items: make([]HomestayDoc, 0, len(raw.Hits.Hits))}
	for _, hit := range raw.Hits.Hits {
		doc := hit.Source
		item := HomestayDoc{ID: doc.ID, Version: doc.Version, Title: doc.Title, SubTitle: doc.SubTitle, Banner: doc.Banner, Info: doc.Info, City: doc.City, Tags: strings.Join(doc.Tags, ","), Star: doc.Star, PeopleNum: doc.PeopleNum, HomestayBusinessID: doc.HomestayBusinessID, UserID: doc.UserID, RowState: doc.RowState, RowType: doc.RowType, FoodInfo: doc.FoodInfo, FoodPrice: doc.FoodPrice, HomestayPrice: doc.HomestayPrice, MarketHomestayPrice: doc.MarketHomestayPrice}
		if doc.Location != nil {
			item.Latitude, item.Longitude = doc.Location.Lat, doc.Location.Lon
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func searchSort(sortBy []string, validGeo bool, lat, lon float64) []any {
	// TODO(practice-07): 把白名单排序转换成 ES DSL，地理排序需要合法坐标，并保留 id 稳定次序。
	return nil
}

func (s *Service) doJSON(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return elasticsearchError(resp)
	}
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	return nil
}

func elasticsearchError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("elasticsearch status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

func YuanToFen(value float64) int64 { return int64(math.Round(value * 100)) }
