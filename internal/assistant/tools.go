package assistant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/order"
	"gin-looklook/internal/search"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type listOrdersInput struct {
	PageSize int64 `json:"page_size" jsonschema_description:"返回数量，范围 1 到 10"`
}

type orderDetailInput struct {
	OrderSN string `json:"order_sn" jsonschema:"required" jsonschema_description:"订单号，以 HSO 开头"`
}

type homestayDetailInput struct {
	HomestayID int64 `json:"homestay_id" jsonschema:"required" jsonschema_description:"民宿 ID"`
}

type recommendHomestaysInput struct {
	Keyword        string   `json:"keyword" jsonschema_description:"住宿偏好或关键词，例如西湖、安静、亲子；没有则留空"`
	City           string   `json:"city" jsonschema_description:"目标城市；没有明确城市则留空"`
	MinNightlyYuan float64  `json:"min_nightly_yuan" jsonschema_description:"每晚最低预算，人民币元；没有则为 0"`
	MaxNightlyYuan float64  `json:"max_nightly_yuan" jsonschema_description:"每晚最高预算，人民币元；没有则为 0"`
	Guests         int64    `json:"guests" jsonschema_description:"入住人数；没有说明则为 0"`
	Tags           []string `json:"tags" jsonschema_description:"用户明确要求的标签，最多 5 个"`
	MinStar        float64  `json:"min_star" jsonschema_description:"最低评分，范围 0 到 5"`
	SortBy         string   `json:"sort_by" jsonschema_description:"排序：recommended、price_asc、price_desc、star_desc 或 newest"`
}

type orderSummary struct {
	OrderSN      string  `json:"order_sn"`
	Title        string  `json:"title"`
	State        string  `json:"state"`
	TotalYuan    float64 `json:"total_yuan"`
	CheckInDate  string  `json:"check_in_date"`
	CheckOutDate string  `json:"check_out_date"`
}

type listOrdersOutput struct {
	Orders []orderSummary `json:"orders"`
}

type orderDetailOutput struct {
	OrderSN      string  `json:"order_sn"`
	HomestayID   int64   `json:"homestay_id"`
	Title        string  `json:"title"`
	State        string  `json:"state"`
	TotalYuan    float64 `json:"total_yuan"`
	CheckInDate  string  `json:"check_in_date"`
	CheckOutDate string  `json:"check_out_date"`
	People       int64   `json:"people"`
	Remark       string  `json:"remark,omitempty"`
}

type homestayDetailOutput struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	City          string  `json:"city"`
	Tags          string  `json:"tags"`
	People        int64   `json:"people"`
	PriceYuan     float64 `json:"price_yuan"`
	FoodPriceYuan float64 `json:"food_price_yuan"`
}

type recommendation struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	SubTitle         string  `json:"sub_title,omitempty"`
	City             string  `json:"city"`
	Tags             string  `json:"tags"`
	Star             float64 `json:"star"`
	People           int64   `json:"people"`
	NightlyPriceYuan float64 `json:"nightly_price_yuan"`
	MarketPriceYuan  float64 `json:"market_price_yuan"`
	FoodPriceYuan    float64 `json:"food_price_yuan"`
	Description      string  `json:"description,omitempty"`
}

type recommendHomestaysOutput struct {
	TotalFound    int64            `json:"total_found"`
	ReturnedCount int              `json:"returned_count"`
	Candidates    []recommendation `json:"candidates"`
	Notice        string           `json:"notice"`
}

