package tokenizer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// FormatPreservingTokenGenerator generates tokens that preserve the format of the original data
type FormatPreservingTokenGenerator struct {
	fpt           FormatPreservingTokenizer
	detector      FormatDetector
	normalizer    *FormatNormalizer
	validator     *FormatValidator
	autoDetect    bool
	defaultFormat DataFormat
}

// FPTGeneratorConfig holds configuration for format-preserving token generator
type FPTGeneratorConfig struct {
	Key           []byte                       // Encryption key for FPE (optional)
	AutoDetect    bool                         // Automatically detect format
	DefaultFormat DataFormat                   // Default format when auto-detection fails
	CustomFormats map[DataFormat]FormatPattern // Custom format definitions
}

// NewFormatPreservingTokenGenerator creates a new format-preserving token generator
func NewFormatPreservingTokenGenerator(config FPTGeneratorConfig) *FormatPreservingTokenGenerator {
	// Generate a key if none provided
	key := config.Key
	if len(key) == 0 {
		key = make([]byte, 32) // 256-bit key
		rand.Read(key)
	}

	// Create FPT tokenizer
	fptConfig := FPTConfig{
		Key:           key,
		CustomFormats: config.CustomFormats,
	}
	fpt := NewFormatPreservingTokenizer(fptConfig)

	return &FormatPreservingTokenGenerator{
		fpt:           fpt,
		detector:      NewFormatDetector(),
		normalizer:    NewFormatNormalizer(),
		validator:     NewFormatValidator(),
		autoDetect:    config.AutoDetect,
		defaultFormat: config.DefaultFormat,
	}
}

// Generate creates a format-preserving token
func (fptg *FormatPreservingTokenGenerator) Generate() string {
	// This method is for compatibility with the existing TokenGenerator interface
	// It generates a random format-preserving token for demonstration
	return fptg.GenerateForData("1234-5678-9012-3456", FormatCreditCard)
}

// GenerateForData creates a format-preserving token for specific data
func (fptg *FormatPreservingTokenGenerator) GenerateForData(data string, format DataFormat) string {
	token, err := fptg.TokenizeWithFormat(data, format)
	if err != nil {
		// Fallback to UUID if FPT fails
		return (&UUIDTokenGenerator{}).Generate()
	}
	return token
}

// TokenizeWithFormat tokenizes data while preserving its format
func (fptg *FormatPreservingTokenGenerator) TokenizeWithFormat(data string, format DataFormat) (string, error) {
	// Normalize the data first
	normalizedData := fptg.normalizeData(data, format)

	// Auto-detect format if requested and no format specified
	if format == FormatNone && fptg.autoDetect {
		format = fptg.detector.DetectFormat(normalizedData)
	}

	// Use default format if still no format
	if format == FormatNone {
		format = fptg.defaultFormat
	}

	// Validate the data against the format
	if !fptg.detector.IsValidFormat(normalizedData, format) {
		return "", fmt.Errorf("data does not match format %s", format)
	}

	// Perform additional validation for specific formats
	if err := fptg.validateData(normalizedData, format); err != nil {
		return "", err
	}

	// Generate format-preserving token
	return fptg.fpt.TokenizePreservingFormat(normalizedData, format)
}

// DetokenizeWithFormat retrieves original data from a format-preserving token
func (fptg *FormatPreservingTokenGenerator) DetokenizeWithFormat(token string, format DataFormat) (string, error) {
	return fptg.fpt.DetokenizePreservingFormat(token, format)
}

// normalizeData normalizes data according to its format
func (fptg *FormatPreservingTokenGenerator) normalizeData(data string, format DataFormat) string {
	switch format {
	case FormatCreditCard:
		return fptg.normalizer.NormalizeCreditCard(data)
	case FormatSSN:
		return fptg.normalizer.NormalizeSSN(data)
	case FormatPhone:
		return fptg.normalizer.NormalizePhone(data)
	default:
		return data
	}
}

// validateData performs additional validation for specific formats
func (fptg *FormatPreservingTokenGenerator) validateData(data string, format DataFormat) error {
	switch format {
	case FormatCreditCard:
		if !fptg.validator.ValidateCreditCard(data) {
			return fmt.Errorf("invalid credit card number")
		}
	case FormatSSN:
		if !fptg.validator.ValidateSSN(data) {
			return fmt.Errorf("invalid SSN")
		}
	case FormatPhone:
		if !fptg.validator.ValidatePhone(data) {
			return fmt.Errorf("invalid phone number")
		}
	}
	return nil
}

// DeterministicFormatPreservingTokenGenerator creates tokens deterministically
type DeterministicFormatPreservingTokenGenerator struct {
	*FormatPreservingTokenGenerator
	seed []byte
}

// NewDeterministicFormatPreservingTokenGenerator creates a deterministic FPT generator
func NewDeterministicFormatPreservingTokenGenerator(seed []byte, config FPTGeneratorConfig) *DeterministicFormatPreservingTokenGenerator {
	// Generate deterministic key from seed
	hash := sha256.Sum256(seed)
	config.Key = hash[:]

	fptg := NewFormatPreservingTokenGenerator(config)

	return &DeterministicFormatPreservingTokenGenerator{
		FormatPreservingTokenGenerator: fptg,
		seed:                           seed,
	}
}

