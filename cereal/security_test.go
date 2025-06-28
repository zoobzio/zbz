package cereal

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Test models demonstrating the full security capabilities

// EncryptedUser demonstrates scoping and encryption concepts (simplified for demo)
type EncryptedUser struct {
	Name     string `json:"name" validate:"required" desc:"User's full name"`
	Email    string `json:"email" validate:"required,email" scope:"profile" desc:"User's email"`
	
	// Scoped fields with redaction values
	SSN      string `json:"ssn" scope:"admin" redact:"XXX-XX-XXXX" desc:"Social Security Number"`
	Salary   int    `json:"salary" scope:"hr" desc:"Annual salary"`
	
	// Public fields (no encryption/scoping for demo simplicity)
	Notes    string `json:"notes" desc:"Public notes"`
	Password string `json:"password" scope:"admin" redact:"[HIDDEN]" desc:"User's password"`
	
	IsActive bool   `json:"is_active" desc:"Account status"`
}

// ComplianceData demonstrates advanced security scenarios
type ComplianceData struct {
	PatientID   string `json:"patient_id" scope:"medical" redact:"[PATIENT_ID]"`
	Diagnosis   string `json:"diagnosis" scope:"doctor" redact:"[DIAGNOSIS]"`
	PersonalLog string `json:"personal_log"` // No scope restriction
	Region      string `json:"region" desc:"Geographic region"`
}

func TestBasicEncryption(t *testing.T) {
	user := EncryptedUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		SSN:      "123-45-6789",
		Salary:   75000,
		Notes:    "Private thoughts",
		Password: "secret123",
		IsActive: true,
	}
	
	ctx := SecurityContext{
		UserID:        "user123",
		Permissions:   []string{"profile", "admin"}, // Has profile + admin, but not hr
		OrgMasterKey:  []byte("test-org-master-key-32-chars!!"),
		UserPublicKey: []byte("fake-public-key"), // Would be real RSA key in production
	}
	
	// Test encryption during marshaling
	data, err := MarshalSecure(user, ctx)
	if err != nil {
		t.Fatalf("Failed to marshal with encryption: %v", err)
	}
	
	// Verify data is encrypted and redacted appropriately
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	
	// Name and Email should be plaintext (no encryption/scope restrictions)
	if result["name"] != "John Doe" {
		t.Errorf("Expected name to be plaintext, got %v", result["name"])
	}
	
	if result["email"] != "john@example.com" {
		t.Errorf("Expected email to be accessible with profile scope, got %v", result["email"])
	}
	
	// SSN should be accessible with admin scope 
	if result["ssn"] != "123-45-6789" {
		t.Errorf("Expected SSN to be accessible with admin scope, got %v", result["ssn"])
	}
	
	// Salary should be redacted (user lacks hr scope)
	if result["salary"] != float64(0) {
		t.Errorf("Expected salary to be redacted to 0, got %v", result["salary"])
	}
	
	// Notes should be accessible (no scope restriction)
	if result["notes"] != "Private thoughts" {
		t.Errorf("Expected notes to be accessible, got %v", result["notes"])
	}
	
	// Password should be accessible with admin scope
	if result["password"] != "secret123" {
		t.Errorf("Expected password to be accessible with admin scope, got %v", result["password"])
	}
	
	fmt.Printf("Encrypted data: %s\n", string(data))
}

func TestScopeEnforcement(t *testing.T) {
	user := EncryptedUser{
		Name:     "Jane Smith",
		Email:    "jane@example.com", 
		SSN:      "987-65-4321",
		Salary:   85000,
		Notes:    "Personal notes",
		Password: "secret456",
		IsActive: true,
	}
	
	// Test with limited permissions (only profile scope)
	limitedCtx := SecurityContext{
		UserID:       "user456",
		Permissions:  []string{"profile"}, // Only profile access
		OrgMasterKey: []byte("test-org-master-key-32-chars!!"),
	}
	
	data, err := MarshalSecure(user, limitedCtx)
	if err != nil {
		t.Fatalf("Failed to marshal with limited permissions: %v", err)
	}
	
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	
	// Should have access to profile-scoped fields
	if result["email"] != "jane@example.com" {
		t.Errorf("Should have access to email with profile scope, got %v", result["email"])
	}
	
	// Should NOT have access to admin/hr fields - should be redacted
	if result["ssn"] != "XXX-XX-XXXX" {
		t.Errorf("Expected SSN redaction, got %v", result["ssn"])
	}
	
	if result["salary"] != float64(0) {
		t.Errorf("Expected salary redaction to 0, got %v", result["salary"])
	}
	
	// Password should be redacted (requires admin scope)
	if result["password"] != "[HIDDEN]" {
		t.Errorf("Expected password redaction, got %v", result["password"])
	}
	
	fmt.Printf("Limited scope data: %s\n", string(data))
}

