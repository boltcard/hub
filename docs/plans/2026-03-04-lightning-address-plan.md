# Lightning Address Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add lightning address support so each card has a randomly generated address (e.g. `a3f7b2c1@domain.com`) that can receive Lightning payments.

**Architecture:** Two new LNURL-pay endpoints serve the lightning address protocol. A `ln_address` column on the `cards` table maps hex usernames to cards. Invoice creation reuses the existing `phoenix.CreateInvoice()` + `Db_add_card_receipt()` flow; settlement happens automatically via the existing Phoenix WebSocket listener.

**Tech Stack:** Go 1.25.7 (CGo/SQLite), Gorilla Mux, Phoenix Server API, React 19 + TypeScript + Tailwind v4 + shadcn/ui

**Design doc:** `docs/plans/2026-03-04-lightning-address-design.md`

---

### Task 1: Schema Migration — Add ln_address columns

**Files:**
- Modify: `docker/card/db/db_create.go` (append new function after line 197)
- Modify: `docker/card/db/db_init.go` (lines 40-46)
- Modify: `docker/card/db/db_test.go` (line 25 — update schema version check)

**Step 1: Write the migration function**

Add `update_schema_6()` to the end of `docker/card/db/db_create.go`:

```go
func update_schema_6(db *sql.DB) {

	// Generate random hex addresses for existing cards
	rows, err := db.Query("SELECT card_id FROM cards WHERE 1=1")
	if err != nil {
		log.Printf("update_schema_6 select error: %q", err)
		return
	}
	var cardIds []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		cardIds = append(cardIds, id)
	}
	rows.Close()

	sqlStmt := `
		BEGIN TRANSACTION;
		ALTER TABLE cards ADD COLUMN ln_address CHAR(12) NOT NULL DEFAULT '';
		ALTER TABLE cards ADD COLUMN ln_address_enabled CHAR(1) NOT NULL DEFAULT 'Y';
		UPDATE settings SET value='7' WHERE name='schema_version_number';
		COMMIT TRANSACTION;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("update_schema_6 alter error: %q", err)
		return
	}

	// Backfill existing cards with random hex addresses
	for _, id := range cardIds {
		addr := randomHex8()
		_, err := db.Exec("UPDATE cards SET ln_address = $1 WHERE card_id = $2", addr, id)
		if err != nil {
			log.Printf("update_schema_6 backfill error for card %d: %q", id, err)
		}
	}

	// Create partial unique index
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_cards_ln_address ON cards(ln_address) WHERE ln_address != ''")
	if err != nil {
		log.Printf("update_schema_6 index error: %q", err)
	}
}

// randomHex8 generates an 8-character random hex string for lightning addresses.
func randomHex8() string {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return hex.EncodeToString(b)
}
```

Note: Add `"crypto/rand"`, `"encoding/hex"`, `"fmt"`, and `"time"` to the import block in `db_create.go`.

**Step 2: Update db_init.go to call the migration**

In `docker/card/db/db_init.go`, replace lines 40-46:

```go
	if Db_get_setting(db_conn, "schema_version_number") == "5" {
		update_schema_5(db_conn) // note column
	}

	if Db_get_setting(db_conn, "schema_version_number") != "6" {
		panic("database schema is not as expected")
	}
```

with:

```go
	if Db_get_setting(db_conn, "schema_version_number") == "5" {
		update_schema_5(db_conn) // note column
	}

	if Db_get_setting(db_conn, "schema_version_number") == "6" {
		update_schema_6(db_conn) // ln_address columns
	}

	if Db_get_setting(db_conn, "schema_version_number") != "7" {
		panic("database schema is not as expected")
	}
```

**Step 3: Update schema version test**

In `docker/card/db/db_test.go`, update `TestDbInit_SchemaMigratesToLatest` (line 25):

Change `"6"` to `"7"` in the expected version.

**Step 4: Run tests to verify migration works**

Run: `cd docker/card && go test -race -count=1 ./db/`

Expected: All tests pass. The schema version check test now expects "7".

**Step 5: Commit**

```
feat: add schema v7 migration for lightning address columns
```

---

### Task 2: Update Card struct and Db_get_card to include ln_address fields

**Files:**
- Modify: `docker/card/db/db_get.go` (lines 157-222 — Card struct and Db_get_card)

**Step 1: Add fields to Card struct**

In `docker/card/db/db_get.go`, add two fields to the `Card` struct (after `Note` field at line 181):

```go
	Ln_address         string
	Ln_address_enabled string
