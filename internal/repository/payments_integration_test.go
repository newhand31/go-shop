//go:build integration

package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// stubAPI stands in for the rest of the API: these tests only exercise the callback route.
type stubAPI struct{}

func (stubAPI) ServeHTTP(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }

type paymentEnv struct {
	pool    *pgxpool.Pool
	verify  *ecpay.Verifier
	server  http.Handler
	orderID int
	total   int
}

// newPaymentEnv migrates a fresh database, places one order (a keyboard), and wires the real callback stack:
// handler, service, repository, Postgres.
func newPaymentEnv(t *testing.T) paymentEnv {
	t.Helper()
	pool := openMigrated(t)
	order, err := repository.NewOrders(pool).Place(context.Background(), []repository.OrderLine{{ProductID: 1, Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	v := ecpay.NewVerifier(ecpay.SandboxMerchantID, ecpay.SandboxHashKey, ecpay.SandboxHashIV)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewPaymentService(v, repository.NewPayments(pool), logger)
	return paymentEnv{pool: pool, verify: v, server: handler.WithPaymentCallback(stubAPI{}, svc, logger), orderID: order.ID, total: order.TotalTWD}
}

func (e paymentEnv) callback(tradeNo string, amount int) url.Values {
	return e.verify.Sign(url.Values{
		"MerchantID": {ecpay.SandboxMerchantID}, "MerchantTradeNo": {ecpay.OrderRef(e.orderID)}, "TradeNo": {tradeNo},
		"TradeAmt": {strconv.Itoa(amount)}, "RtnCode": {"1"}, "RtnMsg": {"Succeeded"},
	})
}

func (e paymentEnv) post(form url.Values) (int, string) {
	req := httptest.NewRequest(http.MethodPost, handler.ECPayCallbackPath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.server.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func (e paymentEnv) state(t *testing.T) (payments int, paid bool) {
	t.Helper()
	ctx := context.Background()
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM payments`).Scan(&payments); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT paid_at IS NOT NULL FROM orders WHERE id = $1`, e.orderID).Scan(&paid); err != nil {
		t.Fatal(err)
	}
	return
}

// The incident, fixed: ECPay resends the same callback (our reply was slow, or it just does). Many deliveries arrive at
// once; every one gets "1|OK", and the payment is recorded exactly once.
func TestPayments_ResentCallbackIsRecordedOnce(t *testing.T) {
	e := newPaymentEnv(t)
	form := e.callback("2611141530450001", e.total)

	const deliveries = 10
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < deliveries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if code, body := e.post(form); code != http.StatusOK || body != "1|OK" {
				t.Errorf("delivery got %d %q, want 200 and 1|OK", code, body)
			}
		}()
	}
	close(start)
	wg.Wait()

	if payments, paid := e.state(t); payments != 1 || !paid {
		t.Fatalf("payments=%d paid=%v, want 1 and true", payments, paid)
	}
}

// Two different payments (different TradeNo) are two different facts; only a repeat of the same one is dropped.
func TestPayments_DifferentTradeNoIsNotADuplicate(t *testing.T) {
	e := newPaymentEnv(t)
	for _, tradeNo := range []string{"A1", "A2"} {
		if code, _ := e.post(e.callback(tradeNo, e.total)); code != http.StatusOK {
			t.Fatalf("callback %s status = %d", tradeNo, code)
		}
	}
	if payments, _ := e.state(t); payments != 2 {
		t.Fatalf("payments = %d, want 2", payments)
	}
}

// The key lives in the database: even code that skips our service cannot insert the same TradeNo twice.
func TestPayments_UniqueKeyIsEnforcedByTheDatabase(t *testing.T) {
	e := newPaymentEnv(t)
	insert := func() error {
		_, err := e.pool.Exec(context.Background(), `INSERT INTO payments (order_id, trade_no, amount_twd) VALUES ($1, 'K1', 1)`, e.orderID)
		return err
	}
	if err := insert(); err != nil {
		t.Fatal(err)
	}
	var pgErr *pgconn.PgError
	if err := insert(); !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("second insert error = %v, want unique_violation 23505", err)
	}
}

func TestPayments_ForgedAndMismatchedCallbacksRecordNothing(t *testing.T) {
	e := newPaymentEnv(t)

	forged := e.callback("F1", e.total)
	forged.Set("TradeAmt", "1") // tampered after signing
	wrongAmount := e.callback("F2", e.total-1)
	unknownOrder := e.verify.Sign(url.Values{
		"MerchantID": {ecpay.SandboxMerchantID}, "MerchantTradeNo": {ecpay.OrderRef(99999)}, "TradeNo": {"F3"},
		"TradeAmt": {strconv.Itoa(e.total)}, "RtnCode": {"1"},
	})
	for name, form := range map[string]url.Values{"forged": forged, "wrong amount": wrongAmount, "unknown order": unknownOrder} {
		if code, body := e.post(form); code != http.StatusBadRequest || body == "1|OK" {
			t.Errorf("%s: got %d %q, want 400 and not 1|OK", name, code, body)
		}
	}
	if payments, paid := e.state(t); payments != 0 || paid {
		t.Fatalf("payments=%d paid=%v, want nothing recorded", payments, paid)
	}
}
