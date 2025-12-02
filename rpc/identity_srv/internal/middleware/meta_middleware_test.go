package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetaInfoMiddleware(t *testing.T) {
	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.Default()
		middleware := NewMetaInfoMiddleware(logger)

		assert.NotNil(t, middleware)
		assert.Equal(t, logger, middleware.logger)
	})

	t.Run("with nil logger uses default", func(t *testing.T) {
		middleware := NewMetaInfoMiddleware(nil)

		assert.NotNil(t, middleware)
		assert.NotNil(t, middleware.logger)
	})
}

func TestMetaInfoMiddleware_ServerMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		existingMeta    map[string]string
		expectGenerated bool
		validateFunc    func(*testing.T, context.Context, *bytes.Buffer)
	}{
		{
			name: "both IDs exist - no generation needed",
			existingMeta: map[string]string{
				"request_id": "existing-request-id",
				"trace_id":   "existing-trace-id",
			},
			expectGenerated: false,
			validateFunc: func(t *testing.T, ctx context.Context, logBuf *bytes.Buffer) {
				assert.Equal(t, "existing-request-id", GetRequestID(ctx))
				assert.Equal(t, "existing-trace-id", GetTraceID(ctx))

				// Should not contain warning about generation
				logOutput := logBuf.String()
				assert.NotContains(t, logOutput, "Generated missing request_id")
			},
		},
		{
			name:            "no IDs exist - generate both",
			existingMeta:    map[string]string{},
			expectGenerated: true,
			validateFunc: func(t *testing.T, ctx context.Context, logBuf *bytes.Buffer) {
				requestID := GetRequestID(ctx)
				traceID := GetTraceID(ctx)

				assert.NotEmpty(t, requestID)
				assert.NotEmpty(t, traceID)
				assert.Equal(
					t,
					requestID,
					traceID,
				) // trace_id should equal request_id when generated

				// Should contain warning about generation
				logOutput := logBuf.String()
				assert.Contains(t, logOutput, "Generated missing request_id")
				assert.Contains(t, logOutput, requestID)
			},
		},
		{
			name: "only request_id exists - generate trace_id",
			existingMeta: map[string]string{
				"request_id": "existing-request-id",
			},
			expectGenerated: true,
			validateFunc: func(t *testing.T, ctx context.Context, logBuf *bytes.Buffer) {
				assert.Equal(t, "existing-request-id", GetRequestID(ctx))
				assert.Equal(
					t,
					"existing-request-id",
					GetTraceID(ctx),
				) // trace_id should equal request_id

				// Should not contain warning (only warns for request_id generation)
				logOutput := logBuf.String()
				assert.NotContains(t, logOutput, "Generated missing request_id")
			},
		},
		{
			name: "only trace_id exists - generate request_id",
			existingMeta: map[string]string{
				"trace_id": "existing-trace-id",
			},
			expectGenerated: true,
			validateFunc: func(t *testing.T, ctx context.Context, logBuf *bytes.Buffer) {
				requestID := GetRequestID(ctx)
				traceID := GetTraceID(ctx)

				assert.NotEmpty(t, requestID)
				// trace_id should remain unchanged since it already existed
				assert.Equal(t, "existing-trace-id", traceID)

				// Should contain warning about generation
				logOutput := logBuf.String()
				assert.Contains(t, logOutput, "Generated missing request_id")
			},
		},
		{
			name: "empty string IDs - should regenerate",
			existingMeta: map[string]string{
				"request_id": "",
				"trace_id":   "",
			},
			expectGenerated: true,
			validateFunc: func(t *testing.T, ctx context.Context, logBuf *bytes.Buffer) {
				requestID := GetRequestID(ctx)
				traceID := GetTraceID(ctx)

				assert.NotEmpty(t, requestID)
				assert.NotEmpty(t, traceID)
				assert.Equal(t, requestID, traceID)

				// Should contain warning about generation
				logOutput := logBuf.String()
				assert.Contains(t, logOutput, "Generated missing request_id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logger with buffer to capture logs
			var logBuf bytes.Buffer

			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			middleware := NewMetaInfoMiddleware(logger)
			serverMiddleware := middleware.ServerMiddleware()

			// Create context with existing meta info
			ctx := createContextWithMeta(tt.existingMeta)

			// Mock endpoint to capture final context
			var finalCtx context.Context

			mockEndpoint := func(ctx context.Context, req, resp interface{}) error {
				finalCtx = ctx
				return nil
			}

			// Apply middleware
			wrappedEndpoint := serverMiddleware(mockEndpoint)
			err := wrappedEndpoint(ctx, nil, nil)

			// Verify no error
			require.NoError(t, err)
			require.NotNil(t, finalCtx)

			// Run validation
			tt.validateFunc(t, finalCtx, &logBuf)
		})
	}
}

