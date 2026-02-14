package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vmkteam/zenrpc/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzServerDo feeds arbitrary bytes into Server.Do().
// Invariants:
//   - must never panic
//   - must always return valid JSON (or nil for notifications)
//   - response must always contain "jsonrpc":"2.0"
func FuzzServerDo(f *testing.F) {
	// seed corpus: valid requests
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":0}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":[3,2],"id":0}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.divide","params":{"a":1,"b":0},"id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pow","params":{"base":3},"id":0}`))

	// seed corpus: notifications (no id)
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2}}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi"}`))

	// seed corpus: batch requests
	f.Add([]byte(`[{"jsonrpc":"2.0","method":"arith.pi","id":1}]`))
	f.Add([]byte(`[{"jsonrpc":"2.0","method":"arith.pi","id":1},{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":1,"b":2},"id":2}]`))
	f.Add([]byte(`[]`))

	// seed corpus: malformed JSON
	f.Add([]byte(`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`))
	f.Add([]byte(`{invalid json`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))
	f.Add([]byte(`123`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`true`))

	// seed corpus: invalid requests
	f.Add([]byte(`{"jsonrpc":"1.0","method":"arith.pi","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","params":{"a":1},"id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"nonexistent","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"rpc.discover","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pow","params":{"base":"3"},"id":0}`))

	// seed corpus: edge-case IDs
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":"string-id"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":-1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":1.5}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":null}`))

	// seed corpus: batch with non-object elements
	f.Add([]byte(`[1]`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`[null]`))

	// seed corpus: deeply nested / large
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":999999999999999999,"b":999999999999999999},"id":1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		resp, err := testRPC.Do(context.Background(), data)
		require.NoError(t, err, "Do() returned error")

		// nil or "null" response is valid (notification or all-notification batch)
		if resp == nil || string(resp) == "null" {
			return
		}

		// response must be valid JSON
		require.True(t, json.Valid(resp), "Do() returned invalid JSON: %s", resp)

		// response must contain jsonrpc version
		// could be a single response or a batch array
		if resp[0] == '[' {
			var batch []json.RawMessage
			require.NoError(t, json.Unmarshal(resp, &batch), "batch response is not a JSON array")
			for i, item := range batch {
				assertValidResponse(t, i, item)
			}
		} else {
			assertValidResponse(t, 0, resp)
		}
	})
}

// assertValidResponse checks that a single JSON-RPC response has required fields.
func assertValidResponse(t *testing.T, idx int, data []byte) {
	t.Helper()

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m), "response[%d] is not a JSON object", idx)
	require.Contains(t, m, "jsonrpc", "response[%d] missing 'jsonrpc' field", idx)
	assert.Equal(t, `"2.0"`, string(m["jsonrpc"]), "response[%d] jsonrpc version", idx)

	_, hasResult := m["result"]
	_, hasError := m["error"]

	// must have exactly one of result or error
	assert.False(t, hasResult && hasError, "response[%d] has both 'result' and 'error'", idx)
	assert.True(t, hasResult || hasError, "response[%d] has neither 'result' nor 'error'", idx)
}

// FuzzIsArray checks that IsArray never panics on arbitrary input.
func FuzzIsArray(f *testing.F) {
	f.Add([]byte(`[1]`))
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(` [1]`))
	f.Add([]byte("\t[1]"))
	f.Add([]byte("\n[1]"))
	f.Add([]byte("\r\n[1]"))
	f.Add([]byte(` {"a":1}`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))
	f.Add([]byte(`123`))
	f.Add([]byte{0x00})
	f.Add([]byte{0xff, 0xfe})

	f.Fuzz(func(t *testing.T, data []byte) {
		// must not panic; result is either true or false
		result := zenrpc.IsArray(json.RawMessage(data))
		assert.IsType(t, true, result)
	})
}

// FuzzConvertToObject checks that ConvertToObject never panics
// and returns valid JSON on success.
func FuzzConvertToObject(f *testing.F) {
	f.Add([]byte(`[1,2]`))
	f.Add([]byte(`[3]`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`[1,2,3,4,5]`))
	f.Add([]byte(`["a","b"]`))
	f.Add([]byte(`[null,null]`))
	f.Add([]byte(`[true,false]`))
	f.Add([]byte(`[1.5,2.5]`))
	f.Add([]byte(`[{"nested":1},[1,2]]`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))
	f.Add([]byte(`{}`))

	keys := []string{"a", "b", "c"}

	f.Fuzz(func(t *testing.T, data []byte) {
		result, err := zenrpc.ConvertToObject(keys, json.RawMessage(data))
		if err != nil {
			return // errors are fine, panics are not
		}

		// on success, result must be valid JSON
		require.True(t, json.Valid(result), "ConvertToObject returned invalid JSON: %s", result)

		// result must be a JSON object
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(result, &m), "ConvertToObject result is not an object")
	})
}

// FuzzServeHTTP tests the full HTTP handler with arbitrary request bodies.
// Invariants:
//   - must never panic
//   - must return valid HTTP status code
//   - non-empty response body must be valid JSON
func FuzzServeHTTP(f *testing.F) {
	// seed corpus: valid requests
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.pi","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":0}`))
	f.Add([]byte(`[{"jsonrpc":"2.0","method":"arith.pi","id":1}]`))

	// seed corpus: malformed
	f.Add([]byte(`{invalid}`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))

	// seed corpus: edge cases
	f.Add([]byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2}}`)) // notification
	f.Add([]byte(`{"jsonrpc":"1.0","method":"arith.pi","id":1}`))                       // wrong version
	f.Add([]byte(`{"jsonrpc":"2.0","method":"","id":1}`))                               // empty method

	f.Fuzz(func(t *testing.T, body []byte) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		testRPC.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		// status must be valid
		assert.True(t, resp.StatusCode >= 100 && resp.StatusCode < 600,
			"invalid HTTP status: %d", resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "failed to read response body")

		// empty body is valid (notifications)
		if len(respBody) == 0 {
			return
		}

		// non-empty body must be valid JSON
		assert.True(t, json.Valid(respBody), "ServeHTTP returned invalid JSON: %s", respBody)
	})
}
