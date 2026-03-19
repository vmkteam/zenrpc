package zenrpc_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/vmkteam/zenrpc/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ErrorMsg ---

func TestErrorMsg(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{zenrpc.ParseError, "Parse error"},
		{zenrpc.InvalidRequest, "Invalid Request"},
		{zenrpc.MethodNotFound, "Method not found"},
		{zenrpc.InvalidParams, "Invalid params"},
		{zenrpc.InternalError, "Internal error"},
		{zenrpc.ServerError, "Server error"},
		{-99999, ""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			assert.Equal(t, tt.want, zenrpc.ErrorMsg(tt.code))
		})
	}
}

// --- NewResponseError ---

func TestNewResponseError(t *testing.T) {
	t.Run("StandardMessage", func(t *testing.T) {
		r := zenrpc.NewResponseError(nil, zenrpc.ParseError, "", nil)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.ParseError, r.Error.Code)
		assert.Equal(t, "Parse error", r.Error.Message)
		assert.Equal(t, zenrpc.Version, r.Version)
		assert.Nil(t, r.ID)
	})

	t.Run("CustomMessage", func(t *testing.T) {
		r := zenrpc.NewResponseError(nil, zenrpc.InternalError, "custom msg", nil)
		require.NotNil(t, r.Error)
		assert.Equal(t, "custom msg", r.Error.Message)
	})

	t.Run("WithData", func(t *testing.T) {
		r := zenrpc.NewResponseError(nil, zenrpc.InternalError, "", "extra")
		require.NotNil(t, r.Error)
		assert.Equal(t, "extra", r.Error.Data)
	})

	t.Run("WithID", func(t *testing.T) {
		id := json.RawMessage(`42`)
		r := zenrpc.NewResponseError(&id, zenrpc.InternalError, "", nil)
		require.NotNil(t, r.ID)
		assert.Equal(t, "42", string(*r.ID))
	})
}

// --- NewStringError ---

func TestNewStringError(t *testing.T) {
	e := zenrpc.NewStringError(zenrpc.InvalidParams, "bad param")
	assert.Equal(t, zenrpc.InvalidParams, e.Code)
	assert.Equal(t, "bad param", e.Message)
	assert.NoError(t, e.Err)
}

// --- NewError ---

func TestNewError(t *testing.T) {
	inner := errors.New("something went wrong")
	e := zenrpc.NewError(zenrpc.InternalError, inner)
	assert.Equal(t, zenrpc.InternalError, e.Code)
	assert.Equal(t, "something went wrong", e.Message)
	require.ErrorIs(t, e, inner)
}

// --- Error.Error ---

func TestError_Error(t *testing.T) {
	t.Run("FromErr", func(t *testing.T) {
		e := &zenrpc.Error{Code: zenrpc.InternalError, Err: errors.New("inner")}
		assert.Equal(t, "inner", e.Error())
	})

	t.Run("FromMessage", func(t *testing.T) {
		e := &zenrpc.Error{Code: zenrpc.InternalError, Message: "custom"}
		assert.Equal(t, "custom", e.Error())
	})

	t.Run("FromCode", func(t *testing.T) {
		e := &zenrpc.Error{Code: zenrpc.InternalError}
		assert.Equal(t, "Internal error", e.Error())
	})
}

// --- Error.Unwrap ---

func TestError_Unwrap(t *testing.T) {
	inner := errors.New("root cause")
	e := zenrpc.NewError(zenrpc.InternalError, inner)

	require.ErrorIs(t, e, inner)
	assert.Equal(t, inner, errors.Unwrap(e))
}

// --- Response.Set ---

func TestResponse_Set(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(42)
		require.NotNil(t, r.Result)
		assert.Nil(t, r.Error)
		assert.Equal(t, "42", string(*r.Result))
	})

	t.Run("StructValue", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(struct{ X int }{1})
		require.NotNil(t, r.Result)
		assert.Nil(t, r.Error)
		assert.JSONEq(t, `{"X":1}`, string(*r.Result))
	})

	t.Run("GoError", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(nil, errors.New("boom"))
		assert.Nil(t, r.Result)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.InternalError, r.Error.Code)
	})

	t.Run("ZenrpcError", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(nil, zenrpc.NewStringError(zenrpc.InvalidParams, "bad"))
		assert.Nil(t, r.Result)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.InvalidParams, r.Error.Code)
		assert.Equal(t, "bad", r.Error.Message)
	})

	t.Run("NilZenrpcError", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(nil, (*zenrpc.Error)(nil))
		require.NotNil(t, r.Result)
		assert.Nil(t, r.Error)
		assert.Equal(t, "null", string(*r.Result))
	})

	t.Run("ErrorAsValue", func(t *testing.T) {
		var r zenrpc.Response
		r.Set(errors.New("boom"))
		assert.Nil(t, r.Result)
		require.NotNil(t, r.Error)
		assert.Equal(t, zenrpc.InternalError, r.Error.Code)
	})
}

// --- Response.JSON ---

func TestResponse_JSON(t *testing.T) {
	r := zenrpc.NewResponseError(nil, zenrpc.MethodNotFound, "", nil)
	b := r.JSON()

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Contains(t, m, "error")
	assert.Equal(t, `"2.0"`, string(m["jsonrpc"]))
}
