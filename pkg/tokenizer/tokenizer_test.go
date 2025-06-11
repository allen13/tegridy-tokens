package tokenizer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Store(ctx context.Context, token *Token, encryptedData []byte) error {
	args := m.Called(ctx, token, encryptedData)
	return args.Error(0)
}

func (m *MockRepository) Retrieve(ctx context.Context, tokenValue string) ([]byte, error) {
	args := m.Called(ctx, tokenValue)
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) StoreBatch(ctx context.Context, items []TokenItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockRepository) RetrieveBatch(ctx context.Context, tokens []string) (map[string][]byte, error) {
	args := m.Called(ctx, tokens)
	if args.Get(0) != nil {
		return args.Get(0).(map[string][]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockEncryptor is a mock implementation of Encryptor
type MockEncryptor struct {
	mock.Mock
}

func (m *MockEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	args := m.Called(ctx, plaintext)
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	args := m.Called(ctx, ciphertext)
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockEncryptor) GenerateDataKey(ctx context.Context) ([]byte, []byte, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil && args.Get(1) != nil {
		return args.Get(0).([]byte), args.Get(1).([]byte), args.Error(2)
	}
	return nil, nil, args.Error(2)
}

func TestTokenizer_Tokenize(t *testing.T) {
	ctx := context.Background()
	
	t.Run("SuccessfulTokenization", func(t *testing.T) {
		// Setup mocks
		mockRepo := new(MockRepository)
		mockEnc := new(MockEncryptor)
		
		// Create tokenizer
		tkn := NewTokenizer(Config{
			Repository:     mockRepo,
			Encryptor:      mockEnc,
			TokenGenerator: &UUIDTokenGenerator{},
		})
		
		// Setup expectations
		mockEnc.On("Encrypt", ctx, []byte("test-data")).Return([]byte("encrypted"), nil)
		mockRepo.On("Store", ctx, mock.AnythingOfType("*tokenizer.Token"), []byte("encrypted")).Return(nil)
		
		// Execute
		req := TokenRequest{
			Data: "test-data",
		}
		resp, err := tkn.Tokenize(ctx, req)
		
		// Assert
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Token)
		assert.NotEmpty(t, resp.TokenID)
		
		// Verify mocks
		mockEnc.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
	
	t.Run("EncryptionFailure", func(t *testing.T) {
		// Setup mocks
		mockRepo := new(MockRepository)
		mockEnc := new(MockEncryptor)
		
		// Create tokenizer
		tkn := NewTokenizer(Config{
			Repository: mockRepo,
			Encryptor:  mockEnc,
		})
		
		// Setup expectations
		mockEnc.On("Encrypt", ctx, []byte("test-data")).Return(nil, errors.New("encryption error"))
		
		// Execute
		req := TokenRequest{
			Data: "test-data",
		}
		resp, err := tkn.Tokenize(ctx, req)
		
		// Assert
		assert.NoError(t, err) // Error is captured in response
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "encryption failed")
		
		// Verify mocks
		mockEnc.AssertExpectations(t)
	})
}

func TestTokenizer_Detokenize(t *testing.T) {
	ctx := context.Background()
	
	t.Run("SuccessfulDetokenization", func(t *testing.T) {
		// Setup mocks
		mockRepo := new(MockRepository)
		mockEnc := new(MockEncryptor)
		
		// Create tokenizer
		tkn := NewTokenizer(Config{
			Repository: mockRepo,
			Encryptor:  mockEnc,
		})
		
		// Setup expectations
		mockRepo.On("Retrieve", ctx, "test-token").Return([]byte("encrypted"), nil)
		mockEnc.On("Decrypt", ctx, []byte("encrypted")).Return([]byte("test-data"), nil)
		
		// Execute
		data, err := tkn.Detokenize(ctx, "test-token")
		
		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "test-data", data)
		
		// Verify mocks
		mockRepo.AssertExpectations(t)
		mockEnc.AssertExpectations(t)
	})
	
	t.Run("TokenNotFound", func(t *testing.T) {
		// Setup mocks
		mockRepo := new(MockRepository)
		mockEnc := new(MockEncryptor)
		
		// Create tokenizer
		tkn := NewTokenizer(Config{
			Repository: mockRepo,
			Encryptor:  mockEnc,
		})
		
		// Setup expectations
		mockRepo.On("Retrieve", ctx, "invalid-token").Return(nil, errors.New("token not found"))
		
		// Execute
		_, err := tkn.Detokenize(ctx, "invalid-token")
		
		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve token")
		
		// Verify mocks
		mockRepo.AssertExpectations(t)
	})
}

func TestTokenGenerators(t *testing.T) {
	t.Run("UUIDTokenGenerator", func(t *testing.T) {
		gen := &UUIDTokenGenerator{}
		
		// Generate multiple tokens
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token := gen.Generate()
			assert.NotEmpty(t, token)
			assert.Len(t, token, 36) // UUID v4 format
			
			// Check uniqueness
			assert.False(t, tokens[token], "Token should be unique")
			tokens[token] = true
		}
	})
	
	t.Run("SecureTokenGenerator", func(t *testing.T) {
		gen := &SecureTokenGenerator{Length: 32}
		
		// Generate multiple tokens
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token := gen.Generate()
			assert.NotEmpty(t, token)
			
			// Check uniqueness
			assert.False(t, tokens[token], "Token should be unique")
			tokens[token] = true
		}
	})
}

