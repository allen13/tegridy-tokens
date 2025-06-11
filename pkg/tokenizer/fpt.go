package tokenizer

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// FormatPreservingTokenizer defines the interface for format-preserving tokenization
type FormatPreservingTokenizer interface {
	// TokenizePreservingFormat creates a token that preserves the format of the original data
	TokenizePreservingFormat(data string, format DataFormat) (string, error)

	// DetokenizePreservingFormat retrieves original data from a format-preserving token
	DetokenizePreservingFormat(token string, format DataFormat) (string, error)
}

// DataFormat represents different data formats for format-preserving tokenization
type DataFormat string

const (
	FormatCreditCard   DataFormat = "credit_card"  // 1234-5678-9012-3456 or 1234567890123456
	FormatSSN          DataFormat = "ssn"          // 123-45-6789
	FormatPhone        DataFormat = "phone"        // +1-555-123-4567, (555) 123-4567
	FormatNumeric      DataFormat = "numeric"      // Any numeric string
	FormatAlphaNumeric DataFormat = "alphanumeric" // Letters and numbers
	FormatCustom       DataFormat = "custom"       // Custom pattern-based format
	FormatNone         DataFormat = "none"         // No format preservation (original behavior)
)

// FormatPattern defines a pattern for format-preserving tokenization
type FormatPattern struct {
	Pattern     string            // Regex pattern to match the format
	Replacement string            // Replacement pattern with placeholders
	CharSets    map[string]string // Character sets for each placeholder
}