func TestMetaInfoMiddleware_ensureTraceIDs(t *testing.T) {
	middleware := NewMetaInfoMiddleware(slog.Default())

	t.Run("generates valid UUIDs", func(t *testing.T) {
		ctx := context.Background()

		resultCtx := middleware.ensureTraceIDs(ctx)

		requestID := GetRequestID(resultCtx)
		traceID := GetTraceID(resultCtx)

		// Verify UUIDs are generated and valid format
		assert.NotEmpty(t, requestID)
		assert.NotEmpty(t, traceID)
		assert.Equal(t, requestID, traceID)

		// Basic UUID format check (36 characters with hyphens)
		assert.Len(t, requestID, 36)
		assert.Contains(t, requestID, "-")
	})

	t.Run("preserves existing IDs", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "preserve-me",
			"trace_id":   "preserve-me-too",
		})

		resultCtx := middleware.ensureTraceIDs(ctx)

		assert.Equal(t, "preserve-me", GetRequestID(resultCtx))
		assert.Equal(t, "preserve-me-too", GetTraceID(resultCtx))
	})
}

func TestMetaInfoMiddleware_logTraceInfo(t *testing.T) {
	var logBuf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	middleware := NewMetaInfoMiddleware(logger)

	t.Run("logs trace info when IDs present", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "test-request-id",
			"trace_id":   "test-trace-id",
		})

		middleware.logTraceInfo(ctx)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "RPC request received")
		assert.Contains(t, logOutput, "test-request-id")
		assert.Contains(t, logOutput, "test-trace-id")
		assert.Contains(t, logOutput, "middleware=trace")
	})

	t.Run("handles empty context gracefully", func(t *testing.T) {
		logBuf.Reset()

		ctx := context.Background()

		middleware.logTraceInfo(ctx)

		// Should not log anything when no trace info is available
		logOutput := logBuf.String()
		assert.Empty(t, logOutput)
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns ID when present", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "test-request-id",
		})

		result := GetRequestID(ctx)
		assert.Equal(t, "test-request-id", result)
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()

		result := GetRequestID(ctx)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string for empty value", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "",
		})

		result := GetRequestID(ctx)
		assert.Equal(t, "", result)
	})
}

func TestGetTraceID(t *testing.T) {
	t.Run("returns ID when present", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"trace_id": "test-trace-id",
		})

		result := GetTraceID(ctx)
		assert.Equal(t, "test-trace-id", result)
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()

		result := GetTraceID(ctx)
		assert.Equal(t, "", result)
	})
}

func TestLoggingAttrs(t *testing.T) {
	t.Run("returns attributes for both IDs", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "test-request-id",
			"trace_id":   "test-trace-id",
		})

		attrs := LoggingAttrs(ctx)

		assert.Len(t, attrs, 2)

		// Convert to map for easier testing
		attrMap := make(map[string]string)
		for _, attr := range attrs {
			attrMap[attr.Key] = attr.Value.String()
		}

		assert.Equal(t, "test-request-id", attrMap["request_id"])
		assert.Equal(t, "test-trace-id", attrMap["trace_id"])
	})

	t.Run("returns only available attributes", func(t *testing.T) {
		ctx := createContextWithMeta(map[string]string{
			"request_id": "test-request-id",
		})

		attrs := LoggingAttrs(ctx)

		assert.Len(t, attrs, 1)
		assert.Equal(t, "request_id", attrs[0].Key)
		assert.Equal(t, "test-request-id", attrs[0].Value.String())
	})

	t.Run("returns empty slice when no IDs present", func(t *testing.T) {
		ctx := context.Background()

		attrs := LoggingAttrs(ctx)

		assert.Len(t, attrs, 0)
	})
}

