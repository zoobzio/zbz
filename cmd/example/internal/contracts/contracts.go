package contracts

import (
	"zbz/cmd/example/internal/models"
	"zbz/api"
	"zbz/api/auth"
	"zbz/api/cache"
	"zbz/api/database"
	"zbz/zlog"
)

// createPrimaryDatabaseDriver creates the PostgreSQL driver for the primary database
func createPrimaryDatabaseDriver() zbz.DatabaseDriver {
	driver, err := database.NewPostgreSQLDriver(zbz.GetConfig().DSN())
	if err != nil {
		zlog.Fatal("Failed to create PostgreSQL driver", zlog.Err(err))
	}
	return driver
}

// PrimaryDatabase is the shared database contract that all cores can reference
var PrimaryDatabase = zbz.DatabaseContract{
	BaseContract: zbz.BaseContract{
		Name:        "primary",
		Description: "Primary application database for all core business data",
	},
	Driver: createPrimaryDatabaseDriver(), // User-initialized PostgreSQL driver
}

// HTTPContract defines the HTTP service configuration
var HTTPContract = zbz.HTTPContract{
	BaseContract: zbz.BaseContract{
		Name:        "primary",
		Description: "Primary HTTP server for the application",
	},
	Driver:       "default", // Will use default implementation for now
	Port:         "8080",
	Host:         "0.0.0.0",
	DevMode:      true,
	TemplatesDir: "internal/templates",
}

// createAuth0Driver creates the Auth0 driver for authentication
func createAuth0Driver() zbz.AuthDriver {
	// Create memory cache for auth data
	memoryCache := cache.NewMemoryCache()
	
	// Get global configuration
	globalConfig := zbz.GetConfig()
	
	// Auth0 configuration from global config
	config := &auth.AuthConfig{
		Domain:       globalConfig.AuthDomain(),
		ClientID:     globalConfig.AuthClientID(),
		ClientSecret: globalConfig.AuthClientSecret(),
		RedirectURL:  globalConfig.AuthCallback(),
		Scopes:       []string{"openid", "profile", "email", "read:users", "write:users"},
	}
	
	zlog.Info("Auth0 config from environment", 
		zlog.String("domain", config.Domain),
		zlog.String("client_id", config.ClientID),
		zlog.String("redirect_url", config.RedirectURL))
	
	zlog.Info("Created Auth0 driver", 
		zlog.String("domain", config.Domain),
		zlog.String("client_id", config.ClientID),
		zlog.String("redirect_url", config.RedirectURL))
	
	driver := auth.NewAuth0Auth(memoryCache, config)
	return driver
}

// AuthContract defines the authentication service configuration
// This will only be created if proper Auth0 environment variables are set
var AuthContract = zbz.AuthContract{
	BaseContract: zbz.BaseContract{
		Name:        "primary",
		Description: "Primary Auth0 authentication service",
	},
	Driver: createAuth0Driver(),
}


// ContactContract defines the contract for contact management
var ContactContract = zbz.CoreContract[models.Contact]{
	BaseContract: zbz.BaseContract{
		Name:        "Contact",
		Description: "contact_resource", // Remark key for resource description
	},
	ModelDescription: "contact_model", // Remark key for model description
	DatabaseContract: PrimaryDatabase,
}

// CompanyContract defines the contract for company management
var CompanyContract = zbz.CoreContract[models.Company]{
	BaseContract: zbz.BaseContract{
		Name:        "Company",
		Description: "company_resource", // Remark key for resource description
	},
	ModelDescription: "company_model", // Remark key for model description
	DatabaseContract: PrimaryDatabase,
}

// FormContract defines the contract for form management
var FormContract = zbz.CoreContract[models.Form]{
	BaseContract: zbz.BaseContract{
		Name:        "Form",
		Description: "form_resource", // Remark key for resource description
	},
	ModelDescription: "form_model", // Remark key for model description
	DatabaseContract: PrimaryDatabase,
}

// PropertyContract defines the contract for property management
var PropertyContract = zbz.CoreContract[models.Property]{
	BaseContract: zbz.BaseContract{
		Name:        "Property",
		Description: "property_resource", // Remark key for resource description
	},
	ModelDescription: "property_model", // Remark key for model description
	DatabaseContract: PrimaryDatabase,
}

// FieldContract defines the contract for field management
var FieldContract = zbz.CoreContract[models.Field]{
	BaseContract: zbz.BaseContract{
		Name:        "Field",
		Description: "field_resource", // Remark key for resource description
	},
	ModelDescription: "field_model", // Remark key for model description
	DatabaseContract: PrimaryDatabase,
}