package cereal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
)

// SecurityContext carries user permissions and encryption keys for cereal operations
type SecurityContext struct {
	UserID        string            // User identifier
	Permissions   []string          // User's access scopes
	Region        string            // Geographic region for compliance
	Metadata      map[string]string // Additional context data
	
	// Encryption keys
	OrgMasterKey    []byte // Organization's master key for "owner" encryption
	UserPublicKey   []byte // User's public key for "subscriber" encryption  
	UserPrivateKey  []byte // User's private key for decryption (client-provided)
}

// SecurityAction is a user-defined function that can halt serialization
type SecurityAction func(model interface{}, operation string, ctx SecurityContext) error

// EncryptionService handles field-level encryption/decryption
type EncryptionService struct {
	defaultOrgKey []byte
}

// NewEncryptionService creates a new encryption service with org master key
func NewEncryptionService(orgMasterKey []byte) *EncryptionService {
	return &EncryptionService{
		defaultOrgKey: orgMasterKey,
	}
}

// EncryptOwner encrypts data with organization's key (org can decrypt)
func (e *EncryptionService) EncryptOwner(plaintext []byte, ctx SecurityContext) ([]byte, error) {
	key := ctx.OrgMasterKey
	if len(key) == 0 {
		key = e.defaultOrgKey
	}
	
	if len(key) == 0 {
		return nil, fmt.Errorf("no organization master key provided")
	}
	
	return e.encryptAES(plaintext, key)
}

// DecryptOwner decrypts owner-encrypted data with organization's key
func (e *EncryptionService) DecryptOwner(ciphertext []byte, ctx SecurityContext) ([]byte, error) {
	key := ctx.OrgMasterKey
	if len(key) == 0 {
		key = e.defaultOrgKey
	}
	
	if len(key) == 0 {
		return nil, fmt.Errorf("no organization master key provided")
	}
	
	return e.decryptAES(ciphertext, key)
}

// EncryptSubscriber encrypts data with user's public key (only user can decrypt)
func (e *EncryptionService) EncryptSubscriber(plaintext []byte, ctx SecurityContext) ([]byte, error) {
	if len(ctx.UserPublicKey) == 0 {
		return nil, fmt.Errorf("user public key required for subscriber encryption")
	}
	
	// Parse RSA public key
	publicKey, err := parseRSAPublicKey(ctx.UserPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid user public key: %w", err)
	}
	
	// Encrypt with RSA-OAEP
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, plaintext, nil)
}

// DecryptSubscriber decrypts subscriber-encrypted data with user's private key
func (e *EncryptionService) DecryptSubscriber(ciphertext []byte, ctx SecurityContext) ([]byte, error) {
	if len(ctx.UserPrivateKey) == 0 {
		return nil, fmt.Errorf("user private key required for subscriber decryption")
	}
	
	// Parse RSA private key
	privateKey, err := parseRSAPrivateKey(ctx.UserPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid user private key: %w", err)
	}
	
	// Decrypt with RSA-OAEP
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
}