```

**Step 2: Update Db_get_card SELECT and Scan**

In `docker/card/db/db_get.go`, update the `Db_get_card` function:

Change the SQL (line 188-194) to add `ln_address, ln_address_enabled` to the SELECT:

```go
	sqlStatement := `SELECT card_id, key0_auth, key1_enc, ` +
		`key2_cmac, key3, key4, login, password, access_token, ` +
		`refresh_token, uid, last_counter_value, ` +
		`lnurlw_request_timeout_sec, lnurlw_enable, ` +
		`lnurlw_k1, lnurlw_k1_expiry, tx_limit_sats, ` +
		`day_limit_sats, uid_privacy, pin_enable, pin_number, ` +
		`pin_limit_sats, wiped, note, ln_address, ln_address_enabled FROM cards WHERE card_id=$1 AND wiped = 'N';`
```

Add two more Scan fields (after `&c.Note` at line 220):

```go
		&c.Ln_address,
		&c.Ln_address_enabled)
```

Remove the closing paren from the `&c.Note)` line so it becomes `&c.Note,`.

**Step 3: Run tests**

Run: `cd docker/card && go test -race -count=1 ./db/`

Expected: All tests pass.

**Step 4: Commit**

```
feat: add ln_address fields to Card struct and Db_get_card
```

---

### Task 3: Add Db_get_card_by_ln_address lookup function

**Files:**
- Modify: `docker/card/db/db_get.go` (add new function)
- Modify: `docker/card/db/db_test.go` (add test)

**Step 1: Write the test**

Add to `docker/card/db/db_test.go`:

```go
func TestDbGetCardByLnAddress_Found(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	// Get the auto-generated ln_address
	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Ln_address == "" {
		t.Fatal("expected non-empty ln_address")
	}

	cardId := Db_get_card_by_ln_address(db, card.Ln_address)
	if cardId != 1 {
		t.Fatalf("expected card_id 1, got %d", cardId)
	}
}

func TestDbGetCardByLnAddress_NotFound(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	cardId := Db_get_card_by_ln_address(db, "nonexistent")
	if cardId != 0 {
		t.Fatalf("expected 0 for unknown address, got %d", cardId)
	}
}

func TestDbGetCardByLnAddress_DisabledExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	card, _ := Db_get_card(db, 1)
	// Disable ln_address
	db.Exec("UPDATE cards SET ln_address_enabled = 'N' WHERE card_id = 1")

	cardId := Db_get_card_by_ln_address(db, card.Ln_address)
	if cardId != 0 {
		t.Fatalf("expected 0 for disabled address, got %d", cardId)
	}
}

func TestDbGetCardByLnAddress_WipedExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	card, _ := Db_get_card(db, 1)
	Db_wipe_card(db, 1)

	cardId := Db_get_card_by_ln_address(db, card.Ln_address)
	if cardId != 0 {
		t.Fatalf("expected 0 for wiped card, got %d", cardId)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd docker/card && go test -race -count=1 ./db/ -run TestDbGetCardByLnAddress`

Expected: FAIL — `Db_get_card_by_ln_address` not defined.

**Step 3: Implement the function**

Add to `docker/card/db/db_get.go`:

```go
func Db_get_card_by_ln_address(db_conn *sql.DB, ln_address string) (card_id int) {

	sqlStatement := `SELECT card_id FROM cards WHERE ln_address=$1 AND ln_address_enabled='Y' AND wiped='N';`
	row := db_conn.QueryRow(sqlStatement, ln_address)

	value := 0
	err := row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}
```

**Step 4: Run tests to verify they pass**

Run: `cd docker/card && go test -race -count=1 ./db/ -run TestDbGetCardByLnAddress`

Expected: PASS

**Step 5: Commit**

```
feat: add Db_get_card_by_ln_address lookup function
```

---

### Task 4: Update card insert functions to generate ln_address

**Files:**
- Modify: `docker/card/db/db_insert.go` (both insert functions)
- Modify: `docker/card/db/db_test.go` (verify ln_address populated)

**Step 1: Write test**

Add to `docker/card/db/db_test.go`:

```go
func TestDbInsertCard_GeneratesLnAddress(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Ln_address) != 8 {
		t.Fatalf("expected 8-char ln_address, got %q (len %d)", card.Ln_address, len(card.Ln_address))
	}
	if card.Ln_address_enabled != "Y" {
		t.Fatalf("expected ln_address_enabled 'Y', got %q", card.Ln_address_enabled)
	}
}

