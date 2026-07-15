package assistant

import (
	"context"
	"errors"
	"strings"
	"time"

	"gin-looklook/internal/order"
	"gin-looklook/internal/search"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type orderReader interface {
	List(ctx context.Context, userID, lastID, pageSize, state int64) ([]order.HomestayOrder, error)
	Detail(ctx context.Context, userID int64, sn string) (*order.HomestayOrder, error)
}

type homestayReader interface {
	Homestay(ctx context.Context, id int64) (*travel.Homestay, error)
}

type catalogSearcher interface {
	Search(ctx context.Context, query search.Query) (*search.SearchResult, error)
}

// Service runs a read-only Eino ReAct agent for the authenticated user.
type Service struct {
	model     model.ToolCallingChatModel
	orders    orderReader
	homestays homestayReader
	catalog   catalogSearcher
	timeout   time.Duration
}

func NewService(ctx context.Context, cfg shared.Config, orders orderReader, homestays homestayReader, catalog catalogSearcher) (*Service, error) {
	s := &Service{orders: orders, homestays: homestays, catalog: catalog, timeout: cfg.AgentTimeout}
	if cfg.AgentAPIKey == "" {
		return s, nil
	}
	if s.timeout <= 0 {
		s.timeout = 20 * time.Second
	}
	temperature := float32(0.2)
	maxTokens := 600
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:              cfg.AgentAPIKey,
		BaseURL:             cfg.AgentBaseURL,
		Model:               cfg.AgentModel,
		Timeout:             s.timeout,
		Temperature:         &temperature,
		MaxCompletionTokens: &maxTokens,
	})
	if err != nil {
		return nil, err
	}
	s.model = chatModel
	return s, nil
}

func (s *Service) Enabled() bool { return s != nil && s.model != nil }

func (s *Service) Ask(ctx context.Context, userID int64, question string) (string, error) {
	if !s.Enabled() {
		return "", shared.E(shared.CodeCommon, "AI assistant is not configured", nil)
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return "", shared.E(shared.CodeParam, "question is required", nil)
	}
	tools, err := s.tools(userID)
	if err != nil {
		return "", err
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "homestay_customer_service",
		Description: "根据需求推荐真实民宿，并帮助登录用户查询自己的订单",
		Instruction: `你是民宿平台的智能客服。你只能依据工具返回的数据回答，不得编造产品、订单、价格、房态或状态。
用户提出城市、预算、人数、偏好等住宿需求时，使用推荐工具检索真实民宿，并说明每个推荐与需求匹配的具体理由。
如果关键需求缺失，可以先询问城市、入住人数、每晚预算或偏好；信息足够时直接检索，不要重复追问。
推荐结果只是房源属性匹配，不代表指定日期一定有房；你没有实时房态工具，不得声称可以查询空房或库存，只能提醒用户在下单页面确认。
订单工具已固定绑定当前登录用户，不要询问或猜测其他用户的信息。
你没有取消、支付、退款或修改数据的工具；用户要求执行有副作用的操作时，明确说明只能提供查询和操作指引。
金额单位为人民币元，日期使用清晰易读的格式。回答简洁，并使用中文。`,
		Model: s.model,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools,
		}},
		MaxIterations: 6,
	})
	if err != nil {
		return "", err
	}

	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	iter := adk.NewRunner(runCtx, adk.RunnerConfig{Agent: agent}).Query(runCtx, question)
	var answer string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		message, messageErr := event.Output.MessageOutput.GetMessage()
		if messageErr != nil {
			return "", messageErr
		}
		if message != nil && message.Role == schema.Assistant && strings.TrimSpace(message.Content) != "" {
			answer = strings.TrimSpace(message.Content)
		}
	}
	if answer == "" {
		return "", errors.New("agent returned no answer")
	}
	return answer, nil
}

func (s *Service) tools(userID int64) ([]tool.BaseTool, error) {
	return buildTools(s.orders, s.homestays, s.catalog, userID)
}
