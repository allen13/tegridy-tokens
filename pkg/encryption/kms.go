package encryption

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/allen13/tegridy-tokens/pkg/tokenizer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSEncryptor implements the Encryptor interface using AWS KMS
type KMSEncryptor struct {
	client       *kms.Client
	keyID        string
	useEnvelope  bool
	dataKeyCache *dataKeyCache
}

// KMSConfig holds configuration for KMS encryptor
type KMSConfig struct {
	Client          *kms.Client
	KeyID           string
	UseEnvelope     bool  // Use envelope encryption for better performance
	CacheDataKeys   bool  // Cache data keys for envelope encryption
	CacheSize       int   // Number of data keys to cache
	CacheTTLSeconds int64 // TTL for cached data keys
}

// dataKeyCache caches data keys for envelope encryption
type dataKeyCache struct {
	mu       sync.RWMutex
	keys     map[string]*cachedDataKey
	maxSize  int
	ttl      int64
}

type cachedDataKey struct {
	plaintext  []byte
	ciphertext []byte
	createdAt  int64
}

// NewKMSEncryptor creates a new KMS encryptor
func NewKMSEncryptor(cfg KMSConfig) tokenizer.Encryptor {
	enc := &KMSEncryptor{
		client:      cfg.Client,
		keyID:       cfg.KeyID,
		useEnvelope: cfg.UseEnvelope,
	}
	
	if cfg.CacheDataKeys && cfg.UseEnvelope {
		cacheSize := cfg.CacheSize
		if cacheSize <= 0 {
			cacheSize = 100
		}
		ttl := cfg.CacheTTLSeconds
		if ttl <= 0 {
			ttl = 300 // 5 minutes default
		}
		
		enc.dataKeyCache = &dataKeyCache{
			keys:    make(map[string]*cachedDataKey),
			maxSize: cacheSize,
			ttl:     ttl,
		}
	}
	
	return enc
}

// Encrypt encrypts data using KMS
func (e *KMSEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	if e.useEnvelope {
		return e.envelopeEncrypt(ctx, plaintext)
	}
	return e.directEncrypt(ctx, plaintext)
}

// Decrypt decrypts data using KMS
func (e *KMSEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if e.useEnvelope {
		return e.envelopeDecrypt(ctx, ciphertext)
	}
	return e.directDecrypt(ctx, ciphertext)
}

// GenerateDataKey generates a data key for envelope encryption
func (e *KMSEncryptor) GenerateDataKey(ctx context.Context) ([]byte, []byte, error) {
	input := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(e.keyID),
		KeySpec: types.DataKeySpecAes256,
	}
	
	result, err := e.client.GenerateDataKey(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate data key: %w", err)
	}
	
	return result.Plaintext, result.CiphertextBlob, nil
}

// directEncrypt uses KMS directly (limited to 4KB)
func (e *KMSEncryptor) directEncrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	// KMS direct encryption is limited to 4096 bytes
	if len(plaintext) > 4096 {
		return nil, fmt.Errorf("plaintext too large for direct KMS encryption (max 4096 bytes)")
	}
	
	input := &kms.EncryptInput{
		KeyId:     aws.String(e.keyID),
		Plaintext: plaintext,
	}
	
	result, err := e.client.Encrypt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("KMS encryption failed: %w", err)
	}
	
	return result.CiphertextBlob, nil
}

// directDecrypt uses KMS directly
func (e *KMSEncryptor) directDecrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	input := &kms.DecryptInput{
		CiphertextBlob: ciphertext,
	}
	
	result, err := e.client.Decrypt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("KMS decryption failed: %w", err)
	}
	
	return result.Plaintext, nil
}

// envelopeEncrypt uses envelope encryption for better performance and no size limits
func (e *KMSEncryptor) envelopeEncrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	// Get or generate data key
	dataKeyPlaintext, dataKeyCiphertext, err := e.getOrGenerateDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(dataKeyPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	// Combine: data key ciphertext length (4 bytes) + data key ciphertext + encrypted data
	dataKeyCiphertextLen := int32(len(dataKeyCiphertext))
	result := make([]byte, 4+len(dataKeyCiphertext)+len(ciphertext))
	
	// Write data key ciphertext length
	result[0] = byte(dataKeyCiphertextLen >> 24)
	result[1] = byte(dataKeyCiphertextLen >> 16)
	result[2] = byte(dataKeyCiphertextLen >> 8)
	result[3] = byte(dataKeyCiphertextLen)
	
	// Write data key ciphertext
	copy(result[4:4+len(dataKeyCiphertext)], dataKeyCiphertext)
	
	// Write encrypted data
	copy(result[4+len(dataKeyCiphertext):], ciphertext)
	
	return result, nil
}