func TestDbInsertCardWithUid_GeneratesLnAddress(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card_with_uid(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1", "uid1", "tag1")

	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Ln_address) != 8 {
		t.Fatalf("expected 8-char ln_address, got %q (len %d)", card.Ln_address, len(card.Ln_address))
	}
}

func TestDbInsertCard_UniqueLnAddresses(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login2", "pass2")

	card1, _ := Db_get_card(db, 1)
	card2, _ := Db_get_card(db, 2)
	if card1.Ln_address == card2.Ln_address {
		t.Fatalf("expected unique ln_addresses, both got %q", card1.Ln_address)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd docker/card && go test -race -count=1 ./db/ -run TestDbInsertCard_Generates`

Expected: FAIL — ln_address is empty because insert doesn't set it yet.

**Step 3: Update Db_insert_card**

In `docker/card/db/db_insert.go`, update `Db_insert_card` to include `ln_address`:

```go
func Db_insert_card(db_conn *sql.DB, key0 string, key1 string, k2 string, key3 string, key4 string,
	login string, password string) {

	lnAddress := randomHex8()

	// insert a new card record
	sqlStatement := `INSERT INTO cards (key0_auth, key1_enc,` +
		` key2_cmac, key3, key4, login, password, ln_address)` +
		` VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	res, err := db_conn.Exec(sqlStatement, key0, key1, k2, key3, key4, login, password, lnAddress)
	if err != nil {
		log.Error("db_insert_card error: ", err)
		return
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_insert_card rows affected error: ", err)
		return
	}
	if count != 1 {
		log.Error("db_insert_card: expected one record to be inserted")
	}
}
```

Do the same for `Db_insert_card_with_uid`:

```go
func Db_insert_card_with_uid(db_conn *sql.DB, key0 string, key1 string, k2 string, key3 string, key4 string,
	login string, password string, uid string, group_tag string) {

	lnAddress := randomHex8()

	// insert a new card record
	sqlStatement := `INSERT INTO cards (key0_auth, key1_enc,` +
		` key2_cmac, key3, key4, login, password, uid, group_tag, ln_address)` +
		` VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`
	res, err := db_conn.Exec(sqlStatement, key0, key1, k2, key3, key4, login, password, uid, group_tag, lnAddress)
	if err != nil {
		log.Error("db_insert_card_with_uid error: ", err)
		return
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_insert_card_with_uid rows affected error: ", err)
		return
	}
	if count != 1 {
		log.Error("db_insert_card_with_uid: expected one record to be inserted")
	}
}
```

Note: `randomHex8()` is defined in `db_create.go` (Task 1). Both files are in package `db` so it's accessible.

**Step 4: Run all db tests**

Run: `cd docker/card && go test -race -count=1 ./db/`

Expected: All tests pass.

**Step 5: Commit**

```
feat: auto-generate ln_address on card insert
```

---

### Task 5: Update admin API to expose ln_address fields

**Files:**
- Modify: `docker/card/web/admin_api_cards.go` (lines 84-106 for GET, lines 122-146 for PUT limits)

**Step 1: Update adminApiGetCard response**

In `docker/card/web/admin_api_cards.go`, update `adminApiGetCard` (around line 94-106) to include the new fields:

```go
	hostDomain := db.Db_get_setting(app.db_conn, "host_domain")

	writeJSON(w, map[string]any{
		"cardId":           card.Card_id,
		"uid":              card.Uid,
		"note":             card.Note,
		"balanceSats":      balance,
		"lnurlwEnable":     card.Lnurlw_enable,
		"txLimitSats":      card.Tx_limit_sats,
		"dayLimitSats":     card.Day_limit_sats,
		"pinEnable":        card.Pin_enable,
		"pinLimitSats":     card.Pin_limit_sats,
		"wiped":            card.Wiped,
		"lnAddress":        card.Ln_address,
		"lnAddressEnabled": card.Ln_address_enabled,
		"hostDomain":       hostDomain,
	})
```

**Step 2: Update adminApiUpdateCardLimits to accept lnAddressEnabled**

In `docker/card/web/admin_api_cards.go`, update `adminApiUpdateCardLimits`:

Add `LnAddressEnabled` to the request struct:

```go
	var req struct {
		TxLimitSats      int    `json:"txLimitSats"`
		DayLimitSats     int    `json:"dayLimitSats"`
		LnurlwEnable     string `json:"lnurlwEnable"`
		LnAddressEnabled string `json:"lnAddressEnabled"`
	}
```

After the existing `lnurlwEnable` validation, add:

```go
	// Validate lnAddressEnabled (optional — default to current value)
	if req.LnAddressEnabled != "" && req.LnAddressEnabled != "Y" && req.LnAddressEnabled != "N" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "lnAddressEnabled must be Y or N"})
		return
	}
```

After the existing `Db_update_card_without_pin` call, add:

```go
	if req.LnAddressEnabled != "" {
		db.Db_update_card_ln_address_enabled(app.db_conn, cardId, req.LnAddressEnabled)
	}
```

**Step 3: Add Db_update_card_ln_address_enabled to db_update.go**

Add to `docker/card/db/db_update.go`:

```go
func Db_update_card_ln_address_enabled(db_conn *sql.DB, card_id int, ln_address_enabled string) {

	sqlStatement := `UPDATE cards SET ln_address_enabled = $1 WHERE card_id = $2 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, ln_address_enabled, card_id)
	if err != nil {
		log.Error("db_update_card_ln_address_enabled error: ", err)
	}
}
```

**Step 4: Run all tests**

Run: `cd docker/card && go test -race -count=1 ./...`

Expected: All tests pass.

**Step 5: Commit**

```
feat: expose ln_address fields in admin API
```

---

### Task 6: LNURL-pay request handler

**Files:**
- Create: `docker/card/web/lnurlp.go`
- Modify: `docker/card/web/app.go` (add routes at line 47, before `/new`)

**Step 1: Write the LNURL-pay request handler test**

Add to `docker/card/web/web_test.go`:

```go
func TestLnurlpRequest_ValidAddress(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	card, _ := db.Db_get_card(app.db_conn, 1)

	handler := app.CreateHandler_LnurlpRequest()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address, nil)
	r.SetPathValue("username", card.Ln_address)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["tag"] != "payRequest" {
		t.Fatalf("expected tag 'payRequest', got %v", resp["tag"])
	}
	if resp["commentAllowed"] != float64(140) {
		t.Fatalf("expected commentAllowed 140, got %v", resp["commentAllowed"])
	}
	if resp["minSendable"] != float64(1000) {
		t.Fatalf("expected minSendable 1000, got %v", resp["minSendable"])
	}
}

