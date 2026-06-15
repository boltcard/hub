package phoenix

// This file exposes a small amount of test-only surface so that packages
// outside phoenix (notably web) can exercise handlers that call the Phoenix
// API against a mock server. The phoenix package's own tests use the private
// helpers in phoenix_test.go, but those are not visible to other packages,
// and phoenixBaseURL / the cached password are deliberately unexported.
//
// These functions must only be called from tests. They are kept in a normal
// (non _test.go) file purely so they are importable from other packages'
// tests; they perform no I/O and have no effect unless explicitly invoked.

// UseMockPhoenix points the Phoenix client at baseURL (typically an
// httptest.Server URL) and primes a dummy password so that Phoenix-calling
// code reaches the mock instead of failing at password load. It returns a
// restore function that reverts both the base URL and the password cache to
// their previous values.
//
// Always defer the returned restore: the password cache and base URL are
// process-global, so leaving them primed would leak into other tests (for
// example, code paths that expect an unconfigured Phoenix). Because the
// underlying state is global, callers must not run such tests in parallel with
// anything that exercises Phoenix.
func UseMockPhoenix(baseURL string) (restore func()) {
	oldURL := phoenixBaseURL
	oldPassword := cachedPassword
	oldErr := passwordErr

	// Mark the one-time loader as done so getPassword reads the cache below
	// rather than trying to open the real phoenix config file.
	passwordOnce.Do(func() {})

	phoenixBaseURL = baseURL
	cachedPassword = "testpass"
	passwordErr = nil

	return func() {
		phoenixBaseURL = oldURL
		cachedPassword = oldPassword
		passwordErr = oldErr
	}
}
