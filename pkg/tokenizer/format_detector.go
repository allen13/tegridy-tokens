package tokenizer

import (
	"regexp"
	"strings"
)

// FormatDetector provides automatic format detection for data
type FormatDetector interface {
	// DetectFormat automatically detects the format of the given data
	DetectFormat(data string) DataFormat
	
	// IsValidFormat checks if data matches the specified format
	IsValidFormat(data string, format DataFormat) bool
}

// formatDetector implements format detection logic
type formatDetector struct {
	patterns map[DataFormat]*regexp.Regexp
}

// NewFormatDetector creates a new format detector
func NewFormatDetector() FormatDetector {
	detector := &formatDetector{
		patterns: make(map[DataFormat]*regexp.Regexp),
	}
	
	// Compile regex patterns for format detection
	detector.patterns[FormatCreditCard] = regexp.MustCompile(`^(\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4})$`)
	detector.patterns[FormatSSN] = regexp.MustCompile(`^\d{3}[-\s]?\d{2}[-\s]?\d{4}$`)
	detector.patterns[FormatPhone] = regexp.MustCompile(`^(\+?1[-.\s]?)?\(?(\d{3})\)?[-.\s]?(\d{3})[-.\s]?(\d{4})$`)
	detector.patterns[FormatNumeric] = regexp.MustCompile(`^\d+$`)
	detector.patterns[FormatAlphaNumeric] = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	
	return detector
}

// DetectFormat automatically detects the format of the given data
func (fd *formatDetector) DetectFormat(data string) DataFormat {
	data = strings.TrimSpace(data)
	
	// Check specific formats in order of specificity
	formatOrder := []DataFormat{
		FormatCreditCard,
		FormatSSN,
		FormatPhone,
		FormatNumeric,
		FormatAlphaNumeric,
	}
	
	for _, format := range formatOrder {
		if fd.IsValidFormat(data, format) {
			return format
		}
	}
	
	// Default to no format preservation
	return FormatNone
}

// IsValidFormat checks if data matches the specified format
func (fd *formatDetector) IsValidFormat(data string, format DataFormat) bool {
	data = strings.TrimSpace(data)
	
	pattern, exists := fd.patterns[format]
	if !exists {
		return false
	}
	
	return pattern.MatchString(data)
}

// FormatValidator provides additional validation for specific formats
type FormatValidator struct{}

// NewFormatValidator creates a new format validator
func NewFormatValidator() *FormatValidator {
	return &FormatValidator{}
}

// ValidateCreditCard performs Luhn algorithm validation for credit cards
func (fv *FormatValidator) ValidateCreditCard(cardNumber string) bool {
	// Remove non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")
	
	// Must be 13-19 digits
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	
	// Luhn algorithm
	sum := 0
	alternate := false
	
	for i := len(digits) - 1; i >= 0; i-- {
		digit := int(digits[i] - '0')
		
		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}
		
		sum += digit
		alternate = !alternate
	}
	
	return sum%10 == 0
}

// ValidateSSN performs basic SSN validation
func (fv *FormatValidator) ValidateSSN(ssn string) bool {
	// Remove non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(ssn, "")
	
	// Must be exactly 9 digits
	if len(digits) != 9 {
		return false
	}
	
	// Check for invalid patterns
	invalidPatterns := []string{
		"000", // Area number cannot be 000
		"666", // Area number cannot be 666
	}
	
	areaNumber := digits[:3]
	for _, pattern := range invalidPatterns {
		if areaNumber == pattern {
			return false
		}
	}
	
	// Area number cannot start with 9
	if digits[0] == '9' {
		return false
	}
	
	// Group number cannot be 00
	if digits[3:5] == "00" {
		return false
	}
	
	// Serial number cannot be 0000
	if digits[5:9] == "0000" {
		return false
	}
	
	return true
}