func TestLnurlpRequest_NotFound(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_LnurlpRequest()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/nonexistent", nil)
	r.SetPathValue("username", "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLnurlpRequest_DisabledAddress(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	card, _ := db.Db_get_card(app.db_conn, 1)
	db.Db_update_card_ln_address_enabled(app.db_conn, 1, "N")

	handler := app.CreateHandler_LnurlpRequest()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address, nil)
	r.SetPathValue("username", card.Ln_address)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for disabled address, got %d", w.Code)
	}
}
```

Note: These tests use `r.SetPathValue("username", ...)` because the handler reads the username from the request path. If using Gorilla Mux vars, the test will need to use `mux.SetURLVars(r, map[string]string{"username": ...})` instead. Check which approach the handler uses.

**Step 2: Run tests to verify they fail**

Run: `cd docker/card && go test -race -count=1 ./web/ -run TestLnurlp`

Expected: FAIL — `CreateHandler_LnurlpRequest` not defined.

**Step 3: Create `web/lnurlp.go`**

Create `docker/card/web/lnurlp.go`:

```go
package web

import (
	"card/db"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func lnurlpMetadata(username, hostDomain string) string {
	return fmt.Sprintf(`[["text/plain","Payment to %s@%s"]]`, username, hostDomain)
}

func (app *App) CreateHandler_LnurlpRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		username := vars["username"]
		if username == "" {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		cardId := db.Db_get_card_by_ln_address(app.db_conn, username)
		if cardId == 0 {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		hostDomain := db.Db_get_setting(app.db_conn, "host_domain")
		metadata := lnurlpMetadata(username, hostDomain)

		writeJSON(w, map[string]interface{}{
			"tag":            "payRequest",
			"callback":       "https://" + hostDomain + "/.well-known/lnurlp/" + username + "/callback",
			"minSendable":    1000,
			"maxSendable":    100000000000,
			"metadata":       metadata,
			"commentAllowed": 140,
		})
	}
}

func descriptionHash(metadata string) string {
	hash := sha256.Sum256([]byte(metadata))
	return hex.EncodeToString(hash[:])
}
```

**Step 4: Register routes in app.go**

In `docker/card/web/app.go`, add before the `/new` route (before line 47):

```go
	// Lightning Address (LNURL-pay)
	router.Path("/.well-known/lnurlp/{username}").Methods("GET").HandlerFunc(app.CreateHandler_LnurlpRequest())
	router.Path("/.well-known/lnurlp/{username}/callback").Methods("GET").HandlerFunc(app.CreateHandler_LnurlpCallback())
```

Note: `CreateHandler_LnurlpCallback` doesn't exist yet — add a stub for now:

Add to `docker/card/web/lnurlp.go`:

```go
func (app *App) CreateHandler_LnurlpCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"status": "ERROR", "reason": "not implemented"})
	}
}
```

**Step 5: Update tests to use mux.SetURLVars**

Since the handler uses `mux.Vars(r)`, tests need to set vars via Gorilla Mux. Update the test to use:

```go
import "github.com/gorilla/mux"