// Generate creates a deterministic format-preserving token
func (dfptg *DeterministicFormatPreservingTokenGenerator) Generate() string {
	// Generate deterministic token based on seed
	return dfptg.GenerateForData("1234-5678-9012-3456", FormatCreditCard)
}

// FormatPreservingBatchTokenizer handles batch FPT operations
type FormatPreservingBatchTokenizer struct {
	generator *FormatPreservingTokenGenerator
}

// NewFormatPreservingBatchTokenizer creates a new batch FPT tokenizer
func NewFormatPreservingBatchTokenizer(config FPTGeneratorConfig) *FormatPreservingBatchTokenizer {
	return &FormatPreservingBatchTokenizer{
		generator: NewFormatPreservingTokenGenerator(config),
	}
}

// TokenizeBatch tokenizes multiple items while preserving their formats
func (fpbt *FormatPreservingBatchTokenizer) TokenizeBatch(requests []FPTTokenRequest) []FPTTokenResponse {
	responses := make([]FPTTokenResponse, len(requests))

	for i, req := range requests {
		token, err := fpbt.generator.TokenizeWithFormat(req.Data, req.Format)

		response := FPTTokenResponse{
			Token:           token,
			OriginalData:    req.Data,
			DetectedFormat:  fpbt.generator.detector.DetectFormat(req.Data),
			RequestedFormat: req.Format,
			Success:         err == nil,
		}

		if err != nil {
			response.Error = err.Error()
		}

		responses[i] = response
	}

	return responses
}

// DetokenizeBatch detokenizes multiple tokens
func (fpbt *FormatPreservingBatchTokenizer) DetokenizeBatch(requests []FPTDetokenRequest) []FPTDetokenResponse {
	responses := make([]FPTDetokenResponse, len(requests))

	for i, req := range requests {
		data, err := fpbt.generator.DetokenizeWithFormat(req.Token, req.Format)

		response := FPTDetokenResponse{
			OriginalData: data,
			Token:        req.Token,
			Format:       req.Format,
			Success:      err == nil,
		}

		if err != nil {
			response.Error = err.Error()
		}

		responses[i] = response
	}

	return responses
}

// FPTTokenRequest represents a format-preserving tokenization request
type FPTTokenRequest struct {
	Data   string     `json:"data"`
	Format DataFormat `json:"format"`
}

// FPTTokenResponse represents a format-preserving tokenization response
type FPTTokenResponse struct {
	Token           string     `json:"token"`
	OriginalData    string     `json:"original_data,omitempty"`
	DetectedFormat  DataFormat `json:"detected_format"`
	RequestedFormat DataFormat `json:"requested_format"`
	Success         bool       `json:"success"`
	Error           string     `json:"error,omitempty"`
}

// FPTDetokenRequest represents a format-preserving detokenization request
type FPTDetokenRequest struct {
	Token  string     `json:"token"`
	Format DataFormat `json:"format"`
}

// FPTDetokenResponse represents a format-preserving detokenization response
type FPTDetokenResponse struct {
	OriginalData string     `json:"original_data"`
	Token        string     `json:"token,omitempty"`
	Format       DataFormat `json:"format"`
	Success      bool       `json:"success"`
	Error        string     `json:"error,omitempty"`
}

// FormatPreservingTokenizerWrapper wraps the existing tokenizer to support FPT
type FormatPreservingTokenizerWrapper struct {
	standardTokenizer Tokenizer
	fptGenerator      *FormatPreservingTokenGenerator
	enableFPT         bool
}

// NewFormatPreservingTokenizerWrapper creates a wrapper that supports both standard and FPT
func NewFormatPreservingTokenizerWrapper(standardTokenizer Tokenizer, fptConfig FPTGeneratorConfig) *FormatPreservingTokenizerWrapper {
	return &FormatPreservingTokenizerWrapper{
		standardTokenizer: standardTokenizer,
		fptGenerator:      NewFormatPreservingTokenGenerator(fptConfig),
		enableFPT:         true,
	}
}

// TokenizeWithFormatPreservation tokenizes data with optional format preservation
func (fptw *FormatPreservingTokenizerWrapper) TokenizeWithFormatPreservation(ctx context.Context, req TokenRequest, preserveFormat bool) (*TokenResponse, error) {
	if !preserveFormat || !fptw.enableFPT {
		return fptw.standardTokenizer.Tokenize(ctx, req)
	}

	// Detect format if not specified
	format := DataFormat(req.Format)
	if format == "" || format == FormatNone {
		format = fptw.fptGenerator.detector.DetectFormat(req.Data)
	}

	// Use FPT if format detected, otherwise fall back to standard
	if format != FormatNone {
		token, err := fptw.fptGenerator.TokenizeWithFormat(req.Data, format)
		if err == nil {
			return &TokenResponse{
				Token:   token,
				Success: true,
			}, nil
		}
	}

	// Fallback to standard tokenization
	return fptw.standardTokenizer.Tokenize(ctx, req)
}