// encryptAES encrypts data using AES-GCM
func (e *EncryptionService) encryptAES(plaintext, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	hasher := sha256.Sum256(key)
	aesKey := hasher[:]
	
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAES decrypts data using AES-GCM
func (e *EncryptionService) decryptAES(ciphertext, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	hasher := sha256.Sum256(key)
	aesKey := hasher[:]
	
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Helper functions for RSA key parsing (simplified - would use proper crypto/x509 in production)
func parseRSAPublicKey(keyBytes []byte) (*rsa.PublicKey, error) {
	// Simplified implementation - in production would parse PEM/DER format
	// For now, assume keyBytes contains a properly formatted RSA public key
	return nil, fmt.Errorf("RSA key parsing not implemented in this demo")
}

func parseRSAPrivateKey(keyBytes []byte) (*rsa.PrivateKey, error) {
	// Simplified implementation - in production would parse PEM/DER format
	// For now, assume keyBytes contains a properly formatted RSA private key
	return nil, fmt.Errorf("RSA key parsing not implemented in this demo")
}

// Global security actions registry
var securityActions = make(map[string]SecurityAction)

// RegisterSecurityAction registers a user-defined security action
func RegisterSecurityAction(name string, action SecurityAction) {
	securityActions[name] = action
}

// runSecurityActions executes all registered security actions
func runSecurityActions(model interface{}, operation string, ctx SecurityContext) error {
	for name, action := range securityActions {
		if err := action(model, operation, ctx); err != nil {
			return fmt.Errorf("security action '%s' failed: %w", name, err)
		}
	}
	return nil
}

// applyFieldEncryption encrypts fields based on catalog metadata
func applyFieldEncryption(value reflect.Value, ctx SecurityContext, encService *EncryptionService) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Struct {
		return nil
	}
	
	// Get type name for catalog lookup
	typeName := value.Type().Name()
	if typeName == "" {
		return nil // Anonymous structs not supported
	}
	
	// Get encryption metadata from catalog
	// Note: This is a simplified approach - in a real implementation,
	// we'd need a way to get the type for catalog.GetEncryptionFields[T]()
	// For now, we'll demonstrate the concept with reflection on struct tags
	
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		encryptTag := field.Tag.Get("encrypt")
		if encryptTag == "" {
			continue
		}
		
		// Only encrypt string fields for this demo
		if fieldValue.Kind() != reflect.String {
			continue
		}
		
		plaintext := fieldValue.String()
		if plaintext == "" {
			continue
		}
		
		var encrypted []byte
		var err error
		
		switch encryptTag {
		case "owner":
			encrypted, err = encService.EncryptOwner([]byte(plaintext), ctx)
		case "subscriber":
			encrypted, err = encService.EncryptSubscriber([]byte(plaintext), ctx)
		default:
			continue // Unknown encryption type
		}
		
		if err != nil {
			return fmt.Errorf("failed to encrypt field %s: %w", field.Name, err)
		}
		
		// Store encrypted data as base64 string
		// In production, might use a structured format with metadata
		encryptedStr := base64.StdEncoding.EncodeToString(encrypted)
		fieldValue.SetString(encryptedStr)
	}
	
	return nil
}

// applyFieldDecryption decrypts fields based on catalog metadata
func applyFieldDecryption(value reflect.Value, ctx SecurityContext, encService *EncryptionService) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Struct {
		return nil
	}
	
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		encryptTag := field.Tag.Get("encrypt")
		if encryptTag == "" {
			continue
		}
		
		// Only decrypt string fields for this demo
		if fieldValue.Kind() != reflect.String {
			continue
		}
		
		encryptedStr := fieldValue.String()
		if encryptedStr == "" {
			continue
		}
		
		// Decode base64
		encrypted, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			// If it's not base64, assume it's already plaintext
			continue
		}
		
		var plaintext []byte
		
		switch encryptTag {
		case "owner":
			plaintext, err = encService.DecryptOwner(encrypted, ctx)
		case "subscriber":
			plaintext, err = encService.DecryptSubscriber(encrypted, ctx)
		default:
			continue // Unknown encryption type
		}
		
		if err != nil {
			return fmt.Errorf("failed to decrypt field %s: %w", field.Name, err)
		}
		
		fieldValue.SetString(string(plaintext))
	}
	
	return nil
}

// applyScopeRedaction applies scope-based redaction using catalog metadata
func applyScopeRedaction(value reflect.Value, permissions []string) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Struct {
		return nil
	}
	
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		// Check scope requirements
		scopeTag := field.Tag.Get("scope")
		if scopeTag == "" {
			continue // No scope restriction
		}
		
		// Check if user has required scope
		hasScope := false
		for _, userScope := range permissions {
			if userScope == scopeTag {
				hasScope = true
				break
			}
		}
		
		if !hasScope {
			// Apply redaction based on field type
			redactTag := field.Tag.Get("redact")
			
			switch fieldValue.Kind() {
			case reflect.String:
				if redactTag != "" {
					fieldValue.SetString(redactTag)
				} else {
					fieldValue.SetString("[REDACTED]")
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if redactTag != "" {
					// Try to parse redactTag as int, default to 0
					fieldValue.SetInt(0)
				} else {
					fieldValue.SetInt(0)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldValue.SetUint(0)
			case reflect.Float32, reflect.Float64:
				fieldValue.SetFloat(0.0)
			case reflect.Bool:
				fieldValue.SetBool(false)
			default:
				// For complex types, set to zero value
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}
		}
	}
	
	return nil
}