// ValidatePhone performs basic phone number validation
func (fv *FormatValidator) ValidatePhone(phone string) bool {
	// Remove non-digit characters except +
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	
	// US phone number validation
	if strings.HasPrefix(cleaned, "+1") {
		digits := cleaned[2:]
		return len(digits) == 10 && digits[0] != '0' && digits[0] != '1'
	}
	
	// Domestic US number
	if len(cleaned) == 10 {
		return cleaned[0] != '0' && cleaned[0] != '1'
	}
	
	// 11 digits starting with 1
	if len(cleaned) == 11 && cleaned[0] == '1' {
		return cleaned[1] != '0' && cleaned[1] != '1'
	}
	
	return false
}

// FormatNormalizer provides data normalization for consistent formatting
type FormatNormalizer struct{}

// NewFormatNormalizer creates a new format normalizer
func NewFormatNormalizer() *FormatNormalizer {
	return &FormatNormalizer{}
}

// NormalizeCreditCard normalizes credit card numbers to a standard format
func (fn *FormatNormalizer) NormalizeCreditCard(cardNumber string) string {
	// Remove all non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")
	
	// Format as XXXX-XXXX-XXXX-XXXX
	if len(digits) >= 16 {
		return digits[:4] + "-" + digits[4:8] + "-" + digits[8:12] + "-" + digits[12:16]
	}
	
	return digits
}

// NormalizeSSN normalizes SSN to XXX-XX-XXXX format
func (fn *FormatNormalizer) NormalizeSSN(ssn string) string {
	// Remove all non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(ssn, "")
	
	if len(digits) == 9 {
		return digits[:3] + "-" + digits[3:5] + "-" + digits[5:9]
	}
	
	return digits
}

// NormalizePhone normalizes phone numbers to +1-XXX-XXX-XXXX format
func (fn *FormatNormalizer) NormalizePhone(phone string) string {
	// Remove all non-digit characters except +
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	
	// Handle different formats
	if strings.HasPrefix(cleaned, "+1") && len(cleaned) == 12 {
		digits := cleaned[2:]
		return "+1-" + digits[:3] + "-" + digits[3:6] + "-" + digits[6:10]
	}
	
	if len(cleaned) == 11 && cleaned[0] == '1' {
		digits := cleaned[1:]
		return "+1-" + digits[:3] + "-" + digits[3:6] + "-" + digits[6:10]
	}
	
	if len(cleaned) == 10 {
		return "+1-" + cleaned[:3] + "-" + cleaned[3:6] + "-" + cleaned[6:10]
	}
	
	return cleaned
}

// GetFormatDescription returns a human-readable description of the format
func GetFormatDescription(format DataFormat) string {
	descriptions := map[DataFormat]string{
		FormatCreditCard:   "Credit Card Number (preserves XXXX-XXXX-XXXX-XXXX format)",
		FormatSSN:          "Social Security Number (preserves XXX-XX-XXXX format)",
		FormatPhone:        "Phone Number (preserves +1-XXX-XXX-XXXX format)",
		FormatNumeric:      "Numeric String (preserves length and numeric characters)",
		FormatAlphaNumeric: "Alphanumeric String (preserves length and character types)",
		FormatCustom:       "Custom Format (preserves user-defined pattern)",
		FormatNone:         "No Format Preservation (secure random token)",
	}
	
	if desc, exists := descriptions[format]; exists {
		return desc
	}
	
	return "Unknown Format"
}

// GetSampleFormat returns a sample of what the format looks like
func GetSampleFormat(format DataFormat) string {
	samples := map[DataFormat]string{
		FormatCreditCard:   "1234-5678-9012-3456",
		FormatSSN:          "123-45-6789",
		FormatPhone:        "+1-555-123-4567",
		FormatNumeric:      "1234567890",
		FormatAlphaNumeric: "ABC123DEF",
		FormatCustom:       "(depends on pattern)",
		FormatNone:         "Base64 encoded token",
	}
	
	if sample, exists := samples[format]; exists {
		return sample
	}
	
	return "N/A"
}