func TestBatchOperations(t *testing.T) {
	ctx := context.Background()
	
	t.Run("BatchTokenization", func(t *testing.T) {
		// Setup mocks
		mockRepo := new(MockRepository)
		mockEnc := new(MockEncryptor)
		
		// Create tokenizer with limited workers for predictable testing
		tkn := NewTokenizer(Config{
			Repository: mockRepo,
			Encryptor:  mockEnc,
			Workers:    2,
		})
		
		// Setup expectations
		mockEnc.On("Encrypt", ctx, mock.Anything).Return([]byte("encrypted"), nil)
		mockRepo.On("StoreBatch", ctx, mock.AnythingOfType("[]tokenizer.TokenItem")).Return(nil)
		
		// Create batch request
		requests := []TokenRequest{
			{Data: "data1"},
			{Data: "data2"},
			{Data: "data3"},
		}
		
		// Execute
		resp, err := tkn.TokenizeBatch(ctx, BatchTokenRequest{Requests: requests})
		
		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 3, resp.Success)
		assert.Equal(t, 0, resp.Failed)
		assert.Len(t, resp.Responses, 3)
		
		// Verify all responses are successful
		for _, r := range resp.Responses {
			assert.True(t, r.Success)
			assert.NotEmpty(t, r.Token)
		}
		
		// Verify mocks
		mockEnc.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestTokenWithTTL(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockRepo := new(MockRepository)
	mockEnc := new(MockEncryptor)
	
	// Create tokenizer
	tkn := NewTokenizer(Config{
		Repository: mockRepo,
		Encryptor:  mockEnc,
	})
	
	// Setup expectations
	ttl := int64(3600)
	mockEnc.On("Encrypt", ctx, []byte("ttl-data")).Return([]byte("encrypted"), nil)
	mockRepo.On("Store", ctx, mock.MatchedBy(func(token *Token) bool {
		return token.TTL != nil && *token.TTL == ttl
	}), []byte("encrypted")).Return(nil)
	
	// Execute
	req := TokenRequest{
		Data:       "ttl-data",
		TTLSeconds: &ttl,
	}
	resp, err := tkn.Tokenize(ctx, req)
	
	// Assert
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	
	// Verify mocks
	mockEnc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func BenchmarkTokenization(b *testing.B) {
	ctx := context.Background()
	
	// Setup mocks
	mockRepo := new(MockRepository)
	mockEnc := new(MockEncryptor)
	
	// Create tokenizer
	tkn := NewTokenizer(Config{
		Repository: mockRepo,
		Encryptor:  mockEnc,
	})
	
	// Setup expectations
	mockEnc.On("Encrypt", ctx, mock.Anything).Return([]byte("encrypted"), nil)
	mockRepo.On("Store", ctx, mock.Anything, mock.Anything).Return(nil)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := TokenRequest{
			Data: "benchmark-data",
		}
		_, _ = tkn.Tokenize(ctx, req)
	}
}