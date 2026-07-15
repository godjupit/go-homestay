package assistant

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gin-looklook/internal/order"
	"gin-looklook/internal/search"
	"gin-looklook/internal/travel"

	"github.com/cloudwego/eino/components/tool"
)

type mockOrders struct {
	listUserID   int64
	detailUserID int64
}

func (m *mockOrders) List(_ context.Context, userID, _, _, _ int64) ([]order.HomestayOrder, error) {
	m.listUserID = userID
	return []order.HomestayOrder{{
		SN: "HSO1", Title: "西湖民宿", TradeState: order.TradeStateWaitPay,
		OrderTotalPrice: 29900, LiveStartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		LiveEndDate: time.Date(2026, 8, 2, 0, 0, 0, 0, time.Local),
	}}, nil
}

func (m *mockOrders) Detail(_ context.Context, userID int64, sn string) (*order.HomestayOrder, error) {
	m.detailUserID = userID
	return &order.HomestayOrder{SN: sn, HomestayID: 11, Title: "西湖民宿", OrderTotalPrice: 29900}, nil
}

type mockHomestays struct{}

func (*mockHomestays) Homestay(_ context.Context, id int64) (*travel.Homestay, error) {
	return &travel.Homestay{ID: id, Title: "西湖民宿", City: "杭州", HomestayPrice: 29900}, nil
}

type mockCatalog struct {
	query search.Query
}

func (m *mockCatalog) Search(_ context.Context, query search.Query) (*search.SearchResult, error) {
	m.query = query
	return &search.SearchResult{Total: 3, Items: []search.HomestayDoc{
		{ID: 1, Title: "单人小屋", City: "杭州", PeopleNum: 1, HomestayPrice: 19900, Star: 4.9},
		{ID: 2, Title: "西湖家庭房", City: "杭州", Tags: "亲子,湖景", PeopleNum: 4, HomestayPrice: 39900, MarketHomestayPrice: 49900, Star: 4.8, Info: "适合家庭入住"},
		{ID: 3, Title: "六人庭院", City: "杭州", PeopleNum: 6, HomestayPrice: 69900, Star: 4.7},
	}}, nil
}

func findInvokableTool(t *testing.T, tools []tool.BaseTool, name string) tool.InvokableTool {
	t.Helper()
	for _, candidate := range tools {
		info, err := candidate.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if info.Name == name {
			invokable, ok := candidate.(tool.InvokableTool)
			if !ok {
				t.Fatalf("tool %s is not invokable", name)
			}
			return invokable
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil
}

func TestOrderToolsBindAuthenticatedUser(t *testing.T) {
	orders := &mockOrders{}
	tools, err := buildTools(orders, &mockHomestays{}, &mockCatalog{}, 42)
	if err != nil {
		t.Fatal(err)
	}

	listResult, err := findInvokableTool(t, tools, "list_my_orders").InvokableRun(context.Background(), `{"page_size":5}`)
	if err != nil {
		t.Fatal(err)
	}
	if orders.listUserID != 42 {
		t.Fatalf("list userID = %d, want authenticated user 42", orders.listUserID)
	}
	var list listOrdersOutput
	if err = json.Unmarshal([]byte(listResult), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Orders) != 1 || list.Orders[0].TotalYuan != 299 {
		t.Fatalf("unexpected list output: %+v", list)
	}

	_, err = findInvokableTool(t, tools, "get_my_order").InvokableRun(context.Background(), `{"order_sn":"HSO1"}`)
	if err != nil {
		t.Fatal(err)
	}
	if orders.detailUserID != 42 {
		t.Fatalf("detail userID = %d, want authenticated user 42", orders.detailUserID)
	}
}

func TestHomestayToolUsesYuan(t *testing.T) {
	tools, err := buildTools(&mockOrders{}, &mockHomestays{}, &mockCatalog{}, 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := findInvokableTool(t, tools, "get_homestay").InvokableRun(context.Background(), `{"homestay_id":11}`)
	if err != nil {
		t.Fatal(err)
	}
	var value homestayDetailOutput
	if err = json.Unmarshal([]byte(result), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID != 11 || value.PriceYuan != 299 {
		t.Fatalf("unexpected homestay output: %+v", value)
	}
}

func TestRecommendToolSearchesRealCatalogAndFiltersCapacity(t *testing.T) {
	catalog := &mockCatalog{}
	tools, err := buildTools(&mockOrders{}, &mockHomestays{}, catalog, 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := findInvokableTool(t, tools, "recommend_homestays").InvokableRun(
		context.Background(),
		`{"keyword":"湖景","city":"杭州","min_nightly_yuan":200,"max_nightly_yuan":500,"guests":3,"tags":["亲子"],"min_star":4.5,"sort_by":"price_asc"}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.query.City != "杭州" || catalog.query.Keyword != "湖景" || catalog.query.MinPrice != 20000 || catalog.query.MaxPrice != 50000 {
		t.Fatalf("unexpected search query: %+v", catalog.query)
	}
	if catalog.query.PageSize != 50 || len(catalog.query.SortBy) != 1 || catalog.query.SortBy[0] != "price_asc" {
		t.Fatalf("unexpected paging or sort: %+v", catalog.query)
	}
	var output recommendHomestaysOutput
	if err = json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatal(err)
	}
	if output.ReturnedCount != 2 || output.Candidates[0].ID != 2 || output.Candidates[0].NightlyPriceYuan != 399 {
		t.Fatalf("unexpected recommendations: %+v", output)
	}
}
