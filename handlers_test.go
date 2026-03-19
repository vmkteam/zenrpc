package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServeHTTPWithHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		h string
		s int
	}{
		{
			h: "application/json",
			s: 200,
		},
		{
			h: "application/json; charset=utf-8",
			s: 200,
		},
		{
			h: "application/text; charset=utf-8",
			s: 415,
		},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, c.h, bytes.NewBufferString(`{"jsonrpc": "2.0", "method": "arith.pi", "id": 2 }`))
		require.NoError(t, err)
		assert.Equal(t, c.s, res.StatusCode, "Input: %s", c.h)
		res.Body.Close()
	}
}

func TestServer_ServeHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"rem":1}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": [ 1, 24 ], "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"rem":1}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 0 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"divide by zero"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "Arith.Divide", "params": { "a": 1, "b": 1 }, "id": "1" }`,
			out: `{"jsonrpc":"2.0","id":"1","error":{"code":401,"message":"we do not serve 1"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":6}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "multiply", "params": { "a": 4, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":8}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3, "exp": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":27}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.sum", "params": { "a": 3, "b": 3 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":6,"message":"` + ts.Listener.Addr().String() + `"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":9}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": [ 3 ], "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":9}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": [ 3, 3 ], "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":27}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pi", "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":3.141592653589793}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.checkerror", "id": 0, "params": [ false ] }`,
			out: `{"jsonrpc":"2.0","id":0,"result":null}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.checkerror", "id": 0, "params": [ true ] }`,
			out: `{"jsonrpc":"2.0","id":0,"error":{"code":-32603,"message":"test"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.checkzenrpcerror", "id": 0, "params": [ false ] }`,
			out: `{"jsonrpc":"2.0","id":0,"result":null}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.checkzenrpcerror", "id": 0, "params": [ true ] }`,
			out: `{"jsonrpc":"2.0","id":0,"error":{"code":500,"message":"test"}}`},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(c.in))
		require.NoError(t, err)

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, c.out, string(resp), "Input: %s", c.in)
	}
}

func TestServer_ServeHTTPNotifications(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }}`,
			out: ``},
		{
			// should be empty even with error
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 0 }}`,
			out: ``},
		{
			// but parse errors should be displayed
			in:  `{"jsonrpc": "1.0", "method": "Arith.Divide", "params": { "a": 1, "b": 1 }`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`},
		{
			// in batch requests notifications should not be listed in response
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 } }]`,
			out: `[{"jsonrpc":"2.0","id":0,"result":6}]`},
		{
			// order doesn't matter
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 } },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 0 }]`,
			out: `[{"jsonrpc":"2.0","id":0,"result":4}]`},
		{
			// all notifications
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 } },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }}]`,
			out: ``},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(c.in))
		require.NoError(t, err)

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, c.out, string(resp), "Input: %s", c.in)
	}
}

func TestServer_ServeHTTPBatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in  string
		out []string
	}{
		{
			// batch requests should process asynchronously, any order in responses accepted
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":1,"result":9}`,
				`{"jsonrpc":"2.0","id":0,"result":6}`,
				`{"jsonrpc":"2.0","id":2,"result":4}`}},
		{
			// one of the requests errored
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply1", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":1,"result":9}`,
				`{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}`,
				`{"jsonrpc":"2.0","id":2,"result":4}`}},
		{
			// to much batch requests
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply1", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"Invalid Request","data":"max requests length in batch exceeded"}}`}},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(c.in))
		require.NoError(t, err)

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		// checking if count of responses is correct
		cnt := strings.Count(string(resp), `"jsonrpc":"2.0"`)
		assert.Equal(t, len(c.out), cnt, "Input: %s\n batch count mismatch", c.in)

		// checking every response variant to be in response
		for _, check := range c.out {
			assert.Contains(t, string(resp), check, "Input: %s", c.in)
		}
	}
}

func TestServer_ServeHTTPWithErrors(t *testing.T) {
	rpcHiddenErrorField := zenrpc.NewServer(zenrpc.Options{AllowCORS: true, HideErrorDataField: true})
	rpcHiddenErrorField.Register("arith", &testdata.ArithService{})

	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	tsHid := httptest.NewServer(http.HandlerFunc(rpcHiddenErrorField.ServeHTTP))
	defer tsHid.Close()

	var tc = []struct {
		url     string
		in, out string
	}{
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "multiple1", "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "test.multiple1", "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "params": { "a": 1, "b": 0 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": 1, "params": "bar"}`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`,
			// in spec: {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
		},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": "3" }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"error":{"code":-32602,"message":"Invalid params","data":"json: cannot unmarshal string into Go struct field .base of type float64"}}`,
		},
		{
			url: tsHid.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": "3" }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"error":{"code":-32602,"message":"Invalid params"}}`,
		},
	}

	for _, c := range tc {
		res, err := http.Post(c.url, "application/json", bytes.NewBufferString(c.in))
		require.NoError(t, err)

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, c.out, string(resp), "Input: %s", c.in)
	}
}

func TestServer_Extensions(t *testing.T) {
	middleware := func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			r := h(ctx, method, params)

			// ignore multiply method
			if method != testdata.RPC.ArithService.Multiply {
				r.Extensions = map[string]interface{}{"debug": "true"}
			}

			return r
		}
	}

	server := zenrpc.NewServer(zenrpc.Options{AllowCORS: true, HideErrorDataField: true})
	server.Register("arith", &testdata.ArithService{})
	server.Use(middleware)

	ts := httptest.NewServer(http.HandlerFunc(server.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		url     string
		in, out string
	}{
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "multiple1", "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }}`,
			out: ``},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"rem":1},"extensions":{"debug":"true"}}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":24}`},
		{
			url: ts.URL,
			in:  `{"jsonrpc": "2.0", "method": "arith.checkerror", "params": [ true ], "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"test"},"extensions":{"debug":"true"}}`},
	}

	for _, c := range tc {
		res, err := http.Post(c.url, "application/json", bytes.NewBufferString(c.in))
		require.NoError(t, err)

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, c.out, string(resp), "Input: %s", c.in)
	}
}

func TestServer_ServeWS(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeWS))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)
	u.Scheme = "ws"

	//nolint:bodyclose
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer ws.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"rem":1}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 0 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"divide by zero"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "Arith.Divide", "params": { "a": 1, "b": 1 }, "id": "1" }`,
			out: `{"jsonrpc":"2.0","id":"1","error":{"code":401,"message":"we do not serve 1"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":6}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "multiply", "params": { "a": 4, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":8}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3, "exp": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":27}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.sum", "params": { "a": 3, "b": 3 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":6,"message":"` + ts.Listener.Addr().String() + `"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":9}`},
	}

	for _, c := range tc {
		err = ws.WriteMessage(websocket.TextMessage, []byte(c.in))
		require.NoError(t, err)

		var resp []byte
		_, resp, err = ws.ReadMessage()
		require.NoError(t, err)

		assert.Equal(t, c.out, string(resp), "Input: %s", c.in)
	}

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)
}
