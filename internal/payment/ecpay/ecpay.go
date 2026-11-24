// Package ecpay verifies and parses the server-to-server callback ECPay (綠界) sends after a payment.
//
// It is written by hand on purpose: the whole protocol is "sort the fields, glue them, URL-encode, lower-case, SHA-256",
// and seeing those lines is worth more than importing a client. It uses only the standard library.
package ecpay

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Sandbox credentials from ECPay's public developer documentation. They are meant for testing and are safe to publish;
// never use them for real money.
const (
	SandboxMerchantID = "3002607"
	SandboxHashKey    = "pwFHCqoQZGmho4w6"
	SandboxHashIV     = "EkRm7iFT261dpevs"
)

// CheckMacField is the name of the signature field in a callback.
const CheckMacField = "CheckMacValue"

// Sentinel errors. Callers use errors.Is to tell "forged" from "malformed".
var (
	ErrBadSignature = errors.New("ecpay: CheckMacValue does not match")
	ErrMalformed    = errors.New("ecpay: malformed callback")
)

// Verifier holds the merchant's secrets.
type Verifier struct {
	merchantID string
	hashKey    string
	hashIV     string
}

// NewVerifier builds a Verifier for one merchant.
func NewVerifier(merchantID, hashKey, hashIV string) *Verifier {
	return &Verifier{merchantID: merchantID, hashKey: hashKey, hashIV: hashIV}
}

// CheckMacValue computes the signature of params (any CheckMacValue entry in params is ignored).
//
// ECPay's recipe, in order:
//  1. sort the fields by name, A–Z, ignoring case;
//  2. join them as k=v&k=v and wrap with HashKey=...& in front and &HashIV=... at the back;
//  3. URL-encode the whole string (.NET style, so a space is +);
//  4. lower-case everything;
//  5. put back ! * ( ), which .NET's encoder leaves alone but Go's escapes (- _ . are left alone by both);
//  6. SHA-256, hex, upper-case.
func (v *Verifier) CheckMacValue(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != CheckMacField {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := strings.ToLower(keys[i]), strings.ToLower(keys[j])
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})

	var sb strings.Builder
	sb.WriteString("HashKey=" + v.hashKey)
	for _, k := range keys {
		sb.WriteString("&" + k + "=" + params.Get(k))
	}
	sb.WriteString("&HashIV=" + v.hashIV)

	encoded := strings.ToLower(url.QueryEscape(sb.String()))
	encoded = strings.NewReplacer("%21", "!", "%2a", "*", "%28", "(", "%29", ")").Replace(encoded)

	sum := sha256.Sum256([]byte(encoded))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// Callback is the part of ECPay's payment notification the shop cares about.
type Callback struct {
	MerchantTradeNo string // our own order reference, sent to ECPay when the payment was created
	TradeNo         string // ECPay's transaction number: unique per payment, identical on every resend
	AmountTWD       int
	Paid            bool // RtnCode == 1
	RtnMsg          string
}

// Verify checks the signature first, and only then reads the fields: an unsigned body is attacker input.
func (v *Verifier) Verify(params url.Values) (Callback, error) {
	got := params.Get(CheckMacField)
	if got == "" {
		return Callback{}, fmt.Errorf("%w: missing %s", ErrBadSignature, CheckMacField)
	}
	want := v.CheckMacValue(params)
	// Constant-time compare, so response timing does not leak how many leading characters matched.
	if subtle.ConstantTimeCompare([]byte(strings.ToUpper(got)), []byte(want)) != 1 {
		return Callback{}, ErrBadSignature
	}
	if id := params.Get("MerchantID"); id != v.merchantID {
		return Callback{}, fmt.Errorf("%w: unexpected MerchantID", ErrMalformed)
	}
	amount, err := strconv.Atoi(params.Get("TradeAmt"))
	if err != nil || amount < 0 || amount > math.MaxInt32 {
		return Callback{}, fmt.Errorf("%w: bad TradeAmt %q", ErrMalformed, params.Get("TradeAmt"))
	}
	cb := Callback{
		MerchantTradeNo: params.Get("MerchantTradeNo"),
		TradeNo:         params.Get("TradeNo"),
		AmountTWD:       amount,
		Paid:            params.Get("RtnCode") == "1",
		RtnMsg:          params.Get("RtnMsg"),
	}
	if cb.MerchantTradeNo == "" || cb.TradeNo == "" {
		return Callback{}, fmt.Errorf("%w: missing trade number", ErrMalformed)
	}
	return cb, nil
}

// Sign returns a copy of params with CheckMacValue filled in. The shop signs its checkout form with it (CheckoutFields);
// tests also use it to play ECPay's part and sign a callback.
func (v *Verifier) Sign(params url.Values) url.Values {
	out := url.Values{}
	for k, vals := range params {
		out[k] = append([]string(nil), vals...)
	}
	out.Set(CheckMacField, v.CheckMacValue(out))
	return out
}

// OrderRef is the MerchantTradeNo for an order id. ECPay allows at most 20 letters and digits.
func OrderRef(orderID int) string { return "GS" + strconv.Itoa(orderID) }

// ParseOrderRef is the inverse of OrderRef.
func ParseOrderRef(ref string) (int, error) {
	rest, ok := strings.CutPrefix(ref, "GS")
	if !ok {
		return 0, fmt.Errorf("%w: unknown MerchantTradeNo %q", ErrMalformed, ref)
	}
	id, err := strconv.Atoi(rest)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: unknown MerchantTradeNo %q", ErrMalformed, ref)
	}
	return id, nil
}

// Where ECPay's checkout page lives. The sandbox accepts only the public sandbox merchant; real money goes to the other.
const (
	sandboxCheckoutURL    = "https://payment-stage.ecpay.com.tw/Cashier/AioCheckOut/V5"
	productionCheckoutURL = "https://payment.ecpay.com.tw/Cashier/AioCheckOut/V5"
)

// CheckoutURL is the address the browser posts the checkout form to. The sandbox merchant id picks the sandbox page,
// so test credentials can never reach the real one.
func (v *Verifier) CheckoutURL() string {
	if v.merchantID == SandboxMerchantID {
		return sandboxCheckoutURL
	}
	return productionCheckoutURL
}

// CheckoutRequest is what the shop tells ECPay about one payment.
type CheckoutRequest struct {
	OrderID     int
	AmountTWD   int
	ItemName    string
	Description string
	ReturnURL   string    // where ECPay's server posts the result: our callback route
	Now         time.Time // becomes MerchantTradeDate
}

// taipei is ECPay's clock: MerchantTradeDate is in Taiwan time.
var taipei = time.FixedZone("CST", 8*60*60)

// CheckoutFields builds the signed form fields of ECPay's all-in-one checkout (EncryptType 1, SHA-256).
// It is the mirror image of Verify: the same CheckMacValue recipe, but here the shop signs and ECPay checks.
func (v *Verifier) CheckoutFields(r CheckoutRequest) url.Values {
	f := url.Values{}
	f.Set("MerchantID", v.merchantID)
	f.Set("MerchantTradeNo", OrderRef(r.OrderID))
	f.Set("MerchantTradeDate", r.Now.In(taipei).Format("2006/01/02 15:04:05"))
	f.Set("PaymentType", "aio")
	f.Set("TotalAmount", strconv.Itoa(r.AmountTWD))
	f.Set("TradeDesc", r.Description)
	f.Set("ItemName", r.ItemName)
	f.Set("ReturnURL", r.ReturnURL)
	f.Set("ChoosePayment", "ALL")
	f.Set("EncryptType", "1")
	return v.Sign(f)
}
