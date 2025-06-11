package tokenizer

import (
	"context"
	"time"
)

// Token represents a tokenized value with metadata
type Token struct {
	ID        string    `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	TTL       *int64    `json:"ttl,omitempty"`
}

// TokenRequest represents a single tokenization request
type TokenRequest struct {
	Data              string            `json:"data"`
	Format            string            `json:"format,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	TTLSeconds        *int64            `json:"ttl_seconds,omitempty"`
	PreserveFormat    bool              `json:"preserve_format,omitempty"`    // Enable format-preserving tokenization
	FormatHint        DataFormat        `json:"format_hint,omitempty"`        // Hint for format detection
}

// TokenResponse represents a tokenization response
type TokenResponse struct {
	Token           string     `json:"token"`
	TokenID         string     `json:"token_id"`
	Success         bool       `json:"success"`
	Error           string     `json:"error,omitempty"`
	FormatPreserved bool       `json:"format_preserved,omitempty"` // Indicates if format was preserved
	DetectedFormat  DataFormat `json:"detected_format,omitempty"`  // Format that was detected/used
}

// BatchTokenRequest represents a batch tokenization request
type BatchTokenRequest struct {
	Requests []TokenRequest `json:"requests"`
}

// BatchTokenResponse represents a batch tokenization response
type BatchTokenResponse struct {
	Responses []TokenResponse `json:"responses"`
	Success   int             `json:"success_count"`
	Failed    int             `json:"failed_count"`
}

// Tokenizer defines the interface for tokenization operations
type Tokenizer interface {
	// Tokenize creates a token for sensitive data
	Tokenize(ctx context.Context, req TokenRequest) (*TokenResponse, error)
	
	// Detokenize retrieves the original data from a token
	Detokenize(ctx context.Context, token string) (string, error)
	
	// TokenizeBatch processes multiple tokenization requests
	TokenizeBatch(ctx context.Context, req BatchTokenRequest) (*BatchTokenResponse, error)
	
	// DetokenizeBatch processes multiple detokenization requests
	DetokenizeBatch(ctx context.Context, tokens []string) (map[string]string, error)
}

// Repository defines the interface for token storage
type Repository interface {
	// Store saves an encrypted token
	Store(ctx context.Context, token *Token, encryptedData []byte) error
	
	// Retrieve gets encrypted data by token
	Retrieve(ctx context.Context, tokenValue string) ([]byte, error)
	
	// StoreBatch saves multiple tokens efficiently
	StoreBatch(ctx context.Context, items []TokenItem) error
	
	// RetrieveBatch gets multiple tokens efficiently
	RetrieveBatch(ctx context.Context, tokens []string) (map[string][]byte, error)
}

// TokenItem represents a token with its encrypted data for batch operations
type TokenItem struct {
	Token         *Token
	EncryptedData []byte
}

// Encryptor defines the interface for encryption operations
type Encryptor interface {
	// Encrypt encrypts data using KMS
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	
	// Decrypt decrypts data using KMS
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
	
	// GenerateDataKey generates a data key for envelope encryption
	GenerateDataKey(ctx context.Context) ([]byte, []byte, error)
}