func TestMetaInfoMiddleware_Integration(t *testing.T) {
	t.Run("complete middleware flow with ID generation", func(t *testing.T) {
		var logBuf bytes.Buffer

		logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		middleware := NewMetaInfoMiddleware(logger)
		serverMiddleware := middleware.ServerMiddleware()

		// Start with empty context
		ctx := context.Background()

		// Business logic that uses the trace IDs
		businessLogic := func(ctx context.Context, req, resp interface{}) error {
			// Verify IDs are available in business logic
			requestID := GetRequestID(ctx)
			traceID := GetTraceID(ctx)

			assert.NotEmpty(t, requestID)
			assert.NotEmpty(t, traceID)
			assert.Equal(t, requestID, traceID)

			// Verify logging attributes work
			attrs := LoggingAttrs(ctx)
			assert.Len(t, attrs, 2)

			return nil
		}

		// Apply middleware and execute
		wrappedEndpoint := serverMiddleware(businessLogic)
		err := wrappedEndpoint(ctx, nil, nil)

		require.NoError(t, err)

		// Verify logging occurred
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Generated missing request_id")
		assert.Contains(t, logOutput, "RPC request received")
		assert.Contains(t, logOutput, "identity_srv")
	})

	t.Run("middleware handles metainfo correctly", func(t *testing.T) {
		middleware := NewMetaInfoMiddleware(slog.Default())
		serverMiddleware := middleware.ServerMiddleware()

		// Create context with metainfo values
		ctx := createContextWithMeta(map[string]string{
			"request_id": "existing-id",
		})

		businessLogic := func(ctx context.Context, req, resp interface{}) error {
			// Verify trace IDs are handled
			assert.Equal(t, "existing-id", GetRequestID(ctx))
			assert.Equal(t, "existing-id", GetTraceID(ctx)) // Should be set to request_id

			return nil
		}

		wrappedEndpoint := serverMiddleware(businessLogic)
		err := wrappedEndpoint(ctx, nil, nil)

		require.NoError(t, err)
	})
}

// Helper functions

// createContextWithMeta creates a context with metainfo values
func createContextWithMeta(metaInfo map[string]string) context.Context {
	ctx := context.Background()

	for key, value := range metaInfo {
		ctx = metainfo.WithPersistentValue(ctx, key, value)
	}

	return ctx
}

// Benchmark tests for performance validation

func BenchmarkMetaInfoMiddleware_WithExistingIDs(b *testing.B) {
	middleware := NewMetaInfoMiddleware(slog.Default())
	serverMiddleware := middleware.ServerMiddleware()

	ctx := createContextWithMeta(map[string]string{
		"request_id": "benchmark-request-id",
		"trace_id":   "benchmark-trace-id",
	})

	mockEndpoint := func(ctx context.Context, req, resp interface{}) error {
		return nil
	}

	wrappedEndpoint := serverMiddleware(mockEndpoint)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = wrappedEndpoint(ctx, nil, nil)
	}
}

func BenchmarkMetaInfoMiddleware_WithGeneration(b *testing.B) {
	middleware := NewMetaInfoMiddleware(slog.Default())
	serverMiddleware := middleware.ServerMiddleware()

	ctx := context.Background()

	mockEndpoint := func(ctx context.Context, req, resp interface{}) error {
		return nil
	}

	wrappedEndpoint := serverMiddleware(mockEndpoint)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = wrappedEndpoint(ctx, nil, nil)
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	ctx := createContextWithMeta(map[string]string{
		"request_id": "benchmark-request-id",
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GetRequestID(ctx)
	}
}

func BenchmarkLoggingAttrs(b *testing.B) {
	ctx := createContextWithMeta(map[string]string{
		"request_id": "benchmark-request-id",
		"trace_id":   "benchmark-trace-id",
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = LoggingAttrs(ctx)
	}
}
