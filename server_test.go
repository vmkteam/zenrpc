package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Setup & helpers ---

var (
	testRPC     = zenrpc.NewServer(zenrpc.Options{BatchMaxLen: 5, AllowCORS: true})
	logRequests = false
)

func TestMain(m *testing.M) {
	testRPC.Register("arith", &testdata.ArithService{})
	testRPC.Register("", &testdata.ArithService{})

	if logRequests {
		testRPC.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
	}

	result := m.Run()
	os.Exit(result)
}

// doRPC calls srv.Do and unmarshals the response into zenrpc.Response.
func doRPC(t *testing.T, srv *zenrpc.Server, req string) zenrpc.Response {
	t.Helper()

	b, err := srv.Do(context.Background(), []byte(req))
	require.NoError(t, err, "Do() error")

	var r zenrpc.Response
	require.NoError(t, json.Unmarshal(b, &r), "Unmarshal response, raw: %s", b)

	return r
}

// doRPCRaw calls srv.Do and unmarshals the response into a raw JSON map,
// useful for testing presence/absence of specific JSON keys.
func doRPCRaw(t *testing.T, srv *zenrpc.Server, req string) map[string]json.RawMessage {
	t.Helper()

	b, err := srv.Do(context.Background(), []byte(req))
	require.NoError(t, err, "Do() error")

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m), "Unmarshal raw response, raw: %s", b)

	return m
}

// buildBatchRequest builds a JSON-RPC 2.0 batch request with n multiply calls.
func buildBatchRequest(n int) []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := range n {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, `{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":%d,"b":2},"id":%d}`, i+1, i)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

// --- SMD ---

func TestServer_SMD(t *testing.T) {
	r := testRPC.SMD()
	b, err := json.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(b), "default")
}

func TestServer_SmdGenerate(t *testing.T) {
	rpc := zenrpc.NewServer(zenrpc.Options{})
	rpc.Register("phonebook", testdata.PhoneBook{DB: testdata.People})
	rpc.Register("arith", testdata.ArithService{})
	rpc.Register("printer", testdata.PrintService{})
	rpc.Register("", testdata.ArithService{})

	b, err := json.MarshalIndent(rpc.SMD(), "", "  ")
	require.NoError(t, err)

	testData, err := os.ReadFile("./testdata/testdata/arithsrv-smd.json")
	require.NoError(t, err, "open test data file")

	assert.Equal(t, string(testData), string(b), "bad zenrpc output")
}

// --- Server unit tests ---

func TestNewServer_Defaults(t *testing.T) {
	srv := zenrpc.NewServer(zenrpc.Options{})
	srv.Register("arith", &testdata.ArithService{})

	t.Run("BatchMaxLen10OK", func(t *testing.T) {
		resp, err := srv.Do(context.Background(), buildBatchRequest(10))
		require.NoError(t, err)

		var results []zenrpc.Response
		require.NoError(t, json.Unmarshal(resp, &results))
		assert.Len(t, results, 10)
	})

	t.Run("BatchMaxLen11Fail", func(t *testing.T) {
		r := doRPC(t, srv, string(buildBatchRequest(11)))
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.InvalidRequest, r.Error.Code)
	})

	t.Run("DefaultTargetURL", func(t *testing.T) {
		srv := zenrpc.NewServer(zenrpc.Options{})
		assert.Equal(t, "/", srv.SMD().Target)
	})

	t.Run("CustomTargetURL", func(t *testing.T) {
		srv := zenrpc.NewServer(zenrpc.Options{TargetURL: "/custom"})
		assert.Equal(t, "/custom", srv.SMD().Target)
	})
}

func TestServer_RegisterAll(t *testing.T) {
	srv := zenrpc.NewServer(zenrpc.Options{})
	srv.RegisterAll(map[string]zenrpc.Invoker{
		"arith": &testdata.ArithService{},
	})

	r := doRPC(t, srv, `{"jsonrpc":"2.0","method":"arith.pi","id":1}`)
	assert.Nil(t, r.Error)
	assert.NotNil(t, r.Result)
}

func TestServer_RegisterOverwrite(t *testing.T) {
	srv := zenrpc.NewServer(zenrpc.Options{})
	srv.Register("svc", &testdata.ArithService{})

	// Verify ArithService method works before overwrite.
	r := doRPC(t, srv, `{"jsonrpc":"2.0","method":"svc.pi","id":1}`)
	assert.Nil(t, r.Error)
	assert.NotNil(t, r.Result)

	// Overwrite with PrintService — ArithService methods should no longer be available.
	srv.Register("svc", &testdata.PrintService{})

	r = doRPC(t, srv, `{"jsonrpc":"2.0","method":"svc.pi","id":2}`)
	require.NotNil(t, r.Error, "old service method should not be available after overwrite")
	assert.Equal(t, zenrpc.MethodNotFound, r.Error.Code)
}

