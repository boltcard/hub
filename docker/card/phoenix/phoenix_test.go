package phoenix

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// primePassword marks the sync.Once as done so InitPassword never tries to
// read the real config file, and sets a known cached password for tests.
func primePassword(pw string) {
	passwordOnce.Do(func() {})
	cachedPassword = pw
	passwordErr = nil
}

// withTestServer points phoenixBaseURL at a test server for the duration of a
// test and primes a non-empty password. The previous base URL is restored on
// cleanup.
func withTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	primePassword("testpass")
	srv := httptest.NewServer(handler)
	old := phoenixBaseURL
	phoenixBaseURL = srv.URL
	t.Cleanup(func() {
		phoenixBaseURL = old
		srv.Close()
	})
}

func TestGetBalance_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getbalance" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		// Phoenix uses HTTP basic auth with empty username + password.
		_, pw, ok := r.BasicAuth()
		if !ok || pw != "testpass" {
			t.Errorf("expected basic auth with testpass, got ok=%v pw=%q", ok, pw)
		}
		w.Write([]byte(`{"balanceSat":12345,"feeCreditSat":67}`))
	})

	bal, err := GetBalance()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bal.BalanceSat != 12345 || bal.FeeCreditSat != 67 {
		t.Fatalf("unexpected balance: %+v", bal)
	}
}

func TestGetBalance_Non200(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	})

	_, err := GetBalance()
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestGetBalance_PasswordEmpty(t *testing.T) {
	// prime an empty password to exercise the getPassword failure path
	primePassword("")
	defer primePassword("testpass")

	_, err := GetBalance()
	if err == nil {
		t.Fatal("expected error when password is empty")
	}
}

func TestCreateInvoice_WithDescription(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/createinvoice" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("description") != "coffee" {
			t.Errorf("expected description=coffee, got %q", r.PostForm.Get("description"))
		}
		if r.PostForm.Has("descriptionHash") {
			t.Errorf("descriptionHash should not be set when Description is used")
		}
		if r.PostForm.Get("amountSat") != "1000" {
			t.Errorf("expected amountSat=1000, got %q", r.PostForm.Get("amountSat"))
		}
		w.Write([]byte(`{"amountSat":1000,"paymentHash":"abc","serialized":"lnbc1..."}`))
	})

	resp, err := CreateInvoice(CreateInvoiceRequest{
		Description: "coffee",
		AmountSat:   "1000",
		ExternalId:  "card-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PaymentHash != "abc" || resp.Serialized != "lnbc1..." || resp.AmountSat != 1000 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateInvoice_WithDescriptionHash(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("descriptionHash") != "deadbeef" {
			t.Errorf("expected descriptionHash=deadbeef, got %q", r.PostForm.Get("descriptionHash"))
		}
		if r.PostForm.Has("description") {
			t.Errorf("description should not be set when DescriptionHash is used")
		}
		w.Write([]byte(`{"amountSat":1000,"paymentHash":"abc","serialized":"lnbc1..."}`))
	})

	_, err := CreateInvoice(CreateInvoiceRequest{
		DescriptionHash: "deadbeef",
		AmountSat:       "1000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetIncomingPayment_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/incoming/myhash" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`{"paymentHash":"myhash","isPaid":true,"receivedSat":500,"fees":2}`))
	})

	p, err := GetIncomingPayment("myhash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsPaid || p.ReceivedSat != 500 || p.Fees != 2 {
		t.Fatalf("unexpected payment: %+v", p)
	}
}

func TestGetOffer_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getoffer" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte("lno1offerstring"))
	})

	offer, err := GetOffer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if offer != "lno1offerstring" {
		t.Fatalf("unexpected offer: %q", offer)
	}
}

func TestListChannels_NormalChannel(t *testing.T) {
	body := `[{
		"type":"fr.acinq.lightning.channel.states.Normal",
		"commitments":{
			"channelParams":{"channelId":"chan1"},
			"active":[{"localCommit":{"spec":{"toLocal":3000,"toRemote":7000}}}]
		}
	}]`
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/listchannels" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(body))
	})

	channels, err := ListChannels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	ch := channels[0]
	if ch.State != "Normal" || ch.ChannelID != "chan1" {
		t.Fatalf("unexpected channel: %+v", ch)
	}
	if ch.BalanceMsat != 3000 || ch.InboundLiquidMsat != 7000 {
		t.Fatalf("unexpected balances: %+v", ch)
	}
}

func TestListChannels_SkipsChannelWithoutCommitments(t *testing.T) {
	body := `[{"type":"fr.acinq.lightning.channel.states.WaitForFundingConfirmed"}]`
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	channels, err := ListChannels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("expected channel without commitments to be skipped, got %d", len(channels))
	}
}

func TestExtractChannel_OfflineUnwrapsNestedState(t *testing.T) {
	// Offline/Syncing channels nest their commitments inside "state".
	raw := listChannelRaw{
		Type:  "fr.acinq.lightning.channel.states.Offline",
		State: []byte(`{"type":"fr.acinq.lightning.channel.states.Normal","commitments":{"channelParams":{"channelId":"nested1"},"active":[{"localCommit":{"spec":{"toLocal":42,"toRemote":58}}}]}}`),
	}
	ch, ok := extractChannel(raw)
	if !ok {
		t.Fatal("expected ok for offline channel with nested commitments")
	}
	if ch.State != "Normal" {
		t.Fatalf("expected inner state Normal, got %q", ch.State)
	}
	if ch.ChannelID != "nested1" || ch.BalanceMsat != 42 || ch.InboundLiquidMsat != 58 {
		t.Fatalf("unexpected channel: %+v", ch)
	}
}

