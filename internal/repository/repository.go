package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var (
	ErrNotFound       = sql.ErrNoRows
	ErrSeckillSoldOut = errors.New("seckill sold out")
)

type Repository struct {
	UserDB         *sql.DB
	TravelDB       *sql.DB
	OrderDB        *sql.DB
	PaymentDB      *sql.DB
	Redis          *redis.Client
	userFlight     singleflight.Group
	homestayFlight singleflight.Group
}

func Open(ctx context.Context, c config.Config) (*Repository, error) {
	open := func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(30)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(3 * time.Minute)
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	}
	userDB, err := open(c.UserDSN)
	if err != nil {
		return nil, fmt.Errorf("open user db: %w", err)
	}
	travelDB, err := open(c.TravelDSN)
	if err != nil {
		_ = userDB.Close()
		return nil, fmt.Errorf("open travel db: %w", err)
	}
	orderDB, err := open(c.OrderDSN)
	if err != nil {
		_ = userDB.Close()
		_ = travelDB.Close()
		return nil, fmt.Errorf("open order db: %w", err)
	}
	paymentDB, err := open(c.PaymentDSN)
	if err != nil {
		_ = userDB.Close()
		_ = travelDB.Close()
		_ = orderDB.Close()
		return nil, fmt.Errorf("open payment db: %w", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("open redis: %w", err)
	}
	return &Repository{UserDB: userDB, TravelDB: travelDB, OrderDB: orderDB, PaymentDB: paymentDB, Redis: rdb}, nil
}

func (r *Repository) Close() {
	_ = r.UserDB.Close()
	_ = r.TravelDB.Close()
	_ = r.OrderDB.Close()
	_ = r.PaymentDB.Close()
	_ = r.Redis.Close()
}

