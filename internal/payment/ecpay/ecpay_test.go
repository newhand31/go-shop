package ecpay_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/newhand31/go-shop/internal/payment/ecpay"
)

func sandbox() *ecpay.Verifier {
	return ecpay.NewVerifier(ecpay.SandboxMerchantID, ecpay.SandboxHashKey, ecpay.SandboxHashIV)
}

func successCallback() url.Values {
	return url.Values{
		"MerchantID":      {ecpay.SandboxMerchantID},
		"MerchantTradeNo": {"GS42"},
		"StoreID":         {""},
		"RtnCode":         {"1"},
		"RtnMsg":          {"Succeeded"},
		"TradeNo":         {"2611141530450001"},
		"TradeAmt":        {"1290"},
		"PaymentDate":     {"2026/11/14 15:30:45"},
		"PaymentType":     {"Credit_CreditCard"},
		"TradeDate":       {"2026/11/14 15:30:12"},
		"SimulatePaid":    {"0"},
	}
}

// The expected values below were produced by a separate implementation (a throwaway Python script written straight
// from ECPay's published steps), so this test checks the Go code against the rules and not against itself.
func TestCheckMacValue_GoldenVectors(t *testing.T) {
	tests := []struct {
		name   string
		params url.Values
		want   string
	}{
		{"typical callback", successCallback(), "C1EC3DB54A0113B6594B21DD8739729403AC2FF517C667975B67DC77C2DFB85D"},
		{
			// Spaces, ! * ( ) ' @, a mixed-case key and an empty-case sort: the encoding rules in one place.
			"special characters and mixed-case keys",
			url.Values{
				"MerchantID":      {"3002607"},
				"ItemName":        {"Keyboard (Pro) *RGB* a@b"},
				"TradeDesc":       {"it's a test!"},
				"merchantTradeNo": {"x y"},
				"Zeta":            {"z"},
			},
			"0F000F09FD3ADC99062D133C0330C8E77D4BE680865DE8731BC35AB8B2349778",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sandbox().CheckMacValue(tt.params); got != tt.want {
				t.Errorf("CheckMacValue = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestCheckMacValue_IgnoresItsOwnField(t *testing.T) {
	p := successCallback()
	want := sandbox().CheckMacValue(p)
	p.Set(ecpay.CheckMacField, "whatever")
	if got := sandbox().CheckMacValue(p); got != want {
		t.Errorf("signature changed when CheckMacValue was present: %s vs %s", got, want)
	}
}

func TestVerify(t *testing.T) {
	signed := sandbox().Sign(successCallback())

	tamper := func(mutate func(url.Values)) url.Values {
		p := sandbox().Sign(successCallback())
		mutate(p)
		return p
	}

	tests := []struct {
		name    string
		params  url.Values
		wantErr error
	}{
		{"valid signature", signed, nil},
		{"lower-case signature is accepted", tamper(func(p url.Values) { p.Set("CheckMacValue", lower(p.Get("CheckMacValue"))) }), nil},
		{"amount changed after signing", tamper(func(p url.Values) { p.Set("TradeAmt", "1") }), ecpay.ErrBadSignature},
		{"order changed after signing", tamper(func(p url.Values) { p.Set("MerchantTradeNo", "GS43") }), ecpay.ErrBadSignature},
		{"extra field added after signing", tamper(func(p url.Values) { p.Set("Extra", "x") }), ecpay.ErrBadSignature},
		{"signature missing", tamper(func(p url.Values) { p.Del("CheckMacValue") }), ecpay.ErrBadSignature},
		{"signature garbage", tamper(func(p url.Values) { p.Set("CheckMacValue", "DEADBEEF") }), ecpay.ErrBadSignature},
		{"signed with another key", url.Values(ecpay.NewVerifier(ecpay.SandboxMerchantID, "otherkey", ecpay.SandboxHashIV).Sign(successCallback())), ecpay.ErrBadSignature},
		{"signed for another merchant", ecpay.NewVerifier("1111111", ecpay.SandboxHashKey, ecpay.SandboxHashIV).Sign(withMerchant("1111111")), ecpay.ErrMalformed},
		{"amount not a number", sandbox().Sign(with("TradeAmt", "12x")), ecpay.ErrMalformed},
		{"amount too large for the database column", sandbox().Sign(with("TradeAmt", "4294968586")), ecpay.ErrMalformed},
		{"trade number missing", sandbox().Sign(with("TradeNo", "")), ecpay.ErrMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sandbox().Verify(tt.params)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Verify error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerify_ParsesFields(t *testing.T) {
	cb, err := sandbox().Verify(sandbox().Sign(successCallback()))
	if err != nil {
		t.Fatal(err)
	}
	want := ecpay.Callback{MerchantTradeNo: "GS42", TradeNo: "2611141530450001", AmountTWD: 1290, Paid: true, RtnMsg: "Succeeded"}
	if cb != want {
		t.Errorf("callback = %+v, want %+v", cb, want)
	}

	failed := successCallback()
	failed.Set("RtnCode", "10100058")
	cb, err = sandbox().Verify(sandbox().Sign(failed))
	if err != nil || cb.Paid {
		t.Errorf("failed payment: paid=%v err=%v, want paid=false and no error", cb.Paid, err)
	}
}

func TestOrderRef(t *testing.T) {
	id, err := ecpay.ParseOrderRef(ecpay.OrderRef(42))
	if err != nil || id != 42 {
		t.Fatalf("round trip = %d, %v", id, err)
	}
	for _, bad := range []string{"", "42", "GS", "GSabc", "GS-1", "GS0", "XX42"} {
		if _, err := ecpay.ParseOrderRef(bad); !errors.Is(err, ecpay.ErrMalformed) {
			t.Errorf("ParseOrderRef(%q) error = %v, want ErrMalformed", bad, err)
		}
	}
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func with(k, v string) url.Values {
	p := successCallback()
	p.Set(k, v)
	return p
}

func withMerchant(id string) url.Values { return with("MerchantID", id) }