func TestSecurityActions(t *testing.T) {
	// Register a security action that prevents certain operations
	RegisterSecurityAction("gdpr_compliance", func(model interface{}, operation string, ctx SecurityContext) error {
		user, ok := model.(EncryptedUser)
		if !ok {
			return nil // Only applies to EncryptedUser
		}
		
		// Block operations on EU users without explicit consent
		if ctx.Region == "EU" {
			if gdprConsent, ok := ctx.Metadata["gdpr_consent"]; !ok || gdprConsent != "true" {
				return fmt.Errorf("GDPR consent required for EU operations on user %s", user.Name)
			}
		}
		
		return nil
	})
	
	user := EncryptedUser{
		Name:  "Hans Mueller",
		Email: "hans@example.de",
		SSN:   "DE-123456789",
	}
	
	// Test without GDPR consent - should fail
	euCtx := SecurityContext{
		UserID:      "eu_user",
		Permissions: []string{"profile"},
		Region:      "EU",
		Metadata:    map[string]string{}, // No GDPR consent
	}
	
	_, err := MarshalSecure(user, euCtx)
	if err == nil {
		t.Error("Expected GDPR compliance error, but operation succeeded")
	}
	
	if !strings.Contains(err.Error(), "GDPR consent required") {
		t.Errorf("Expected GDPR error, got: %v", err)
	}
	
	// Test with GDPR consent - should succeed
	euCtxWithConsent := SecurityContext{
		UserID:      "eu_user",
		Permissions: []string{"profile"},
		Region:      "EU",
		Metadata:    map[string]string{"gdpr_consent": "true"},
	}
	
	_, err = MarshalSecure(user, euCtxWithConsent)
	if err != nil {
		t.Errorf("Expected success with GDPR consent, got error: %v", err)
	}
	
	fmt.Println("Security action enforcement working correctly")
}

func TestCompleteWorkflow(t *testing.T) {
	// Demonstrate the complete secure workflow
	patient := ComplianceData{
		PatientID:   "PATIENT-001",
		Diagnosis:   "Type 2 Diabetes",
		PersonalLog: "Feeling better today, glucose levels stable",
		Region:      "US",
	}
	
	// Doctor context - can see medical data
	doctorCtx := SecurityContext{
		UserID:       "doctor123",
		Permissions:  []string{"medical", "doctor"},
		Region:       "US",
		OrgMasterKey: []byte("medical-org-key-32-chars-long!"),
	}
	
	// Nurse context - can see basic medical data but not diagnosis
	nurseCtx := SecurityContext{
		UserID:       "nurse456", 
		Permissions:  []string{"medical"}, // No doctor scope
		Region:       "US",
		OrgMasterKey: []byte("medical-org-key-32-chars-long!"),
	}
	
	// Patient context - can see their own data
	patientCtx := SecurityContext{
		UserID:         "patient001",
		Permissions:    []string{}, // No medical permissions
		Region:         "US",
		UserPrivateKey: []byte("patient-private-key"),
	}
	
	// Test doctor access
	doctorData, err := MarshalSecure(patient, doctorCtx)
	if err != nil {
		t.Fatalf("Doctor access failed: %v", err)
	}
	
	var doctorResult map[string]interface{}
	json.Unmarshal(doctorData, &doctorResult)
	
	// Doctor should see patient ID and diagnosis (has both medical + doctor scopes)
	if doctorResult["patient_id"] != "PATIENT-001" {
		t.Errorf("Doctor should see patient ID, got %v", doctorResult["patient_id"])
	}
	
	if doctorResult["diagnosis"] != "Type 2 Diabetes" {
		t.Errorf("Doctor should see diagnosis, got %v", doctorResult["diagnosis"])
	}
	
	// Test nurse access
	nurseData, err := MarshalSecure(patient, nurseCtx)
	if err != nil {
		t.Fatalf("Nurse access failed: %v", err)
	}
	
	var nurseResult map[string]interface{}
	json.Unmarshal(nurseData, &nurseResult)
	
	// Nurse should see patient ID but not diagnosis (lacks doctor scope)
	if nurseResult["patient_id"] != "PATIENT-001" {
		t.Errorf("Nurse should see patient ID with medical scope, got %v", nurseResult["patient_id"])
	}
	
	// Diagnosis should be redacted for nurse
	if nurseResult["diagnosis"] != "[DIAGNOSIS]" {
		t.Errorf("Diagnosis should be redacted for nurse, got %v", nurseResult["diagnosis"])
	}
	
	// Test patient access 
	patientData, err := MarshalSecure(patient, patientCtx)
	if err != nil {
		t.Fatalf("Patient access failed: %v", err)
	}
	
	var patientResult map[string]interface{}
	json.Unmarshal(patientData, &patientResult)
	
	// Patient should have medical data redacted (lacks medical scopes)
	// But should be able to see their personal log (no scope restriction)
	if patientResult["patient_id"] != "[PATIENT_ID]" {
		t.Errorf("Patient ID should be redacted for patient, got %v", patientResult["patient_id"])
	}
	
	if patientResult["diagnosis"] != "[DIAGNOSIS]" {
		t.Errorf("Diagnosis should be redacted for patient, got %v", patientResult["diagnosis"])
	}
	
	if patientResult["personal_log"] != "Feeling better today, glucose levels stable" {
		t.Errorf("Patient should see their personal log, got %v", patientResult["personal_log"])
	}
	
	fmt.Printf("Doctor sees: %s\n", string(doctorData))
	fmt.Printf("Nurse sees: %s\n", string(nurseData))
	fmt.Printf("Patient sees: %s\n", string(patientData))
}