func scanUser(row interface{ Scan(...any) error }) (*model.User, error) {
	var v model.User
	err := row.Scan(&v.ID, &v.CreateTime, &v.UpdateTime, &v.DeleteTime, &v.DelState, &v.Version, &v.Mobile, &v.Password, &v.Nickname, &v.Sex, &v.Avatar, &v.Info)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

const userFields = "id,create_time,update_time,delete_time,del_state,version,mobile,password,nickname,sex,avatar,info"

func (r *Repository) UserByMobile(ctx context.Context, mobile string) (*model.User, error) {
	return scanUser(r.UserDB.QueryRowContext(ctx, "SELECT "+userFields+" FROM user WHERE mobile=? AND del_state=0 LIMIT 1", mobile))
}

func (r *Repository) UserByID(ctx context.Context, id int64) (*model.User, error) {
	key := fmt.Sprintf("gin:looklook:v2:user:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v model.User
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.userFlight.Do(key, func() (any, error) {
		v, err := scanUser(r.UserDB.QueryRowContext(ctx, "SELECT "+userFields+" FROM user WHERE id=? AND del_state=0 LIMIT 1", id))
		if err != nil {
			return nil, err
		}
		if data, err := json.Marshal(v); err == nil {
			_ = r.Redis.Set(ctx, key, data, 10*time.Minute).Err()
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return loaded.(*model.User), nil
}

func (r *Repository) UserAuthByUser(ctx context.Context, userID int64, authType string) (*model.UserAuth, error) {
	var v model.UserAuth
	err := r.UserDB.QueryRowContext(ctx, "SELECT id,user_id,auth_key,auth_type FROM user_auth WHERE user_id=? AND auth_type=? AND del_state=0 LIMIT 1", userID, authType).Scan(&v.ID, &v.UserID, &v.AuthKey, &v.AuthType)
	return &v, err
}
func (r *Repository) UserAuthByKey(ctx context.Context, authType, authKey string) (*model.UserAuth, error) {
	var v model.UserAuth
	err := r.UserDB.QueryRowContext(ctx, "SELECT id,user_id,auth_key,auth_type FROM user_auth WHERE auth_type=? AND auth_key=? AND del_state=0 LIMIT 1", authType, authKey).Scan(&v.ID, &v.UserID, &v.AuthKey, &v.AuthType)
	return &v, err
}

func (r *Repository) CreateUser(ctx context.Context, user *model.User, auth *model.UserAuth) (int64, error) {
	tx, err := r.UserDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT INTO user(delete_time,del_state,version,mobile,password,nickname,sex,avatar,info) VALUES(FROM_UNIXTIME(0),0,0,?,?,?,?,?,?)", user.Mobile, user.Password, user.Nickname, user.Sex, user.Avatar, user.Info)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO user_auth(delete_time,del_state,version,user_id,auth_key,auth_type) VALUES(FROM_UNIXTIME(0),0,0,?,?,?)", id, auth.AuthKey, auth.AuthType)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

const homestayFields = "id,version,title,sub_title,banner,info,city,tags,star,latitude,longitude,people_num,homestay_business_id,user_id,row_state,row_type,food_info,food_price,homestay_price,market_homestay_price"
const homestayJoinFields = "h.id,h.version,h.title,h.sub_title,h.banner,h.info,h.city,h.tags,h.star,h.latitude,h.longitude,h.people_num,h.homestay_business_id,h.user_id,h.row_state,h.row_type,h.food_info,h.food_price,h.homestay_price,h.market_homestay_price"

func scanHomestay(row interface{ Scan(...any) error }) (*model.Homestay, error) {
	var v model.Homestay
	err := row.Scan(&v.ID, &v.Version, &v.Title, &v.SubTitle, &v.Banner, &v.Info, &v.City, &v.Tags, &v.Star, &v.Latitude, &v.Longitude, &v.PeopleNum, &v.HomestayBusinessID, &v.UserID, &v.RowState, &v.RowType, &v.FoodInfo, &v.FoodPrice, &v.HomestayPrice, &v.MarketHomestayPrice)
	return &v, err
}

func (r *Repository) HomestayByID(ctx context.Context, id int64) (*model.Homestay, error) {
	key := fmt.Sprintf("gin:looklook:v2:homestay:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v model.Homestay
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.homestayFlight.Do(key, func() (any, error) {
		v, err := scanHomestay(r.TravelDB.QueryRowContext(ctx, "SELECT "+homestayFields+" FROM homestay WHERE id=? AND del_state=0 LIMIT 1", id))
		if err != nil {
			return nil, err
		}
		if data, err := json.Marshal(v); err == nil {
			_ = r.Redis.Set(ctx, key, data, 10*time.Minute).Err()
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return loaded.(*model.Homestay), nil
}

func scanHomestayRows(rows *sql.Rows) ([]model.Homestay, error) {
	var out []model.Homestay
	defer rows.Close()
	for rows.Next() {
		v, err := scanHomestay(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) HomestaysByActivity(ctx context.Context, rowType string, page, pageSize int64) ([]model.Homestay, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	rows, err := r.TravelDB.QueryContext(ctx, "SELECT "+homestayJoinFields+" FROM homestay h JOIN homestay_activity a ON a.data_id=h.id WHERE a.row_type=? AND a.row_status=1 AND a.del_state=0 AND h.del_state=0 ORDER BY a.data_id DESC LIMIT ? OFFSET ?", rowType, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return scanHomestayRows(rows)
}
func (r *Repository) HomestaysByBusiness(ctx context.Context, businessID, lastID, pageSize int64) ([]model.Homestay, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	query := "SELECT " + homestayFields + " FROM homestay WHERE homestay_business_id=? AND del_state=0"
	args := []any{businessID}
	if lastID > 0 {
		query += " AND id<?"
		args = append(args, lastID)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, pageSize)
	rows, err := r.TravelDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanHomestayRows(rows)
}
func (r *Repository) GuessHomestays(ctx context.Context) ([]model.Homestay, error) {
	rows, err := r.TravelDB.QueryContext(ctx, "SELECT "+homestayFields+" FROM homestay WHERE del_state=0 ORDER BY id DESC LIMIT 5")
	if err != nil {
		return nil, err
	}
	return scanHomestayRows(rows)
}

func (r *Repository) Businesses(ctx context.Context, lastID, pageSize int64) ([]model.HomestayBusiness, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	q := "SELECT id,title,user_id,info,boss_info,row_state,star,tags,cover,header_img FROM homestay_business WHERE del_state=0"
	args := []any{}
	if lastID > 0 {
		q += " AND id<?"
		args = append(args, lastID)
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, pageSize)
	rows, err := r.TravelDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.HomestayBusiness
	for rows.Next() {
		var v model.HomestayBusiness
		if err := rows.Scan(&v.ID, &v.Title, &v.UserID, &v.Info, &v.BossInfo, &v.RowState, &v.Star, &v.Tags, &v.Cover, &v.HeaderImg); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) BusinessByID(ctx context.Context, id int64) (*model.HomestayBusiness, error) {
	var v model.HomestayBusiness
	err := r.TravelDB.QueryRowContext(ctx, "SELECT id,title,user_id,info,boss_info,row_state,star,tags,cover,header_img FROM homestay_business WHERE id=? AND del_state=0 LIMIT 1", id).Scan(&v.ID, &v.Title, &v.UserID, &v.Info, &v.BossInfo, &v.RowState, &v.Star, &v.Tags, &v.Cover, &v.HeaderImg)
	return &v, err
}
func (r *Repository) GoodBossUserIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.TravelDB.QueryContext(ctx, "SELECT data_id FROM homestay_activity WHERE row_type='goodBusiness' AND row_status=1 AND del_state=0 ORDER BY data_id DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (r *Repository) Comments(ctx context.Context, lastID, pageSize int64) ([]model.HomestayComment, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	q := "SELECT id,homestay_id,user_id,content,star FROM homestay_comment WHERE del_state=0"
	args := []any{}
	if lastID > 0 {
		q += " AND id<?"
		args = append(args, lastID)
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, pageSize)
	rows, err := r.TravelDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.HomestayComment
	for rows.Next() {
		var v model.HomestayComment
		if err := rows.Scan(&v.ID, &v.HomestayID, &v.UserID, &v.Content, &v.Star); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

const orderFields = "id,create_time,update_time,delete_time,del_state,version,sn,user_id,homestay_id,title,sub_title,cover,info,people_num,row_type,need_food,food_info,food_price,homestay_price,market_homestay_price,homestay_business_id,homestay_user_id,live_start_date,live_end_date,live_people_num,trade_state,trade_code,remark,order_total_price,food_total_price,homestay_total_price"

func scanOrder(row interface{ Scan(...any) error }) (*model.HomestayOrder, error) {
	var v model.HomestayOrder
	err := row.Scan(&v.ID, &v.CreateTime, &v.UpdateTime, &v.DeleteTime, &v.DelState, &v.Version, &v.SN, &v.UserID, &v.HomestayID, &v.Title, &v.SubTitle, &v.Cover, &v.Info, &v.PeopleNum, &v.RowType, &v.NeedFood, &v.FoodInfo, &v.FoodPrice, &v.HomestayPrice, &v.MarketHomestayPrice, &v.HomestayBusinessID, &v.HomestayUserID, &v.LiveStartDate, &v.LiveEndDate, &v.LivePeopleNum, &v.TradeState, &v.TradeCode, &v.Remark, &v.OrderTotalPrice, &v.FoodTotalPrice, &v.HomestayTotalPrice)
	return &v, err
}

func (r *Repository) CreateOrder(ctx context.Context, v *model.HomestayOrder) error {
	res, err := insertOrder(ctx, r.OrderDB, v)
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertOrder(ctx context.Context, exec sqlExecutor, v *model.HomestayOrder) (sql.Result, error) {
	return exec.ExecContext(ctx, "INSERT INTO homestay_order(delete_time,del_state,version,sn,user_id,homestay_id,title,sub_title,cover,info,people_num,row_type,need_food,food_info,food_price,homestay_price,market_homestay_price,homestay_business_id,homestay_user_id,live_start_date,live_end_date,live_people_num,trade_state,trade_code,remark,order_total_price,food_total_price,homestay_total_price) VALUES(FROM_UNIXTIME(0),0,0,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", v.SN, v.UserID, v.HomestayID, v.Title, v.SubTitle, v.Cover, v.Info, v.PeopleNum, v.RowType, v.NeedFood, v.FoodInfo, v.FoodPrice, v.HomestayPrice, v.MarketHomestayPrice, v.HomestayBusinessID, v.HomestayUserID, v.LiveStartDate, v.LiveEndDate, v.LivePeopleNum, v.TradeState, v.TradeCode, v.Remark, v.OrderTotalPrice, v.FoodTotalPrice, v.HomestayTotalPrice)
}
func (r *Repository) OrderBySN(ctx context.Context, sn string) (*model.HomestayOrder, error) {
	return scanOrder(r.OrderDB.QueryRowContext(ctx, "SELECT "+orderFields+" FROM homestay_order WHERE sn=? AND del_state=0 LIMIT 1", sn))
}
func (r *Repository) OrdersByUser(ctx context.Context, userID, lastID, pageSize, tradeState int64) ([]model.HomestayOrder, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	q := "SELECT " + orderFields + " FROM homestay_order WHERE user_id=? AND del_state=0"
	args := []any{userID}
	if lastID > 0 {
		q += " AND id<?"
		args = append(args, lastID)
	}
	if tradeState >= -1 && tradeState <= 4 {
		q += " AND trade_state=?"
		args = append(args, tradeState)
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, pageSize)
	rows, err := r.OrderDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.HomestayOrder
	for rows.Next() {
		v, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (r *Repository) UpdateOrderState(ctx context.Context, id, oldVersion, newState int64) error {
	res, err := r.OrderDB.ExecContext(ctx, "UPDATE homestay_order SET trade_state=?,version=version+1 WHERE id=? AND version=?", newState, id, oldVersion)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("optimistic lock conflict")
	}
	return nil
}

const seckillActivityFields = "id,homestay_id,title,price,stock,sold_count,start_time,end_time,status"

func scanSeckillActivity(row interface{ Scan(...any) error }) (*model.SeckillActivity, error) {
	var v model.SeckillActivity
	err := row.Scan(&v.ID, &v.HomestayID, &v.Title, &v.Price, &v.Stock, &v.SoldCount, &v.StartTime, &v.EndTime, &v.Status)
	return &v, err
}

func (r *Repository) ActiveSeckillActivities(ctx context.Context) ([]model.SeckillActivity, error) {
	rows, err := r.OrderDB.QueryContext(ctx, "SELECT "+seckillActivityFields+" FROM seckill_activity WHERE status=1 AND end_time>NOW() ORDER BY start_time,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SeckillActivity
	for rows.Next() {
		v, err := scanSeckillActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) SeckillActivityByID(ctx context.Context, id int64) (*model.SeckillActivity, error) {
	return scanSeckillActivity(r.OrderDB.QueryRowContext(ctx, "SELECT "+seckillActivityFields+" FROM seckill_activity WHERE id=? LIMIT 1", id))
}

// CreateSeckillOrder uses MySQL as the final oversell and idempotency barrier.
func (r *Repository) CreateSeckillOrder(ctx context.Context, reservation model.SeckillReservation, order *model.HomestayOrder) (string, bool, bool, error) {
	tx, err := r.OrderDB.BeginTx(ctx, nil)
	if err != nil {
		return "", false, false, err
	}
	defer tx.Rollback()
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT order_sn FROM seckill_order WHERE reservation_sn=? LIMIT 1", reservation.ReservationSN).Scan(&existing)
	if err == nil && existing != "" {
		return existing, true, false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", false, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO seckill_order(reservation_sn,activity_id,user_id,status) VALUES(?,?,?,0)", reservation.ReservationSN, reservation.ActivityID, reservation.UserID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
			return "", false, false, err
		}
		_ = tx.Rollback()
		if err = r.OrderDB.QueryRowContext(ctx, "SELECT order_sn FROM seckill_order WHERE activity_id=? AND user_id=? LIMIT 1", reservation.ActivityID, reservation.UserID).Scan(&existing); err != nil {
			return "", false, false, err
		}
		if existing == "" {
			return "", false, false, errors.New("seckill order is still processing")
		}
		return existing, true, true, nil
	}
	res, err := tx.ExecContext(ctx, "UPDATE seckill_activity SET sold_count=sold_count+1 WHERE id=? AND status=1 AND sold_count<stock", reservation.ActivityID)
	if err != nil {
		return "", false, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", false, false, ErrSeckillSoldOut
	}
	res, err = insertOrder(ctx, tx, order)
	if err != nil {
		return "", false, false, err
	}
	order.ID, _ = res.LastInsertId()
	if _, err = tx.ExecContext(ctx, "UPDATE seckill_order SET status=1,order_sn=? WHERE reservation_sn=?", order.SN, reservation.ReservationSN); err != nil {
		return "", false, false, err
	}
	if err = tx.Commit(); err != nil {
		return "", false, false, err
	}
	return order.SN, false, false, nil
}

const paymentFields = "id,sn,create_time,update_time,delete_time,del_state,version,user_id,pay_mode,trade_type,trade_state,pay_total,transaction_id,trade_state_desc,order_sn,service_type,pay_status,pay_time"

func scanPayment(row interface{ Scan(...any) error }) (*model.ThirdPayment, error) {
	var v model.ThirdPayment
	err := row.Scan(&v.ID, &v.SN, &v.CreateTime, &v.UpdateTime, &v.DeleteTime, &v.DelState, &v.Version, &v.UserID, &v.PayMode, &v.TradeType, &v.TradeState, &v.PayTotal, &v.TransactionID, &v.TradeStateDesc, &v.OrderSN, &v.ServiceType, &v.PayStatus, &v.PayTime)
	return &v, err
}
func (r *Repository) CreatePayment(ctx context.Context, v *model.ThirdPayment) error {
	_, err := r.PaymentDB.ExecContext(ctx, "INSERT INTO third_payment(sn,delete_time,user_id,pay_mode,trade_type,trade_state,pay_total,transaction_id,trade_state_desc,order_sn,service_type,pay_status,pay_time) VALUES(?,FROM_UNIXTIME(0),?,?,?,?,?,?,?,?,?,?,FROM_UNIXTIME(0))", v.SN, v.UserID, v.PayMode, v.TradeType, v.TradeState, v.PayTotal, v.TransactionID, v.TradeStateDesc, v.OrderSN, v.ServiceType, v.PayStatus)
	return err
}
func (r *Repository) PaymentBySN(ctx context.Context, sn string) (*model.ThirdPayment, error) {
	return scanPayment(r.PaymentDB.QueryRowContext(ctx, "SELECT "+paymentFields+" FROM third_payment WHERE sn=? AND del_state=0 LIMIT 1", sn))
}
func (r *Repository) PaymentByOrder(ctx context.Context, orderSN string) (*model.ThirdPayment, error) {
	return scanPayment(r.PaymentDB.QueryRowContext(ctx, "SELECT "+paymentFields+" FROM third_payment WHERE order_sn=? AND pay_status IN (1,2) AND del_state=0 ORDER BY id DESC LIMIT 1", orderSN))
}
func (r *Repository) UpdatePayment(ctx context.Context, v *model.ThirdPayment) error {
	res, err := r.PaymentDB.ExecContext(ctx, "UPDATE third_payment SET version=version+1,trade_type=?,trade_state=?,transaction_id=?,trade_state_desc=?,pay_status=?,pay_time=? WHERE id=? AND version=?", v.TradeType, v.TradeState, v.TransactionID, v.TradeStateDesc, v.PayStatus, v.PayTime, v.ID, v.Version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("optimistic lock conflict")
	}
	v.Version++
	return nil
}

// UpdatePaymentWithOutbox commits the aggregate update and integration event atomically.
func (r *Repository) UpdatePaymentWithOutbox(ctx context.Context, v *model.ThirdPayment, event model.OutboxEvent) error {
	tx, err := r.PaymentDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE third_payment SET version=version+1,trade_type=?,trade_state=?,transaction_id=?,trade_state_desc=?,pay_status=?,pay_time=? WHERE id=? AND version=?", v.TradeType, v.TradeState, v.TransactionID, v.TradeStateDesc, v.PayStatus, v.PayTime, v.ID, v.Version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("optimistic lock conflict")
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO event_outbox(event_key,topic,message_key,payload) VALUES(?,?,?,?)", event.EventKey, event.Topic, event.MessageKey, event.Payload); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	v.Version++
	return nil
}

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if limit < 1 {
		limit = 100
	}
	rows, err := r.PaymentDB.QueryContext(ctx, "SELECT id,event_key,topic,message_key,payload,retry_count FROM event_outbox WHERE status=0 AND next_retry_at<=NOW() ORDER BY id LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OutboxEvent
	for rows.Next() {
		var v model.OutboxEvent
		if err := rows.Scan(&v.ID, &v.EventKey, &v.Topic, &v.MessageKey, &v.Payload, &v.RetryCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id int64) error {
	_, err := r.PaymentDB.ExecContext(ctx, "UPDATE event_outbox SET status=1,published_at=NOW() WHERE id=? AND status=0", id)
	return err
}

func (r *Repository) RetryOutbox(ctx context.Context, id int64) error {
	_, err := r.PaymentDB.ExecContext(ctx, "UPDATE event_outbox SET retry_count=retry_count+1,next_retry_at=DATE_ADD(NOW(), INTERVAL LEAST(60, POW(2, LEAST(retry_count,5))) SECOND) WHERE id=? AND status=0", id)
	return err
}
