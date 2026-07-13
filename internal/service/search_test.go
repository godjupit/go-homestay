package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"
)

func TestSearchBuildsFiltersAndDecodesDocuments(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/homestays/_search" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":1},"hits":[{"_source":{"id":8,"version":2,"title":"西湖民宿","city":"杭州","tags":["西湖","亲子"],"star":4.8,"location":{"lat":30.25,"lon":120.15},"rowState":1,"homestayPrice":39900}}]}}`))
	}))
	defer server.Close()

	search := NewSearchService(nil, config.Config{ElasticsearchURL: server.URL, SearchIndex: "homestays"})
	result, err := search.Search(context.Background(), model.HomestaySearchQuery{Keyword: "西湖", City: "杭州", MinPrice: 20000, MaxPrice: 50000, Tags: []string{"亲子"}, MinStar: 4.5, Latitude: 30.2, Longitude: 120.1, DistanceKM: 10, SortBy: []string{"distance", "price_asc"}, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Tags != "西湖,亲子" || result.Items[0].HomestayPrice != 39900 {
		t.Fatalf("unexpected result: %+v", result)
	}
	query, ok := requestBody["query"].(map[string]any)
	if !ok || query["bool"] == nil {
		t.Fatalf("missing bool query: %#v", requestBody)
	}
	sorts, ok := requestBody["sort"].([]any)
	if !ok || len(sorts) < 2 {
		t.Fatalf("missing multi sort: %#v", requestBody["sort"])
	}
}

func TestSplitTags(t *testing.T) {
	got := splitTags("亲子, 西湖，停车场")
	if len(got) != 3 || got[1] != "西湖" {
		t.Fatalf("unexpected tags: %v", got)
	}
}
