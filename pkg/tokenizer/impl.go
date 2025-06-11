package tokenizer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// tokenizerImpl implements the Tokenizer interface
type tokenizerImpl struct {
	repository Repository
	encryptor  Encryptor
	tokenGen   TokenGenerator
	workers    int
	// Format-preserving tokenization support
	enableFPT    bool
	fptGenerator *FormatPreservingTokenGenerator
}

// TokenGenerator defines how tokens are generated
type TokenGenerator interface {
	Generate() string
}

// UUIDTokenGenerator generates UUID-based tokens
type UUIDTokenGenerator struct{}

func (g *UUIDTokenGenerator) Generate() string {
	return uuid.New().String()
}

// SecureTokenGenerator generates cryptographically secure tokens
type SecureTokenGenerator struct {
	Length int
}

func (g *SecureTokenGenerator) Generate() string {
	length := g.Length
	if length <= 0 {
		length = 32 // Default length
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if secure random fails
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// Config holds tokenizer configuration
type Config struct {
	Repository     Repository
	Encryptor      Encryptor
	TokenGenerator TokenGenerator
	Workers        int // Number of workers for batch processing
	// Format-preserving tokenization options
	EnableFPT bool               `json:"enable_fpt,omitempty"` // Enable format-preserving tokenization
	FPTConfig FPTGeneratorConfig `json:"fpt_config,omitempty"` // FPT configuration
}

// NewTokenizer creates a new tokenizer instance
func NewTokenizer(cfg Config) Tokenizer {
	if cfg.TokenGenerator == nil {
		cfg.TokenGenerator = &UUIDTokenGenerator{}
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 10 // Default workers
	}

	impl := &tokenizerImpl{
		repository: cfg.Repository,
		encryptor:  cfg.Encryptor,
		tokenGen:   cfg.TokenGenerator,
		workers:    cfg.Workers,
		enableFPT:  cfg.EnableFPT,
	}

	// Initialize FPT generator if enabled
	if cfg.EnableFPT {
		impl.fptGenerator = NewFormatPreservingTokenGenerator(cfg.FPTConfig)
	}

	return impl
}

// Tokenize creates a token for sensitive data
func (t *tokenizerImpl) Tokenize(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	// Check if format-preserving tokenization is requested and available
	if req.PreserveFormat && t.enableFPT && t.fptGenerator != nil {
		return t.tokenizeWithFormatPreservation(ctx, req)
	}

	// Standard tokenization
	return t.tokenizeStandard(ctx, req)
}

// tokenizeStandard performs standard (non-format-preserving) tokenization
func (t *tokenizerImpl) tokenizeStandard(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	// Generate token
	tokenValue := t.tokenGen.Generate()

	// Create token metadata
	token := &Token{
		ID:        uuid.New().String(),
		Value:     tokenValue,
		CreatedAt: time.Now().UTC(),
		TTL:       req.TTLSeconds,
	}

	// Encrypt the sensitive data
	encryptedData, err := t.encryptor.Encrypt(ctx, []byte(req.Data))
	if err != nil {
		return &TokenResponse{
			Success: false,
			Error:   fmt.Sprintf("encryption failed: %v", err),
		}, nil
	}

	// Store the token and encrypted data
	if err := t.repository.Store(ctx, token, encryptedData); err != nil {
		return &TokenResponse{
			Success: false,
			Error:   fmt.Sprintf("storage failed: %v", err),
		}, nil
	}

	return &TokenResponse{
		Token:           tokenValue,
		TokenID:         token.ID,
		Success:         true,
		FormatPreserved: false,
	}, nil
}

// tokenizeWithFormatPreservation performs format-preserving tokenization
func (t *tokenizerImpl) tokenizeWithFormatPreservation(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	// Determine format
	format := req.FormatHint
	if format == FormatNone || format == "" {
		// Auto-detect format
		detector := NewFormatDetector()
		format = detector.DetectFormat(req.Data)
	}

	// Try format-preserving tokenization
	if format != FormatNone {
		fptToken, err := t.fptGenerator.TokenizeWithFormat(req.Data, format)
		if err == nil {
			// Store FPT token with special metadata to indicate FPT
			token := &Token{
				ID:        uuid.New().String(),
				Value:     fptToken,
				CreatedAt: time.Now().UTC(),
				TTL:       req.TTLSeconds,
			}

			// For FPT, we store the original data encrypted with FPT metadata
			metadata := map[string]any{
				"fpt_enabled":   true,
				"fpt_format":    string(format),
				"original_data": req.Data,
			}

			metadataBytes, _ := json.Marshal(metadata)
			encryptedData, err := t.encryptor.Encrypt(ctx, metadataBytes)
			if err != nil {
				// Fallback to standard tokenization
				return t.tokenizeStandard(ctx, req)
			}

			if err := t.repository.Store(ctx, token, encryptedData); err != nil {
				// Fallback to standard tokenization
				return t.tokenizeStandard(ctx, req)
			}

			return &TokenResponse{
				Token:           fptToken,
				TokenID:         token.ID,
				Success:         true,
				FormatPreserved: true,
				DetectedFormat:  format,
			}, nil
		}
	}

	// Fallback to standard tokenization
	resp, err := t.tokenizeStandard(ctx, req)
	if err == nil && resp.Success {
		resp.DetectedFormat = format
	}
	return resp, err
}

// Detokenize retrieves the original data from a token
func (t *tokenizerImpl) Detokenize(ctx context.Context, token string) (string, error) {
	// Retrieve encrypted data
	encryptedData, err := t.repository.Retrieve(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Decrypt the data
	plaintext, err := t.encryptor.Decrypt(ctx, encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Check if this is FPT data by trying to parse as metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal(plaintext, &metadata); err == nil {
		if fptEnabled, exists := metadata["fpt_enabled"].(bool); exists && fptEnabled {
			// This is FPT data - extract original data
			if originalData, exists := metadata["original_data"].(string); exists {
				return originalData, nil
			}
		}
	}

	// Standard tokenization - return the decrypted data directly
	return string(plaintext), nil
}

// TokenizeBatch processes multiple tokenization requests
func (t *tokenizerImpl) TokenizeBatch(ctx context.Context, req BatchTokenRequest) (*BatchTokenResponse, error) {
	resp := &BatchTokenResponse{
		Responses: make([]TokenResponse, len(req.Requests)),
	}

	// Create channels for work distribution
	jobs := make(chan jobItem, len(req.Requests))
	results := make(chan jobResult, len(req.Requests))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < t.workers; i++ {
		wg.Add(1)
		go t.worker(ctx, jobs, results, &wg)
	}

	// Send jobs
	for i, request := range req.Requests {
		jobs <- jobItem{index: i, request: request}
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	tokenItems := make([]TokenItem, 0, len(req.Requests))
	for result := range results {
		resp.Responses[result.index] = result.response
		if result.response.Success {
			resp.Success++
			tokenItems = append(tokenItems, result.tokenItem)
		} else {
			resp.Failed++
		}
	}

	// Batch store successful tokens
	if len(tokenItems) > 0 {
		if err := t.repository.StoreBatch(ctx, tokenItems); err != nil {
			// Update responses for failed batch store
			for i := range resp.Responses {
				if resp.Responses[i].Success {
					resp.Responses[i].Success = false
					resp.Responses[i].Error = fmt.Sprintf("batch storage failed: %v", err)
					resp.Success--
					resp.Failed++
				}
			}
		}
	}

	return resp, nil
}

// DetokenizeBatch processes multiple detokenization requests
func (t *tokenizerImpl) DetokenizeBatch(ctx context.Context, tokens []string) (map[string]string, error) {
	// Retrieve all encrypted data
	encryptedDataMap, err := t.repository.RetrieveBatch(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("batch retrieve failed: %w", err)
	}

	results := make(map[string]string, len(tokens))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process decryption in parallel
	sem := make(chan struct{}, t.workers)
	for token, encryptedData := range encryptedDataMap {
		wg.Add(1)
		sem <- struct{}{}

		go func(token string, data []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			plaintext, err := t.encryptor.Decrypt(ctx, data)
			if err != nil {
				return // Skip failed decryptions
			}

			mu.Lock()
			results[token] = string(plaintext)
			mu.Unlock()
		}(token, encryptedData)
	}

	wg.Wait()
	return results, nil
}

// Worker types for batch processing
type jobItem struct {
	index   int
	request TokenRequest
}

type jobResult struct {
	index     int
	response  TokenResponse
	tokenItem TokenItem
}

// worker processes tokenization jobs
func (t *tokenizerImpl) worker(ctx context.Context, jobs <-chan jobItem, results chan<- jobResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		// Generate token
		tokenValue := t.tokenGen.Generate()
		token := &Token{
			ID:        uuid.New().String(),
			Value:     tokenValue,
			CreatedAt: time.Now().UTC(),
			TTL:       job.request.TTLSeconds,
		}

		// Encrypt data
		encryptedData, err := t.encryptor.Encrypt(ctx, []byte(job.request.Data))
		if err != nil {
			results <- jobResult{
				index: job.index,
				response: TokenResponse{
					Success: false,
					Error:   fmt.Sprintf("encryption failed: %v", err),
				},
			}
			continue
		}

		// Send successful result
		results <- jobResult{
			index: job.index,
			response: TokenResponse{
				Token:   tokenValue,
				TokenID: token.ID,
				Success: true,
			},
			tokenItem: TokenItem{
				Token:         token,
				EncryptedData: encryptedData,
			},
		}
	}
}