func buildTools(orders orderReader, homestays homestayReader, catalog catalogSearcher, userID int64) ([]tool.BaseTool, error) {
	listTool, err := toolutils.InferTool("list_my_orders", "查询当前登录用户最近的订单；问题涉及我的订单、最近订单或订单状态时使用", func(ctx context.Context, input listOrdersInput) (listOrdersOutput, error) {
		pageSize := input.PageSize
		if pageSize < 1 || pageSize > 10 {
			pageSize = 5
		}
		items, listErr := orders.List(ctx, userID, 0, pageSize, 99)
		if listErr != nil {
			return listOrdersOutput{}, listErr
		}
		out := listOrdersOutput{Orders: make([]orderSummary, 0, len(items))}
		for _, item := range items {
			out.Orders = append(out.Orders, orderSummary{
				OrderSN: item.SN, Title: item.Title, State: orderState(item.TradeState),
				TotalYuan:   shared.FenToYuan(item.OrderTotalPrice),
				CheckInDate: formatDate(item.LiveStartDate), CheckOutDate: formatDate(item.LiveEndDate),
			})
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}

	detailTool, err := toolutils.InferTool("get_my_order", "根据订单号查询当前登录用户自己的订单详情；工具会执行订单归属校验", func(ctx context.Context, input orderDetailInput) (orderDetailOutput, error) {
		item, detailErr := orders.Detail(ctx, userID, input.OrderSN)
		if detailErr != nil {
			return orderDetailOutput{}, detailErr
		}
		return orderDetailOutput{
			OrderSN: item.SN, HomestayID: item.HomestayID, Title: item.Title,
			State: orderState(item.TradeState), TotalYuan: shared.FenToYuan(item.OrderTotalPrice),
			CheckInDate: formatDate(item.LiveStartDate), CheckOutDate: formatDate(item.LiveEndDate),
			People: item.LivePeopleNum, Remark: item.Remark,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	homestayTool, err := toolutils.InferTool("get_homestay", "根据民宿 ID 查询公开的民宿标题、城市、标签、容量和价格", func(ctx context.Context, input homestayDetailInput) (homestayDetailOutput, error) {
		item, detailErr := homestays.Homestay(ctx, input.HomestayID)
		if detailErr != nil {
			return homestayDetailOutput{}, detailErr
		}
		return homestayDetailOutput{
			ID: item.ID, Title: item.Title, City: item.City, Tags: item.Tags, People: item.PeopleNum,
			PriceYuan: shared.FenToYuan(item.HomestayPrice), FoodPriceYuan: shared.FenToYuan(item.FoodPrice),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	recommendTool, err := toolutils.InferTool("recommend_homestays", "根据用户的城市、每晚预算、人数、标签、评分和关键词，从真实民宿索引中检索并返回最多 5 个候选；推荐住宿时使用", func(ctx context.Context, input recommendHomestaysInput) (recommendHomestaysOutput, error) {
		if catalog == nil {
			return recommendHomestaysOutput{}, fmt.Errorf("homestay catalog is unavailable")
		}
		if input.MinNightlyYuan < 0 || input.MaxNightlyYuan < 0 || (input.MaxNightlyYuan > 0 && input.MinNightlyYuan > input.MaxNightlyYuan) {
			return recommendHomestaysOutput{}, fmt.Errorf("nightly budget range is invalid")
		}
		if input.Guests < 0 {
			return recommendHomestaysOutput{}, fmt.Errorf("guest count is invalid")
		}
		if input.MinStar < 0 || input.MinStar > 5 {
			return recommendHomestaysOutput{}, fmt.Errorf("minimum star must be between 0 and 5")
		}
		tags := make([]string, 0, min(len(input.Tags), 5))
		for _, tag := range input.Tags {
			if value := strings.TrimSpace(tag); value != "" && len(tags) < 5 {
				tags = append(tags, value)
			}
		}
		sortBy := []string{"star_desc"}
		switch input.SortBy {
		case "price_asc", "price_desc", "star_desc", "newest":
			sortBy = []string{input.SortBy}
		case "", "recommended":
		default:
			return recommendHomestaysOutput{}, fmt.Errorf("unsupported sort order %q", input.SortBy)
		}
		result, searchErr := catalog.Search(ctx, search.Query{
			Keyword: strings.TrimSpace(input.Keyword), City: strings.TrimSpace(input.City),
			MinPrice: search.YuanToFen(input.MinNightlyYuan), MaxPrice: search.YuanToFen(input.MaxNightlyYuan),
			Tags: tags, MinStar: input.MinStar, SortBy: sortBy, Page: 1, PageSize: 50,
		})
		if searchErr != nil {
			return recommendHomestaysOutput{}, searchErr
		}
		out := recommendHomestaysOutput{
			TotalFound: result.Total, Candidates: make([]recommendation, 0, 5),
			Notice: "候选仅按房源属性匹配，指定日期库存和最终价格需在下单时确认。",
		}
		for _, item := range result.Items {
			if input.Guests > 0 && item.PeopleNum < input.Guests {
				continue
			}
			out.Candidates = append(out.Candidates, recommendation{
				ID: item.ID, Title: item.Title, SubTitle: item.SubTitle, City: item.City,
				Tags: item.Tags, Star: item.Star, People: item.PeopleNum,
				NightlyPriceYuan: shared.FenToYuan(item.HomestayPrice),
				MarketPriceYuan:  shared.FenToYuan(item.MarketHomestayPrice),
				FoodPriceYuan:    shared.FenToYuan(item.FoodPrice),
				Description:      truncateText(item.Info, 160),
			})
			if len(out.Candidates) == 5 {
				break
			}
		}
		out.ReturnedCount = len(out.Candidates)
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{listTool, detailTool, homestayTool, recommendTool}, nil
}

func truncateText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func orderState(state int64) string {
	switch state {
	case order.TradeStateCancel:
		return "已取消"
	case order.TradeStateWaitPay:
		return "待支付"
	case order.TradeStateWaitUse:
		return "待入住"
	case order.TradeStateUsed:
		return "已完成"
	case order.TradeStateRefund:
		return "已退款"
	case order.TradeStateExpire:
		return "已过期"
	default:
		return "未知状态"
	}
}

var _ homestayReader = (*travel.Service)(nil)
var _ catalogSearcher = (*search.Service)(nil)