// In each test, replace r.SetPathValue with:
r = mux.SetURLVars(r, map[string]string{"username": card.Ln_address})
```

**Step 6: Run tests**

Run: `cd docker/card && go test -race -count=1 ./web/ -run TestLnurlp`

Expected: PASS

**Step 7: Commit**

```
feat: add LNURL-pay request handler and route registration
```

---

### Task 7: LNURL-pay callback handler

**Files:**
- Modify: `docker/card/web/lnurlp.go` (replace stub)

**Step 1: Write callback tests**

Add to `docker/card/web/web_test.go`:

```go
func TestLnurlpCallback_MissingAmount(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	card, _ := db.Db_get_card(app.db_conn, 1)

	handler := app.CreateHandler_LnurlpCallback()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address+"/callback", nil)
	r = mux.SetURLVars(r, map[string]string{"username": card.Ln_address})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLnurlpCallback_AmountTooLow(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	card, _ := db.Db_get_card(app.db_conn, 1)

	handler := app.CreateHandler_LnurlpCallback()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address+"/callback?amount=999", nil)
	r = mux.SetURLVars(r, map[string]string{"username": card.Ln_address})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for amount too low, got %d", w.Code)
	}
}

func TestLnurlpCallback_AmountTooHigh(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	card, _ := db.Db_get_card(app.db_conn, 1)

	handler := app.CreateHandler_LnurlpCallback()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address+"/callback?amount=100000000001", nil)
	r = mux.SetURLVars(r, map[string]string{"username": card.Ln_address})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for amount too high, got %d", w.Code)
	}
}