func TestIsArray_Whitespace(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want bool
	}{
		{"array", `[1]`, true},
		{"object", `{"a":1}`, false},
		{"space_before_array", " [1]", true},
		{"tab_before_array", "\t[1]", true},
		{"newline_before_array", "\n[1]", true},
		{"crlf_before_array", "\r\n[1]", true},
		{"space_before_object", " {\"a\":1}", false},
		{"tab_before_object", "\t{\"a\":1}", false},
		{"newline_before_object", "\n{\"a\":1}", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, zenrpc.IsArray(json.RawMessage(tt.msg)))
		})
	}
}

func TestConvertToObject_EdgeCases(t *testing.T) {
	t.Run("TooManyParams", func(t *testing.T) {
		keys := []string{"a"}
		params := json.RawMessage(`[1,2]`)
		_, err := zenrpc.ConvertToObject(keys, params)
		assert.Error(t, err)
	})

	t.Run("FewerParams", func(t *testing.T) {
		keys := []string{"a", "b", "c"}
		params := json.RawMessage(`[1]`)
		result, err := zenrpc.ConvertToObject(keys, params)
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(result))
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		keys := []string{"a"}
		params := json.RawMessage(`not json`)
		_, err := zenrpc.ConvertToObject(keys, params)
		assert.Error(t, err)
	})

	t.Run("EmptyArray", func(t *testing.T) {
		keys := []string{"a", "b"}
		params := json.RawMessage(`[]`)
		result, err := zenrpc.ConvertToObject(keys, params)
		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(result))
	})
}

func TestContextFunctions(t *testing.T) {
	t.Run("EmptyContext", func(t *testing.T) {
		ctx := context.Background()

		_, ok := zenrpc.RequestFromContext(ctx)
		assert.False(t, ok)

		assert.Empty(t, zenrpc.NamespaceFromContext(ctx))
		assert.Nil(t, zenrpc.IDFromContext(ctx))
		assert.Nil(t, zenrpc.BatchMethodsFromContext(ctx))
	})

	t.Run("RequestRoundTrip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		ctx := zenrpc.NewRequestContext(context.Background(), req)

		got, ok := zenrpc.RequestFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, req, got)
	})
}

// --- JSON-RPC 2.0 spec compliance ---

func TestDo_IDFormats(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"string", `"42"`},
		{"negative", "-1"},
		{"fractional", "1.5"},
		{"large", "999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"arith.pi","id":%s}`, tt.id)
			r := doRPC(t, testRPC, req)
			assert.JSONEq(t, tt.id, string(*r.ID))
			assert.NotNil(t, r.Result, "result should be present")
		})
	}
}

// TestDo_NotificationNullID verifies that "id":null is treated as a notification
// (no response body). Go json.Unmarshal sets *json.RawMessage to nil for JSON null.
func TestDo_NotificationNullID(t *testing.T) {
	resp, err := testRPC.Do(context.Background(), []byte(`{"jsonrpc":"2.0","method":"arith.pi","id":null}`))
	require.NoError(t, err)
	assert.Equal(t, "null", string(resp))
}

func TestDo_Version(t *testing.T) {
	tests := []struct {
		name string
		req  string
	}{
		{"version_1.0", `{"jsonrpc":"1.0","method":"arith.pi","id":1}`},
		{"version_absent", `{"method":"arith.pi","id":1}`},
		{"version_empty", `{"jsonrpc":"","method":"arith.pi","id":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := doRPC(t, testRPC, tt.req)
			require.NotNil(t, r.Error, "expected error response")
			assert.Equal(t, zenrpc.InvalidRequest, r.Error.Code)
		})
	}
}

func TestDo_RpcDotReserved(t *testing.T) {
	r := doRPC(t, testRPC, `{"jsonrpc":"2.0","method":"rpc.discover","id":1}`)
	require.NotNil(t, r.Error)
	assert.Equal(t, zenrpc.MethodNotFound, r.Error.Code)
}

func TestDo_ParamsEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		req  string
	}{
		{"params_null", `{"jsonrpc":"2.0","method":"arith.pi","params":null,"id":1}`},
		{"params_empty_object", `{"jsonrpc":"2.0","method":"arith.pi","params":{},"id":1}`},
		{"params_empty_array", `{"jsonrpc":"2.0","method":"arith.pi","params":[],"id":1}`},
		{"params_omitted", `{"jsonrpc":"2.0","method":"arith.pi","id":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := doRPC(t, testRPC, tt.req)
			assert.Nil(t, r.Error, "expected success")
			assert.NotNil(t, r.Result)
		})
	}
}

