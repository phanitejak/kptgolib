package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/tracing/tracingtest"
)

const OperationID = "afa0a9f161f6442e8c0cb626697f5393"

func TestGetOperationIDFromContext(t *testing.T) {
	// Must fail without global tracing
	_, err := GetOperationIDFromContext(context.Background())
	require.Error(t, err)

	cleanTracing := tracingtest.SetUp(t)
	defer cleanTracing()

	// Must return an error for empty context
	opID, err := GetOperationIDFromContext(context.Background())
	require.Error(t, err)
	assert.Equal(t, "", opID)

	// Must succeed when tracing is fully enabled and operation context is used
	span, ctx, err := SpanFromOID(OperationID, "test-operation")
	require.NoError(t, err)
	defer span.Finish()

	opID, err = GetOperationIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, OperationID, opID)
	cleanTracing()
}

func TestSpanFromOID(t *testing.T) {
	span, ctx, err := SpanFromOID("", "test-operation")
	require.Error(t, err)
	assert.Nil(t, span)
	assert.Nil(t, ctx)

	span, ctx, err = SpanFromOID(OperationID, "test-operation")
	require.NoError(t, err)
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)

	opID, err := GetOperationIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, OperationID, opID)
}