func TestLnurlpCallback_UnknownAddress(t *testing.T) {
	app := openTestApp(t)

	handler := app.CreateHandler_LnurlpCallback()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/unknown/callback?amount=5000", nil)
	r = mux.SetURLVars(r, map[string]string{"username": "unknown"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLnurlpCallback_ValidAmount_PhoenixUnavailable(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	card, _ := db.Db_get_card(app.db_conn, 1)

	handler := app.CreateHandler_LnurlpCallback()
	r := httptest.NewRequest("GET", "/.well-known/lnurlp/"+card.Ln_address+"/callback?amount=5000000", nil)
	r = mux.SetURLVars(r, map[string]string{"username": card.Ln_address})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// Phoenix is unavailable in tests, so this should return an error
	// but it exercises the validation path successfully
	if w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound {
		t.Fatalf("expected validation to pass (not 400/404), got %d", w.Code)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd docker/card && go test -race -count=1 ./web/ -run TestLnurlpCallback`

Expected: FAIL — stub returns 501 for everything.

**Step 3: Implement the callback handler**

Replace the stub in `docker/card/web/lnurlp.go`:

```go
func (app *App) CreateHandler_LnurlpCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		username := vars["username"]
		if username == "" {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		cardId := db.Db_get_card_by_ln_address(app.db_conn, username)
		if cardId == 0 {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		// Validate amount (in millisats)
		amountStr := r.URL.Query().Get("amount")
		if amountStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "missing amount"})
			return
		}

		amountMsat, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || amountMsat < 1000 || amountMsat > 100000000000 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "amount out of range"})
			return
		}

		amountSats := int(amountMsat / 1000)

		hostDomain := db.Db_get_setting(app.db_conn, "host_domain")
		metadata := lnurlpMetadata(username, hostDomain)
		dHash := descriptionHash(metadata)

		// Create invoice via Phoenix with description hash
		createInvoiceResponse, err := phoenix.CreateInvoice(phoenix.CreateInvoiceRequest{
			Description: dHash,
			AmountSat:   strconv.Itoa(amountSats),
			ExternalId:  "",
		})
		if err != nil {
			log.Error("lnurlp CreateInvoice error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "failed to create invoice"})
			return
		}

		// Insert pending receipt
		db.Db_add_card_receipt(app.db_conn, cardId,
			createInvoiceResponse.Serialized, createInvoiceResponse.PaymentHash, amountSats)

		log.Info("lnurlp invoice created for ", username, " amount=", amountSats)

		writeJSON(w, map[string]interface{}{
			"pr":     createInvoiceResponse.Serialized,
			"routes": []string{},
		})
	}
}
```

Add these imports to `lnurlp.go`:

```go
import (
	"card/db"
	"card/phoenix"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)
```

**Step 4: Run tests**

Run: `cd docker/card && go test -race -count=1 ./web/ -run TestLnurlpCallback`

Expected: PASS (validation tests pass; Phoenix-unavailable test returns 500, not 400/404).

**Step 5: Run all tests**

Run: `cd docker/card && go test -race -count=1 ./...`

Expected: All pass.

**Step 6: Commit**

```
feat: add LNURL-pay callback handler with invoice creation
```

---

### Task 8: Caddyfile rate limiting

**Files:**
- Modify: `Caddyfile` (line 37)

**Step 1: Add LNURL-pay path to rate-limited API paths**

In `Caddyfile`, update the `@api_paths` matcher (line 37):

Change:
```
		path /create /payinvoice /addinvoice /ln /cb
```

To:
```
		path /create /payinvoice /addinvoice /ln /cb /.well-known/lnurlp/*
```

**Step 2: Commit**

```
feat: rate-limit LNURL-pay endpoints (30 req/min per IP)
```

---

### Task 9: Admin UI — Lightning Address card on card detail page

**Files:**
- Modify: `docker/card/admin-ui/src/pages/card-detail.tsx`

**Step 1: Update CardDetail interface**

Add fields to the `CardDetail` interface (around line 41-51):

```typescript
  lnAddress: string;
  lnAddressEnabled: string;
  hostDomain: string;
```

**Step 2: Add Lightning Address card section**

Between the Info/Balance grid (line 237) and the Limits card (line 239), add a new Lightning Address card.

Import `QRCodeSVG` — install the package first:

Run: `cd docker/card/admin-ui && npm install qrcode.react`

Add to the imports:

```typescript
import { QRCodeSVG } from "qrcode.react";
import { Copy, Zap } from "lucide-react";
```

Add the Lightning Address card component JSX after the closing `</div>` of the grid (after line 237) and before the Limits card:

```tsx
      {/* Lightning Address */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <Zap className="h-4 w-4" />
            Lightning Address
          </CardTitle>
          <Badge variant={card.lnAddressEnabled === "Y" ? "default" : "secondary"}>
            {card.lnAddressEnabled === "Y" ? "Enabled" : "Disabled"}
          </Badge>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
              {card.lnAddress}@{card.hostDomain}
            </code>
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7"
              onClick={() => {
                navigator.clipboard.writeText(
                  `${card.lnAddress}@${card.hostDomain}`
                );
                toast.success("Copied to clipboard");
              }}
            >
              <Copy className="h-3.5 w-3.5" />
            </Button>
          </div>
          {card.lnAddressEnabled === "Y" && (
            <div className="flex justify-center p-4 bg-white rounded-lg w-fit">
              <QRCodeSVG
                value={`lightning:${card.lnAddress}@${card.hostDomain}`}
                size={160}
                level="M"
              />
            </div>
          )}
        </CardContent>
      </Card>
