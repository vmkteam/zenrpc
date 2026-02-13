package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"
)

// --- Low-level utilities ---

func BenchmarkIsArray(b *testing.B) {
	object := json.RawMessage(`{"jsonrpc":"2.0","method":"arith.pi","id":1}`)
	array := json.RawMessage(`[{"jsonrpc":"2.0","method":"arith.pi","id":1}]`)

	b.Run("Object", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(object)))
		for b.Loop() {
			zenrpc.IsArray(object)
		}
	})
	b.Run("Array", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(array)))
		for b.Loop() {
			zenrpc.IsArray(array)
		}
	})
}

func BenchmarkConvertToObject(b *testing.B) {
	keys := []string{"a", "b"}
	params := json.RawMessage(`[3,2]`)

	b.ReportAllocs()
	b.SetBytes(int64(len(params)))
	for b.Loop() {
		_, _ = zenrpc.ConvertToObject(keys, params)
	}
}

// --- Core processing via Do() ---

// benchDo is a helper that runs a Do() benchmark with throughput and response size metrics.
func benchDo(b *testing.B, srv *zenrpc.Server, req []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(req)))
	ctx := context.Background()

	for b.Loop() {
		_, _ = srv.Do(ctx, req)
	}
}

func BenchmarkDo_SimpleMethod(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.pi","id":1}`))
}

func BenchmarkDo_MethodWithObjectParams(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":1}`))
}

func BenchmarkDo_MethodWithArrayParams(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":[3,2],"id":1}`))
}

func BenchmarkDo_MethodWithDefaultParam(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.pow","params":{"base":3},"id":1}`))
}

func BenchmarkDo_MethodNotFound(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.nonexistent","id":1}`))
}

func BenchmarkDo_Notification(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2}}`))
}

func BenchmarkDo_InvalidJSON(b *testing.B) {
	benchDo(b, testRPC, []byte(`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`))
}

// --- Batch processing via Do() ---

func BenchmarkDo_Batch(b *testing.B) {
	for _, size := range []int{1, 2, 5} {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			req := buildBatchRequest(size)
			b.ReportAllocs()
			b.SetBytes(int64(len(req)))
			ctx := context.Background()

			for b.Loop() {
				_, _ = testRPC.Do(ctx, req)
			}
		})
	}
}

// --- HTTP transport via ServeHTTP ---

func BenchmarkServeHTTP(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	body := []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":1}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	for b.Loop() {
		resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkServeHTTP_Parallel(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(testRPC.ServeHTTP))
	defer ts.Close()

	body := []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":1}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// --- Middleware overhead ---

func BenchmarkDo_WithMiddleware(b *testing.B) {
	req := []byte(`{"jsonrpc":"2.0","method":"arith.multiply","params":{"a":3,"b":2},"id":1}`)

	b.Run("NoMiddleware", func(b *testing.B) {
		srv := zenrpc.NewServer(zenrpc.Options{})
		srv.Register("arith", &testdata.ArithService{})
		benchDo(b, srv, req)
	})

	b.Run("WithLogger", func(b *testing.B) {
		srv := zenrpc.NewServer(zenrpc.Options{})
		srv.Register("arith", &testdata.ArithService{})
		srv.Use(zenrpc.Logger(log.New(io.Discard, "", 0)))
		benchDo(b, srv, req)
	})
}
