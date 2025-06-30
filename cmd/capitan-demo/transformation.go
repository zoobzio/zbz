package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"zbz/capitan"
)

// Demo data transformation pipeline
func transformationDemo() {
	println("\n4. 🔄 Event Transformation Demo")
	println("   (Events triggering other events)")
	
	// Stage 1: Raw user data → Enriched user data
	capitan.RegisterByteHandler("user.raw_signup", func(data []byte) error {
		var rawData map[string]any
		json.Unmarshal(data, &rawData)
		
		// Simulate data enrichment
		enrichedData := map[string]any{
			"user_id":     rawData["user_id"],
			"email":       rawData["email"],
			"plan":        rawData["plan"],
			"enriched_at": time.Now(),
			"geo_region":  "US-West", // Simulated geo lookup
			"user_tier":   "standard", // Simulated tier calculation
		}
		
		fmt.Printf("   📍 Stage 1: Enriched user data for %s\n", rawData["user_id"])
		
		// Emit enriched event
		capitan.EmitEvent("user.enriched", enrichedData)
		return nil
	})
	
	// Stage 2: Enriched user data → Welcome workflow
	capitan.RegisterByteHandler("user.enriched", func(data []byte) error {
		var enrichedData map[string]any
		json.Unmarshal(data, &enrichedData)
		
		// Simulate welcome workflow
		workflowData := map[string]any{
			"user_id":      enrichedData["user_id"],
			"workflow_id":  fmt.Sprintf("welcome_%s", enrichedData["user_id"]),
			"steps": []string{
				"send_welcome_email",
				"create_onboarding_tasks", 
				"schedule_followup",
			},
			"region": enrichedData["geo_region"],
		}
		
		fmt.Printf("   📧 Stage 2: Welcome workflow started for %s\n", enrichedData["user_id"])
		
		// Emit workflow event
		capitan.EmitEvent("workflow.welcome_started", workflowData)
		return nil
	})
	
	// Stage 3: Welcome workflow → Business metrics
	capitan.RegisterByteHandler("workflow.welcome_started", func(data []byte) error {
		var workflowData map[string]any
		json.Unmarshal(data, &workflowData)
		
		// Simulate business metrics collection
		metricsData := map[string]any{
			"metric_type":    "user_acquisition",
			"user_id":        workflowData["user_id"],
			"acquisition_cost": 50.0, // Simulated cost
			"ltv_prediction":   500.0, // Simulated LTV
			"roi_ratio":        10.0,
			"region":          workflowData["region"],
		}
		
		fmt.Printf("   💰 Stage 3: Business metrics calculated for %s (ROI: 10x)\n", 
			workflowData["user_id"])
		
		// Emit metrics event
		capitan.EmitEvent("metrics.acquisition", metricsData)
		return nil
	})
	
	// Stage 4: Business metrics → Executive dashboard
	capitan.RegisterByteHandler("metrics.acquisition", func(data []byte) error {
		var metricsData map[string]any
		json.Unmarshal(data, &metricsData)
		
		fmt.Printf("   📊 Stage 4: Executive dashboard updated (User: %s, ROI: %.1fx)\n",
			metricsData["user_id"], metricsData["roi_ratio"])
		
		return nil
	})
	
	// Trigger the transformation pipeline
	println("   Starting transformation pipeline...")
	
	capitan.EmitEvent("user.raw_signup", map[string]any{
		"user_id": "transform_user_1",
		"email":   "transform@example.com",
		"plan":    "enterprise",
	})
	
	time.Sleep(200 * time.Millisecond)
	
	println("   ✅ Transformation pipeline complete")
	println("     Raw signup → Enrichment → Workflow → Metrics → Dashboard")
}

// Demo typed event handling
type UserSignupEvent struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
}

type UserHookType int

const (
	UserCreated UserHookType = iota
	UserUpdated
	UserDeleted
)

func (u UserHookType) String() string {
	switch u {
	case UserCreated:
		return "user.created"
	case UserUpdated:
		return "user.updated"
	case UserDeleted:
		return "user.deleted"
	default:
		return "user.unknown"
	}
}

func typedEventDemo() {
	println("\n5. 🎯 Typed Event Demo")
	println("   (Type-safe event handling)")
	
	// Register typed handler
	capitan.RegisterInput[UserSignupEvent](UserCreated, func(event UserSignupEvent) error {
		fmt.Printf("   ✅ Typed handler: User %s signed up with plan %s\n", 
			event.UserID, event.Plan)
		return nil
	})
	
	// Emit typed event
	ctx := context.Background()
	capitan.Emit(ctx, UserCreated, "demo", UserSignupEvent{
		UserID: "typed_user_1",
		Email:  "typed@example.com", 
		Plan:   "enterprise",
	}, map[string]any{
		"source": "typed_demo",
	})
	
	time.Sleep(50 * time.Millisecond)
}