// envelopeDecrypt decrypts envelope encrypted data
func (e *KMSEncryptor) envelopeDecrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 4 {
		return nil, fmt.Errorf("invalid ciphertext: too short")
	}
	
	// Read data key ciphertext length
	dataKeyCiphertextLen := int32(ciphertext[0])<<24 | int32(ciphertext[1])<<16 | 
		int32(ciphertext[2])<<8 | int32(ciphertext[3])
	
	if len(ciphertext) < 4+int(dataKeyCiphertextLen) {
		return nil, fmt.Errorf("invalid ciphertext: data key length mismatch")
	}
	
	// Extract data key ciphertext
	dataKeyCiphertext := ciphertext[4 : 4+dataKeyCiphertextLen]
	
	// Extract encrypted data
	encryptedData := ciphertext[4+dataKeyCiphertextLen:]
	
	// Decrypt data key
	dataKeyPlaintext, err := e.decryptDataKey(ctx, dataKeyCiphertext)
	if err != nil {
		return nil, err
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(dataKeyPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Decrypt data
	if len(encryptedData) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}
	
	nonce, ciphertext := encryptedData[:gcm.NonceSize()], encryptedData[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	
	return plaintext, nil
}

// getOrGenerateDataKey gets a cached data key or generates a new one
func (e *KMSEncryptor) getOrGenerateDataKey(ctx context.Context) ([]byte, []byte, error) {
	// If caching is disabled, always generate new key
	if e.dataKeyCache == nil {
		return e.GenerateDataKey(ctx)
	}
	
	// Try to get from cache
	if key := e.dataKeyCache.get(); key != nil {
		return key.plaintext, key.ciphertext, nil
	}
	
	// Generate new key
	plaintext, ciphertext, err := e.GenerateDataKey(ctx)
	if err != nil {
		return nil, nil, err
	}
	
	// Cache the key
	e.dataKeyCache.put(plaintext, ciphertext)
	
	return plaintext, ciphertext, nil
}

// decryptDataKey decrypts a data key
func (e *KMSEncryptor) decryptDataKey(ctx context.Context, ciphertext []byte) ([]byte, error) {
	// Check cache first
	if e.dataKeyCache != nil {
		ciphertextStr := base64.StdEncoding.EncodeToString(ciphertext)
		if key := e.dataKeyCache.getByChiphertext(ciphertextStr); key != nil {
			return key.plaintext, nil
		}
	}
	
	// Decrypt using KMS
	input := &kms.DecryptInput{
		CiphertextBlob: ciphertext,
	}
	
	result, err := e.client.Decrypt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key: %w", err)
	}
	
	// Cache the decrypted key
	if e.dataKeyCache != nil {
		e.dataKeyCache.put(result.Plaintext, ciphertext)
	}
	
	return result.Plaintext, nil
}

// get retrieves a random cached data key
func (c *dataKeyCache) get() *cachedDataKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	now := time.Now().Unix()
	
	// Find any valid key
	for _, key := range c.keys {
		if now-key.createdAt < c.ttl {
			return key
		}
	}
	
	return nil
}

// getByChiphertext retrieves a cached data key by its ciphertext
func (c *dataKeyCache) getByChiphertext(ciphertextStr string) *cachedDataKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key, exists := c.keys[ciphertextStr]
	if !exists {
		return nil
	}
	
	now := time.Now().Unix()
	if now-key.createdAt >= c.ttl {
		return nil
	}
	
	return key
}

// put adds a data key to the cache
func (c *dataKeyCache) put(plaintext, ciphertext []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Evict old keys if at capacity
	if len(c.keys) >= c.maxSize {
		// Simple eviction: remove first found expired key or oldest key
		now := time.Now().Unix()
		var oldestKey string
		var oldestTime int64 = time.Now().Unix()
		
		for k, v := range c.keys {
			if now-v.createdAt >= c.ttl {
				delete(c.keys, k)
				break
			}
			if v.createdAt < oldestTime {
				oldestKey = k
				oldestTime = v.createdAt
			}
		}
		
		// If no expired key found and still at capacity, remove oldest
		if len(c.keys) >= c.maxSize && oldestKey != "" {
			delete(c.keys, oldestKey)
		}
	}
	
	ciphertextStr := base64.StdEncoding.EncodeToString(ciphertext)
	c.keys[ciphertextStr] = &cachedDataKey{
		plaintext:  plaintext,
		ciphertext: ciphertext,
		createdAt:  time.Now().Unix(),
	}
}