func TestSendLightningPayment_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payinvoice" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("invoice") != "lnbc1..." || r.PostForm.Get("amountSat") != "250" {
			t.Errorf("unexpected form: %v", r.PostForm)
		}
		w.Write([]byte(`{"recipientAmountSat":250,"routingFeeSat":1,"paymentId":"pid","paymentHash":"ph","paymentPreimage":"pre"}`))
	})

	resp, reason, err := SendLightningPayment(SendLightningPaymentRequest{
		AmountSat: "250",
		Invoice:   "lnbc1...",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != "no_error" {
		t.Fatalf("expected reason no_error, got %q", reason)
	}
	if resp.RecipientAmountSat != 250 || resp.RoutingFeeSat != 1 || resp.PaymentPreimage != "pre" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSendLightningPayment_FailStatusCode(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("rejected"))
	})

	_, reason, err := SendLightningPayment(SendLightningPaymentRequest{
		AmountSat: "250",
		Invoice:   "lnbc1...",
	})
	if err == nil {
		t.Fatal("expected error on non-200 status")
	}
	if reason != "fail_status_code" {
		t.Fatalf("expected reason fail_status_code, got %q", reason)
	}
}

func TestSendLightningPayment_DecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	})

	_, reason, err := SendLightningPayment(SendLightningPaymentRequest{
		AmountSat: "250",
		Invoice:   "lnbc1...",
	})
	if err == nil {
		t.Fatal("expected error decoding invalid JSON")
	}
	if reason != "failed_decode_response" {
		t.Fatalf("expected reason failed_decode_response, got %q", reason)
	}
}

func TestListIncomingPayments_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/incoming" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "5" || q.Get("offset") != "10" || q.Get("all") != "true" {
			t.Errorf("unexpected query params: %v", q)
		}
		w.Write([]byte(`[{"paymentHash":"h1","isPaid":true,"receivedSat":100},{"paymentHash":"h2","isPaid":false}]`))
	})

	payments, err := ListIncomingPayments(5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(payments))
	}
	if payments[0].PaymentHash != "h1" || payments[0].ReceivedSat != 100 {
		t.Fatalf("unexpected first payment: %+v", payments[0])
	}
}

func TestListOutgoingPayments_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/outgoing" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`[{"paymentId":"p1","isPaid":true,"sent":250,"fees":1}]`))
	})

	payments, err := ListOutgoingPayments(5, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 1 || payments[0].PaymentID != "p1" || payments[0].Sent != 250 {
		t.Fatalf("unexpected payments: %+v", payments)
	}
}

func TestGetOutgoingPayment_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/outgoing/pid1" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`{"paymentHash":"ph","isPaid":true,"sent":250,"fees":3}`))
	})

	p, err := GetOutgoingPayment("pid1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsPaid || p.SentSat != 250 || p.FeesSat != 3 {
		t.Fatalf("unexpected payment: %+v", p)
	}
}

func TestSendLightningPayment_NoConfig(t *testing.T) {
	primePassword("")
	defer primePassword("testpass")

	_, reason, err := SendLightningPayment(SendLightningPaymentRequest{
		AmountSat: "250",
		Invoice:   "lnbc1...",
	})
	if err == nil {
		t.Fatal("expected error when password unavailable")
	}
	if reason != "no_config" {
		t.Fatalf("expected reason no_config, got %q", reason)
	}
}

func TestPayLightningAddress_Success(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/paylnaddress" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("amountSat") != "1000" {
			t.Errorf("expected amountSat=1000, got %q", r.PostForm.Get("amountSat"))
		}
		if r.PostForm.Get("address") != "alice@example.com" {
			t.Errorf("expected address=alice@example.com, got %q", r.PostForm.Get("address"))
		}
		if r.PostForm.Get("message") != "payout" {
			t.Errorf("expected message=payout, got %q", r.PostForm.Get("message"))
		}
		w.Write([]byte(`{"recipientAmountSat":1000,"routingFeeSat":5,"paymentId":"pid","paymentHash":"hash","paymentPreimage":"pre"}`))
	})

	resp, reason, err := PayLightningAddress(PayLightningAddressRequest{
		AmountSat: "1000",
		Address:   "alice@example.com",
		Message:   "payout",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != "no_error" {
		t.Fatalf("expected reason no_error, got %q", reason)
	}
	if resp.RoutingFeeSat != 5 || resp.PaymentHash != "hash" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestPayLightningAddress_OmitsEmptyMessage(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Has("message") {
			t.Errorf("message should not be set when empty")
		}
		w.Write([]byte(`{"routingFeeSat":1,"paymentHash":"h"}`))
	})

	_, reason, err := PayLightningAddress(PayLightningAddressRequest{
		AmountSat: "500",
		Address:   "bob@example.com",
	})
	if err != nil || reason != "no_error" {
		t.Fatalf("unexpected outcome: reason=%q err=%v", reason, err)
	}
}

func TestPayLightningAddress_Non200(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("nope"))
	})

	_, reason, err := PayLightningAddress(PayLightningAddressRequest{
		AmountSat: "500",
		Address:   "bob@example.com",
	})
	if err == nil {
		t.Fatal("expected error on non-200")
	}
	if reason != "fail_status_code" {
		t.Fatalf("expected reason fail_status_code, got %q", reason)
	}
}

func TestPayLightningAddress_NoConfig(t *testing.T) {
	primePassword("")
	defer primePassword("testpass")

	_, reason, err := PayLightningAddress(PayLightningAddressRequest{
		AmountSat: "500",
		Address:   "bob@example.com",
	})
	if err == nil {
		t.Fatal("expected error when password unavailable")
	}
	if reason != "no_config" {
		t.Fatalf("expected reason no_config, got %q", reason)
	}
}