```

**Step 3: Add ln_address_enabled toggle to the Limits form**

In the limits form section (inside the `limitsForm` state and form JSX), add a toggle for `lnAddressEnabled`.

Update the `limitsForm` state type to include `lnAddressEnabled`:

```typescript
  const [limitsForm, setLimitsForm] = useState<{
    txLimitSats: string;
    dayLimitSats: string;
    lnurlwEnable: string;
    lnAddressEnabled: string;
  } | null>(null);
```

Update `startEditLimits`:

```typescript
  function startEditLimits() {
    setLimitsForm({
      txLimitSats: String(card!.txLimitSats),
      dayLimitSats: String(card!.dayLimitSats),
      lnurlwEnable: card!.lnurlwEnable,
      lnAddressEnabled: card!.lnAddressEnabled,
    });
  }
```

Update `saveLimits` to include `lnAddressEnabled`:

```typescript
  function saveLimits() {
    if (!limitsForm) return;
    limitsMutation.mutate({
      txLimitSats: Number(limitsForm.txLimitSats) || 0,
      dayLimitSats: Number(limitsForm.dayLimitSats) || 0,
      lnurlwEnable: limitsForm.lnurlwEnable,
      lnAddressEnabled: limitsForm.lnAddressEnabled,
    });
  }
```

Update `limitsMutation.mutationFn` type:

```typescript
  const limitsMutation = useMutation({
    mutationFn: (data: {
      txLimitSats: number;
      dayLimitSats: number;
      lnurlwEnable: string;
      lnAddressEnabled: string;
    }) => apiPut(`/cards/${id}/limits`, data),
```

Add a fourth column to the limits form grid (change `sm:grid-cols-3` to `sm:grid-cols-4`), and add:

```tsx
                <div className="space-y-2">
                  <Label>Lightning Address</Label>
                  <Select
                    value={limitsForm.lnAddressEnabled}
                    onValueChange={(v) =>
                      setLimitsForm({ ...limitsForm, lnAddressEnabled: v })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Y">Enabled</SelectItem>
                      <SelectItem value="N">Disabled</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
```

Also add a display row for ln address enabled in the non-editing view (add a 4th column, update grid to `sm:grid-cols-4`):

```tsx
              <div>
                <span className="text-muted-foreground">Lightning Address</span>
                <p>{card.lnAddressEnabled === "Y" ? "Enabled" : "Disabled"}</p>
              </div>
```

**Step 4: Build frontend**

Run: `cd docker/card/admin-ui && npm run build`

Expected: Build succeeds with no errors.

**Step 5: Commit**

```
feat: add lightning address card with QR code to admin card detail page
```

---

### Task 10: Bump version and final integration test

**Files:**
- Modify: `docker/card/build/build.go` (line 3)

**Step 1: Bump version**

In `docker/card/build/build.go`, change:

```go
var Version string = "0.16.0"
```

to:

```go
var Version string = "0.17.0"
```

**Step 2: Run all tests**

Run: `cd docker/card && go test -race -count=1 ./...`

Expected: All tests pass.

**Step 3: Build Docker image**

Run: `cd /home/debian/hub && docker compose build card`

Expected: Build succeeds.

**Step 4: Commit**

```
bump version to 0.17.0
```

---

## Summary of all tasks

| Task | Description | Files |
|------|-------------|-------|
| 1 | Schema v7 migration | `db_create.go`, `db_init.go`, `db_test.go` |
| 2 | Card struct + Db_get_card update | `db_get.go` |
| 3 | Db_get_card_by_ln_address | `db_get.go`, `db_test.go` |
| 4 | Card insert generates ln_address | `db_insert.go`, `db_test.go` |
| 5 | Admin API exposes ln_address | `admin_api_cards.go`, `db_update.go` |
| 6 | LNURL-pay request handler | `lnurlp.go` (new), `app.go`, `web_test.go` |
| 7 | LNURL-pay callback handler | `lnurlp.go`, `web_test.go` |
| 8 | Caddyfile rate limiting | `Caddyfile` |
| 9 | Admin UI lightning address card | `card-detail.tsx` |
| 10 | Version bump + integration | `build.go` |