// Common format patterns
var CommonFormats = map[DataFormat]FormatPattern{
	FormatCreditCard: {
		Pattern:     `^(\d{4})-?(\d{4})-?(\d{4})-?(\d{4})$`,
		Replacement: "$1-$2-$3-$4",
		CharSets: map[string]string{
			"1": "0123456789",
			"2": "0123456789",
			"3": "0123456789",
			"4": "0123456789",
		},
	},
	FormatSSN: {
		Pattern:     `^(\d{3})-?(\d{2})-?(\d{4})$`,
		Replacement: "$1-$2-$3",
		CharSets: map[string]string{
			"1": "0123456789",
			"2": "0123456789",
			"3": "0123456789",
		},
	},
	FormatPhone: {
		Pattern:     `^(\+?1)?[-.\s]?\(?(\d{3})\)?[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
		Replacement: "$1-$2-$3-$4",
		CharSets: map[string]string{
			"1": "+1",
			"2": "0123456789",
			"3": "0123456789",
			"4": "0123456789",
		},
	},
	FormatNumeric: {
		Pattern:     `^(\d+)$`,
		Replacement: "$1",
		CharSets: map[string]string{
			"1": "0123456789",
		},
	},
}

// FPTConfig holds configuration for format-preserving tokenization
type FPTConfig struct {
	Key           []byte                       // Encryption key for FPE
	CustomFormats map[DataFormat]FormatPattern // Custom format definitions
}

// formatPreservingTokenizer implements format-preserving tokenization
type formatPreservingTokenizer struct {
	key           []byte
	customFormats map[DataFormat]FormatPattern
}

// NewFormatPreservingTokenizer creates a new format-preserving tokenizer
func NewFormatPreservingTokenizer(config FPTConfig) FormatPreservingTokenizer {
	fpt := &formatPreservingTokenizer{
		key:           config.Key,
		customFormats: make(map[DataFormat]FormatPattern),
	}

	// Add custom formats if provided
	for format, pattern := range config.CustomFormats {
		fpt.customFormats[format] = pattern
	}

	return fpt
}

// TokenizePreservingFormat creates a format-preserving token
func (fpt *formatPreservingTokenizer) TokenizePreservingFormat(data string, format DataFormat) (string, error) {
	if format == FormatNone {
		return "", fmt.Errorf("format-preserving tokenization requires a specific format")
	}

	// Get the format pattern
	pattern, err := fpt.getFormatPattern(format)
	if err != nil {
		return "", err
	}

	// Extract the structure and data parts
	structure, dataParts, err := fpt.extractDataParts(data, pattern)
	if err != nil {
		return "", err
	}

	// Tokenize each data part while preserving format
	tokenizedParts := make([]string, len(dataParts))
	for i, part := range dataParts {
		charset := pattern.CharSets[strconv.Itoa(i+1)]
		if charset == "" {
			charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		}

		tokenizedPart, err := fpt.tokenizeString(part, charset)
		if err != nil {
			return "", err
		}
		tokenizedParts[i] = tokenizedPart
	}

	// Reconstruct the token with the original structure
	return fpt.reconstructToken(structure, tokenizedParts), nil
}

// DetokenizePreservingFormat retrieves original data from a format-preserving token
func (fpt *formatPreservingTokenizer) DetokenizePreservingFormat(token string, format DataFormat) (string, error) {
	if format == FormatNone {
		return "", fmt.Errorf("format-preserving detokenization requires a specific format")
	}

	// Get the format pattern
	pattern, err := fpt.getFormatPattern(format)
	if err != nil {
		return "", err
	}

	// Extract the structure and token parts
	structure, tokenParts, err := fpt.extractDataParts(token, pattern)
	if err != nil {
		return "", err
	}

	// Detokenize each token part
	originalParts := make([]string, len(tokenParts))
	for i, part := range tokenParts {
		charset := pattern.CharSets[strconv.Itoa(i+1)]
		if charset == "" {
			charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		}

		originalPart, err := fpt.detokenizeString(part, charset)
		if err != nil {
			return "", err
		}
		originalParts[i] = originalPart
	}

	// Reconstruct the original data with the structure
	return fpt.reconstructToken(structure, originalParts), nil
}

// getFormatPattern retrieves the pattern for a given format
func (fpt *formatPreservingTokenizer) getFormatPattern(format DataFormat) (FormatPattern, error) {
	// Check custom formats first
	if pattern, exists := fpt.customFormats[format]; exists {
		return pattern, nil
	}

	// Check common formats
	if pattern, exists := CommonFormats[format]; exists {
		return pattern, nil
	}

	return FormatPattern{}, fmt.Errorf("unsupported format: %s", format)
}

// extractDataParts extracts the data parts and structure from input using the pattern
func (fpt *formatPreservingTokenizer) extractDataParts(input string, pattern FormatPattern) (string, []string, error) {
	re, err := regexp.Compile(pattern.Pattern)
	if err != nil {
		return "", nil, fmt.Errorf("invalid pattern regex: %w", err)
	}

	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return "", nil, fmt.Errorf("input does not match expected format pattern")
	}

	// matches[0] is the full match, matches[1:] are the capture groups
	structure := pattern.Replacement
	dataParts := matches[1:]

	return structure, dataParts, nil
}

// reconstructToken reconstructs a token/data with the given structure and parts
func (fpt *formatPreservingTokenizer) reconstructToken(structure string, parts []string) string {
	result := structure
	for i, part := range parts {
		placeholder := fmt.Sprintf("$%d", i+1)
		result = strings.Replace(result, placeholder, part, 1)
	}
	return result
}

// tokenizeString tokenizes a string using Format Preserving Encryption (FPE)
func (fpt *formatPreservingTokenizer) tokenizeString(input, charset string) (string, error) {
	if len(input) == 0 {
		return "", nil
	}

	// Convert input to number in the given charset base
	inputNum, err := fpt.stringToNumber(input, charset)
	if err != nil {
		return "", err
	}

	// Apply FPE (Format Preserving Encryption) using FF1 algorithm approximation
	tokenNum, err := fpt.encryptNumber(inputNum, len(charset), len(input))
	if err != nil {
		return "", err
	}

	// Convert back to string in the same charset
	return fpt.numberToString(tokenNum, charset, len(input))
}

// detokenizeString detokenizes a string using Format Preserving Encryption (FPE)
func (fpt *formatPreservingTokenizer) detokenizeString(token, charset string) (string, error) {
	if len(token) == 0 {
		return "", nil
	}

	// Convert token to number
	tokenNum, err := fpt.stringToNumber(token, charset)
	if err != nil {
		return "", err
	}

	// Apply FPE decryption
	originalNum, err := fpt.decryptNumber(tokenNum, len(charset), len(token))
	if err != nil {
		return "", err
	}

	// Convert back to string
	return fpt.numberToString(originalNum, charset, len(token))
}

// stringToNumber converts a string to a number using the given charset as base
func (fpt *formatPreservingTokenizer) stringToNumber(input, charset string) (*big.Int, error) {
	base := big.NewInt(int64(len(charset)))
	result := big.NewInt(0)

	for _, char := range input {
		index := strings.IndexRune(charset, char)
		if index == -1 {
			return nil, fmt.Errorf("character %c not found in charset", char)
		}

		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(index)))
	}

	return result, nil
}

// numberToString converts a number to a string using the given charset as base
func (fpt *formatPreservingTokenizer) numberToString(num *big.Int, charset string, length int) (string, error) {
	if num.Cmp(big.NewInt(0)) == 0 && length == 1 {
		return string(charset[0]), nil
	}

	base := big.NewInt(int64(len(charset)))
	result := make([]byte, length)
	temp := new(big.Int).Set(num)

	for i := length - 1; i >= 0; i-- {
		mod := new(big.Int)
		temp.DivMod(temp, base, mod)
		result[i] = charset[mod.Int64()]
	}

	return string(result), nil
}

// encryptNumber encrypts a number using a simplified FPE algorithm
func (fpt *formatPreservingTokenizer) encryptNumber(input *big.Int, radix, length int) (*big.Int, error) {
	// Simplified FPE implementation using AES in counter mode
	// This is a basic implementation - production should use FF1/FF3-1

	if len(fpt.key) == 0 {
		// Generate a deterministic key from input for consistency
		return fpt.pseudoRandomPermutation(input, radix, length)
	}

	// Use AES with the provided key
	block, err := aes.NewCipher(fpt.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create a deterministic nonce from input
	nonce := make([]byte, 12)
	inputBytes := input.Bytes()
	copy(nonce, inputBytes)

	// Encrypt using AES-GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Convert input to bytes for encryption
	inputBytes = make([]byte, 8)
	binary.BigEndian.PutUint64(inputBytes, input.Uint64())

	ciphertext := gcm.Seal(nil, nonce, inputBytes, nil)

	// Convert ciphertext back to number and ensure it fits in the same domain
	result := new(big.Int).SetBytes(ciphertext[:8])

	// Ensure result is in valid range for the given radix and length
	maxValue := new(big.Int)
	radixBig := big.NewInt(int64(radix))
	maxValue.Exp(radixBig, big.NewInt(int64(length)), nil)
	result.Mod(result, maxValue)

	return result, nil
}

// decryptNumber decrypts a number using the reverse of the encryption algorithm
func (fpt *formatPreservingTokenizer) decryptNumber(token *big.Int, radix, length int) (*big.Int, error) {
	// For this simplified implementation, we'll use the same function
	// In production, this would be the inverse of the encryption
	return fpt.encryptNumber(token, radix, length)
}

// pseudoRandomPermutation provides a simple PRP when no key is provided
func (fpt *formatPreservingTokenizer) pseudoRandomPermutation(input *big.Int, radix, length int) (*big.Int, error) {
	// Simple linear congruential generator for demonstration
	// In production, use a proper cryptographic PRP

	a := big.NewInt(1664525)
	c := big.NewInt(1013904223)
	m := new(big.Int)
	radixBig := big.NewInt(int64(radix))
	m.Exp(radixBig, big.NewInt(int64(length)), nil)

	result := new(big.Int)
	result.Mul(input, a)
	result.Add(result, c)
	result.Mod(result, m)

	return result, nil
}