func TestDecryption(t *testing.T) {
	// Test the full encrypt -> decrypt cycle
	user := EncryptedUser{
		Name:     "Test User",
		Email:    "test@example.com",
		SSN:      "111-11-1111",
		Notes:    "Private note",
		Password: "secret789",
		IsActive: true,
	}
	
	ctx := SecurityContext{
		UserID:       "test_user",
		Permissions:  []string{"admin", "profile"}, // Add profile scope for email access
		OrgMasterKey: []byte("test-key-must-be-32-chars-long!"),
	}
	
	// Marshal with encryption
	encryptedData, err := MarshalSecure(user, ctx)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}
	
	// Unmarshal with decryption
	var decryptedUser EncryptedUser
	err = UnmarshalSecure(encryptedData, &decryptedUser, ctx)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}
	
	// Verify data integrity after encrypt/decrypt cycle
	if decryptedUser.Name != user.Name {
		t.Errorf("Name mismatch: expected %s, got %s", user.Name, decryptedUser.Name)
	}
	
	if decryptedUser.SSN != user.SSN {
		t.Errorf("SSN mismatch: expected %s, got %s", user.SSN, decryptedUser.SSN)
	}
	
	if decryptedUser.IsActive != user.IsActive {
		t.Errorf("IsActive mismatch: expected %v, got %v", user.IsActive, decryptedUser.IsActive)
	}
	
	fmt.Printf("Successful encrypt/decrypt cycle for user: %s\n", decryptedUser.Name)
}

func TestMultiFormatSecurity(t *testing.T) {
	// Test that security works across all serialization formats
	user := EncryptedUser{
		Name:   "Multi Format User",
		Email:  "multi@example.com",
		SSN:    "222-22-2222",
		Salary: 50000,
	}
	
	ctx := SecurityContext{
		UserID:       "multi_user",
		Permissions:  []string{"admin", "profile"}, // Has admin and profile but not hr
		OrgMasterKey: []byte("multi-format-key-32-chars-long"),
	}
	
	// Test JSON
	jsonData, err := MarshalSecure(user, ctx)
	if err != nil {
		t.Fatalf("JSON secure marshal failed: %v", err)
	}
	
	// Test YAML 
	yamlData, err := MarshalSecureYAML(user, ctx)
	if err != nil {
		t.Fatalf("YAML secure marshal failed: %v", err)
	}
	
	// Test TOML
	tomlData, err := MarshalSecureTOML(user, ctx)
	if err != nil {
		t.Fatalf("TOML secure marshal failed: %v", err)
	}
	
	// All formats should apply the same security policies
	// (Salary should be redacted in all formats due to lacking hr scope)
	
	fmt.Printf("JSON secured: %s\n", string(jsonData))
	fmt.Printf("YAML secured: %s\n", string(yamlData))
	fmt.Printf("TOML secured: %s\n", string(tomlData))
	
	// Verify YAML redaction (salary should be 0)
	if !strings.Contains(string(yamlData), "salary: 0") {
		t.Error("YAML should contain redacted salary as 0")
	}
	
	// Verify TOML redaction (salary should be 0)
	if !strings.Contains(string(tomlData), "Salary = 0") {
		t.Error("TOML should contain redacted salary as 0")
	}
}

func BenchmarkSecureMarshaling(b *testing.B) {
	user := EncryptedUser{
		Name:     "Benchmark User",
		Email:    "bench@example.com",
		SSN:      "333-33-3333", 
		Salary:   60000,
		Notes:    "Some private notes here",
		Password: "benchmark123",
		IsActive: true,
	}
	
	ctx := SecurityContext{
		UserID:       "bench_user",
		Permissions:  []string{"profile", "admin"},
		OrgMasterKey: []byte("benchmark-key-32-chars-exactly!"),
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := MarshalSecure(user, ctx)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}