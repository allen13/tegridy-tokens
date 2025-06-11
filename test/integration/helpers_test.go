package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// SkipIfShort skips the test if -short flag is provided
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// RequireEnvVar ensures an environment variable is set
func RequireEnvVar(t *testing.T, name string) string {
	value := os.Getenv(name)
	require.NotEmpty(t, value, fmt.Sprintf("Environment variable %s must be set", name))
	return value
}

// GetEnvOrDefault returns environment variable value or default
func GetEnvOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

// TestConfig holds test configuration
type TestConfig struct {
	AWSRegion      string
	TableName      string
	KMSKeyID       string
	TestDataPrefix string
}

// GetTestConfig returns test configuration from environment or defaults
func GetTestConfig() TestConfig {
	return TestConfig{
		AWSRegion:      GetEnvOrDefault("AWS_REGION", "us-east-1"),
		TableName:      GetEnvOrDefault("DYNAMODB_TABLE_NAME", ""),
		KMSKeyID:       GetEnvOrDefault("KMS_KEY_ID", ""),
		TestDataPrefix: GetEnvOrDefault("TEST_DATA_PREFIX", "test-"),
	}
}

// GenerateTestData generates test data with consistent format
func GenerateTestData(prefix string, index int) string {
	return fmt.Sprintf("%s%d-%d", prefix, index, time.Now().UnixNano())
}

// AssertTokenFormat validates token format
func AssertTokenFormat(t *testing.T, token string) {
	require.NotEmpty(t, token, "Token should not be empty")
	require.NotContains(t, token, " ", "Token should not contain spaces")
	// Add more format validations as needed
}

// MeasureOperation measures the time taken for an operation
func MeasureOperation(name string, operation func()) time.Duration {
	start := time.Now()
	operation()
	duration := time.Since(start)
	return duration
}

// BatchTestData represents test data for batch operations
type BatchTestData struct {
	Index    int
	Original string
	Token    string
}

// GenerateBatchTestData creates test data for batch operations
func GenerateBatchTestData(size int, prefix string) []BatchTestData {
	data := make([]BatchTestData, size)
	for i := 0; i < size; i++ {
		data[i] = BatchTestData{
			Index:    i,
			Original: GenerateTestData(prefix, i),
		}
	}
	return data
}

