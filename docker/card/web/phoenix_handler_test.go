package web

import (
	"card/db"
	"card/phoenix"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestAppNoPollers builds an App backed by an initialised in-memory DB
// WITHOUT starting the Phoenix background pollers. The pollers call Phoenix on
// a 30s loop and read the process-global phoenixBaseURL / password cache; tests
// that mutate those globals (via phoenix.UseMockPhoenix) must avoid spawning
// pollers of their own so the mutation cannot race a concurrent poll.
func newTestAppNoPollers(t *testing.T) *App {
	t.Helper()
	db_conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db_conn.Close() })
	db.Db_init(db_conn)
	return &App{db_read: db_conn, db_write: db_conn, hub: newWsHub(), stop: make(chan struct{})}
}

// authedCard inserts a card, sets an access token, and returns the token so a
// handler protected by getAuthenticatedCardID can be exercised.
func authedCard(t *testing.T, app *App, login, token string) {
	t.Helper()
	db.Db_insert_card(app.db_write, "k0", "k1", "k2", "k3", "k4", login, "pass")
	if err := db.Db_set_tokens(app.db_write, login, "pass", token, token+"refresh"); err != nil {
		t.Fatalf("failed to set tokens: %v", err)
	}
}

// TestAddInvoice_CreatesReceipt exercises the wallet AddInvoice handler end to
// end against a mock Phoenix server. Before phoenix.UseMockPhoenix existed,
// this handler could not be tested from the web package because every call hit
// the real (unreachable) Phoenix node. It verifies that the handler forwards
// the requested amount to Phoenix, maps the invoice into the response, and
// records a card_receipt row.
func TestAddInvoice_CreatesReceipt(t *testing.T) {
	app := newTestAppNoPollers(t)
	authedCard(t, app, "invlogin", "invtoken")

	const serialized = "lnbc500n1mockinvoice"
	const paymentHash = "abcdef0123456789"

	// Mock Phoenix /createinvoice and point the client at it.
	var gotAmount string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/createinvoice" {
			t.Errorf("unexpected phoenix path: %s", r.URL.Path)
		}
		r.ParseForm()
		gotAmount = r.FormValue("amountSat")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(phoenix.CreateInvoiceResponse{
			AmountSat:   50,
			PaymentHash: paymentHash,
			Serialized:  serialized,
		})
	}))
	defer srv.Close()
	defer phoenix.UseMockPhoenix(srv.URL)()

	body := `{"amt":"50","memo":"coffee"}`
	r := httptest.NewRequest("POST", "/addinvoice", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer invtoken")
	w := httptest.NewRecorder()

	app.CreateHandler_AddInvoice().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotAmount != "50" {
		t.Fatalf("expected amountSat 50 forwarded to phoenix, got %q", gotAmount)
	}

	var resp AddInvoiceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %s", w.Body.String())
	}
	if resp.PayReq != serialized || resp.PaymentRequest != serialized {
		t.Fatalf("expected invoice %q in response, got pay_req=%q payment_request=%q",
			serialized, resp.PayReq, resp.PaymentRequest)
	}
	if resp.Hash != paymentHash {
		t.Fatalf("expected hash %q, got %q", paymentHash, resp.Hash)
	}

	// The handler should have recorded a card_receipt for the invoice.
	var receipts int
	if err := app.db_read.QueryRow(
		"SELECT COUNT(*) FROM card_receipts WHERE r_hash_hex = ?", paymentHash,
	).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 {
		t.Fatalf("expected 1 card_receipt for the invoice, got %d", receipts)
	}
}

// TestAddInvoice_RejectsBadAmount verifies the handler validates the amount
// before calling Phoenix. A non-positive amount must be rejected with an error.
func TestAddInvoice_RejectsBadAmount(t *testing.T) {
	app := newTestAppNoPollers(t)
	authedCard(t, app, "badamtlogin", "badamttoken")

	body := `{"amt":"0","memo":"x"}`
	r := httptest.NewRequest("POST", "/addinvoice", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer badamttoken")
	w := httptest.NewRecorder()

	app.CreateHandler_AddInvoice().ServeHTTP(w, r)

	var resp AddInvoiceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %s", w.Body.String())
	}
	if resp.Error == "" {
		t.Fatal("expected an error for non-positive amount")
	}
}

// TestAddInvoice_RequiresAuth verifies the handler rejects unauthenticated
// requests before touching Phoenix. The wallet API signals failures with an
// error body rather than an HTTP status code.
func TestAddInvoice_RequiresAuth(t *testing.T) {
	app := newTestAppNoPollers(t)

	body := `{"amt":"50","memo":"x"}`
	r := httptest.NewRequest("POST", "/addinvoice", strings.NewReader(body))
	w := httptest.NewRecorder()

	app.CreateHandler_AddInvoice().ServeHTTP(w, r)

	var resp AddInvoiceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %s", w.Body.String())
	}
	if resp.Error == "" {
		t.Fatal("expected an error for unauthenticated request")
	}
}
