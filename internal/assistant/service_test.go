package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeToolCallingModel struct {
	calls     int
	tools     []*schema.ToolInfo
	toolName  string
	arguments string
	answer    string
}

func (m *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	m.tools = tools
	return m, nil
}

func (m *fakeToolCallingModel) Generate(_ context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.calls++
	if tools := model.GetCommonOptions(nil, opts...).Tools; len(tools) > 0 {
		m.tools = tools
	}
	if m.calls == 1 {
		toolName, arguments := m.toolName, m.arguments
		if toolName == "" {
			toolName, arguments = "list_my_orders", `{"page_size":5}`
		}
		return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			ID: "call-list", Type: "function",
			Function: schema.FunctionCall{Name: toolName, Arguments: arguments},
		}}}, nil
	}
	if len(input) == 0 || input[len(input)-1].Role != schema.Tool {
		return nil, errors.New("tool result was not returned to the model")
	}
	answer := m.answer
	if answer == "" {
		answer = "你有一个待支付订单 HSO1，金额 299 元。"
	}
	return schema.AssistantMessage(answer, nil), nil
}

func TestAskCombinesRecommendationToolWithCustomerServiceAgent(t *testing.T) {
	model := &fakeToolCallingModel{
		toolName:  "recommend_homestays",
		arguments: `{"city":"杭州","max_nightly_yuan":500,"guests":4,"tags":["亲子"],"sort_by":"recommended"}`,
		answer:    "推荐西湖家庭房，每晚 399 元，可住 4 人，符合亲子需求。",
	}
	catalog := &mockCatalog{}
	service := &Service{
		model: model, orders: &mockOrders{}, homestays: &mockHomestays{}, catalog: catalog, timeout: time.Second,
	}
	answer, err := service.Ask(context.Background(), 42, "推荐杭州四人亲子民宿，每晚不超过 500 元")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if !strings.Contains(answer, "西湖家庭房") || model.calls != 2 {
		t.Fatalf("answer=%q modelCalls=%d, want recommendation tool loop", answer, model.calls)
	}
	if catalog.query.City != "杭州" || catalog.query.MaxPrice != 50000 {
		t.Fatalf("recommendation query = %+v", catalog.query)
	}
}

func (*fakeToolCallingModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("streaming is not used in this test")
}

func TestAskRunsEinoReActToolLoop(t *testing.T) {
	model := &fakeToolCallingModel{}
	orders := &mockOrders{}
	service := &Service{
		model: model, orders: orders, homestays: &mockHomestays{}, catalog: &mockCatalog{}, timeout: time.Second,
	}
	answer, err := service.Ask(context.Background(), 42, "我最近的订单是什么？")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if !strings.Contains(answer, "HSO1") || model.calls != 2 {
		t.Fatalf("answer=%q modelCalls=%d, want a two-step tool loop", answer, model.calls)
	}
	if orders.listUserID != 42 {
		t.Fatalf("tool userID = %d, want authenticated user 42", orders.listUserID)
	}
	if len(model.tools) != 4 {
		t.Fatalf("model received %d tools, want 4", len(model.tools))
	}
}
