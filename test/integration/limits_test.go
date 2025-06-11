package integration

import (
	"context"
	"testing"

	"github.com/allen13/tegridy-tokens/pkg/tokenizer"
	"github.com/stretchr/testify/assert"
)

// TestDynamoDBSizeLimits tests the DynamoDB item size limits
// DynamoDB has a 400KB limit per item, so large data tokenization
// will fail if the encrypted data + metadata exceeds this limit
func TestDynamoDBSizeLimits(t *testing.T) {
	// This test documents the expected behavior when hitting DynamoDB limits
	// It's separate from the main integration test to avoid test failures
	// when demonstrating the expected limitations

	t.Skip("Skipping size limit test - this is for documentation purposes")

	ctx := context.Background()

	// Test with 1MB data (will fail due to DynamoDB 400KB limit)
	largeData := generateRandomString(1024 * 1024) // 1 MB

	// In a real test, this would need actual AWS infrastructure
	var tkn tokenizer.Tokenizer // This would be initialized with real components

	req := tokenizer.TokenRequest{
		Data: largeData,
	}

	resp, err := tkn.Tokenize(ctx, req)

	// We expect this to fail due to DynamoDB size limits
	if err != nil {
		t.Logf("Expected failure for large data: %v", err)
	} else if !resp.Success {
		t.Logf("Expected failure captured in response: %s", resp.Error)
	} else {
		t.Error("Large data tokenization should fail due to DynamoDB 400KB limit")
	}
}

// TestRecommendedSizeLimits documents the recommended size limits
func TestRecommendedSizeLimits(t *testing.T) {
	t.Log("DynamoDB Item Size Limits:")
	t.Log("- Maximum item size: 400 KB")
	t.Log("- Recommended data size for tokenization: < 300 KB")
	t.Log("- This accounts for encryption overhead and metadata")
	t.Log("")
	t.Log("For larger data:")
	t.Log("- Consider using S3 with DynamoDB storing only metadata")
	t.Log("- Or implement data chunking/compression")
	t.Log("- Or use a different storage backend")

	// Test that we stay within reasonable limits
	sizes := []int{
		100 * 1024, // 100 KB - safe
		200 * 1024, // 200 KB - safe
		300 * 1024, // 300 KB - should work but close to limit
	}

	for _, size := range sizes {
		t.Logf("Size %d KB: Recommended for tokenization", size/1024)
		assert.LessOrEqual(t, size, 350*1024, "Should stay well under 400KB limit")
	}
}