func TestDo_BatchEdgeCases(t *testing.T) {
	t.Run("empty_array", func(t *testing.T) {
		// [] → single InvalidRequest (spec: "not an Array with at least one value")
		r := doRPC(t, testRPC, `[]`)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.InvalidRequest, r.Error.Code)
	})

	t.Run("non_object_element", func(t *testing.T) {
		// [1] → ParseError; Go cannot unmarshal the integer 1 into a Request struct,
		// so the entire batch fails to parse (deviation from spec which expects per-element InvalidRequest).
		r := doRPC(t, testRPC, `[1]`)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.ParseError, r.Error.Code)
	})

	t.Run("single_element_batch", func(t *testing.T) {
		// Batch with one element must return an array with one response, not a bare object.
		resp, err := testRPC.Do(context.Background(), []byte(`[{"jsonrpc":"2.0","method":"arith.pi","id":1}]`))
		require.NoError(t, err)

		var results []zenrpc.Response
		require.NoError(t, json.Unmarshal(resp, &results), "response must be a JSON array")
		require.Len(t, results, 1)
		assert.Nil(t, results[0].Error)
		assert.NotNil(t, results[0].Result)
	})
}

func TestDo_ResponseStructure(t *testing.T) {
	t.Run("SuccessHasResultNoError", func(t *testing.T) {
		m := doRPCRaw(t, testRPC, `{"jsonrpc":"2.0","method":"arith.pi","id":1}`)
		assert.Contains(t, m, "result")
		assert.NotContains(t, m, "error")
		assert.Equal(t, `"2.0"`, string(m["jsonrpc"]))
	})

	t.Run("ErrorHasErrorNoResult", func(t *testing.T) {
		m := doRPCRaw(t, testRPC, `{"jsonrpc":"2.0","method":"arith.checkerror","params":{"isErr":true},"id":1}`)
		assert.Contains(t, m, "error")
		assert.NotContains(t, m, "result")
		assert.Equal(t, `"2.0"`, string(m["jsonrpc"]))
	})

	t.Run("NullResultPresent", func(t *testing.T) {
		// checkerror with isErr=false returns nil error → result:null, no error field.
		m := doRPCRaw(t, testRPC, `{"jsonrpc":"2.0","method":"arith.checkerror","params":{"isErr":false},"id":1}`)
		assert.Contains(t, m, "result", `"result" must be present even when null`)
		assert.NotContains(t, m, "error")
		assert.Equal(t, "null", string(m["result"]))
	})

	t.Run("AlwaysHasVersion", func(t *testing.T) {
		// Even error responses must have jsonrpc:"2.0".
		m := doRPCRaw(t, testRPC, `{"jsonrpc":"2.0","method":"nonexistent.method","id":1}`)
		assert.Equal(t, `"2.0"`, string(m["jsonrpc"]))
	})

	t.Run("ParseErrorHasNullID", func(t *testing.T) {
		m := doRPCRaw(t, testRPC, `{invalid json`)
		assert.Contains(t, m, "id")
		assert.Equal(t, "null", string(m["id"]))
	})
}

// --- Concurrency ---

func TestDo_Parallel(t *testing.T) {
	t.Parallel()
	const n = 100

	for i := range n {
		a := i + 1
		t.Run(fmt.Sprintf("g%d", i), func(t *testing.T) {
			t.Parallel()

			req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":%d,"b":2},"id":%d}`, a, a)
			r := doRPC(t, testRPC, req)

			var result int
			require.NoError(t, json.Unmarshal(*r.Result, &result))
			assert.Equal(t, a*2, result)
		})
	}
}

func TestDo_BatchParallel(t *testing.T) {
	t.Parallel()
	const n = 50

	for i := range n {
		t.Run(fmt.Sprintf("g%d", i), func(t *testing.T) {
			t.Parallel()

			resp, err := testRPC.Do(context.Background(), buildBatchRequest(2))
			require.NoError(t, err)

			var results []zenrpc.Response
			require.NoError(t, json.Unmarshal(resp, &results))
			assert.Len(t, results, 2)
		})
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Server must handle a cancelled context without panic and return valid JSON-RPC.
	resp, err := testRPC.Do(ctx, []byte(`{"jsonrpc":"2.0","method":"arith.pi","id":1}`))
	require.NoError(t, err)

	var r zenrpc.Response
	require.NoError(t, json.Unmarshal(resp, &r))
	assert.Equal(t, zenrpc.Version, r.Version)
}
