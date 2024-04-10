package protocol

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIncomingData(t *testing.T) {
	ctx := context.Background()
	data := map[string]interface{}{"key": []string{"value"}, "key2": []string{"value2"}}
	// set check
	ctx = SetIncomingData(ctx, data)
	testData := ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[incomingKey].(map[string]interface{})
	assert.Equal(t, data, testData)
	// append check
	ctx, err := AppendIncomingData(ctx, "hello", "world")
	assert.Equal(t, nil, err)
	data["hello"] = "world"
	testData = ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[incomingKey].(map[string]interface{})
	assert.Equal(t, data, testData)
	// get check
	testData, ok := GetIncomingData(ctx)
	assert.Equal(t, true, ok)
	assert.Equal(t, data, testData)

	// test err
	ctx = context.Background()
	testData, ok = GetIncomingData(ctx)
	assert.Equal(t, false, ok)
	assert.Nil(t, testData)

	ctx, err = AppendIncomingData(ctx, "hello", "world")
	assert.Equal(t, nil, err)
	assert.Equal(t, map[string]interface{}{
		"hello": []string{"world"},
	}, ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[incomingKey].(map[string]interface{}))

	ctx1, err := AppendIncomingData(ctx, "hello", "world", "err input")
	assert.Equal(t, ctx, ctx1)
	assert.NotEqual(t, nil, err)
}

func TestOutgoingData(t *testing.T) {
	ctx := context.Background()
	data := map[string]interface{}{"key": []string{"value"}, "key2": []string{"value2"}}
	// set check
	ctx = SetOutgoingData(ctx, data)
	testData := ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[outgoingKey].(map[string]interface{})
	assert.Equal(t, data, testData)
	// append check
	ctx, err := AppendOutgoingData(ctx, "hello", "world")
	assert.Equal(t, nil, err)
	data["hello"] = "world"
	testData = ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[outgoingKey].(map[string]interface{})
	assert.Equal(t, data, testData)
	// get check
	testData, ok := GetOutgoingData(ctx)
	assert.Equal(t, true, ok)
	assert.Equal(t, data, testData)

	// test err
	ctx = context.Background()
	testData, ok = GetOutgoingData(ctx)
	assert.Equal(t, false, ok)
	assert.Nil(t, testData)

	ctx, err = AppendOutgoingData(ctx, "hello", "world")
	assert.Equal(t, nil, err)
	assert.Equal(t, map[string]interface{}{
		"hello": []string{"world"},
	}, ctx.Value(rpcExtraDataKey{}).(map[string]interface{})[outgoingKey].(map[string]interface{}))

	ctx1, err := AppendOutgoingData(ctx, "hello", "world", "err input")
	assert.Equal(t, ctx, ctx1)
	assert.NotEqual(t, nil, err)
}
