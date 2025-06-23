package contracts

import (
	"zbz/cmd/example/internal/models"
	"zbz/lib"
	"zbz/lib/database"
	"zbz/shared/logger"
)

// createPrimaryDatabaseDriver creates the PostgreSQL driver for the primary database
func createPrimaryDatabaseDriver() zbz.DatabaseDriver {
	driver, err := database.NewPostgreSQLDriver(zbz.GetConfig().DSN())
	if err != nil {
		logger.Fatal("Failed to create PostgreSQL driver", logger.Err(err))
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
	Driver:   "default", // Will use default implementation for now
	Port:     "8080",
	Host:     "0.0.0.0",
	DevMode:  true,
}


// ContactContract defines the contract for contact management
var ContactContract = zbz.CoreContract[models.Contact]{
	BaseContract: zbz.BaseContract{
		Name:        "Contact",
		Description: "Contact Management - Store and manage contact information including names, email addresses, phone numbers, and physical addresses. Contacts can be associated with companies and forms.",
	},
	DatabaseContract: PrimaryDatabase,
}

// CompanyContract defines the contract for company management
var CompanyContract = zbz.CoreContract[models.Company]{
	BaseContract: zbz.BaseContract{
		Name:        "Company",
		Description: "Company Management - Manage company profiles and organizational data. Companies serve as containers for contacts and can be associated with multiple forms and business processes.",
	},
	DatabaseContract: PrimaryDatabase,
}

// FormContract defines the contract for form management
var FormContract = zbz.CoreContract[models.Form]{
	BaseContract: zbz.BaseContract{
		Name:        "Form",
		Description: "Form Builder & Management - Create, configure, and manage dynamic forms with custom fields. Forms are the core building blocks for data collection and can contain multiple field types.",
	},
	DatabaseContract: PrimaryDatabase,
}

// PropertyContract defines the contract for property management
var PropertyContract = zbz.CoreContract[models.Property]{
	BaseContract: zbz.BaseContract{
		Name:        "Property",
		Description: "Property Value Storage - Store dynamic property values for form submissions. Properties link fields to their actual data values, enabling flexible data storage for any form configuration.",
	},
	DatabaseContract: PrimaryDatabase,
}

// FieldContract defines the contract for field management
var FieldContract = zbz.CoreContract[models.Field]{
	BaseContract: zbz.BaseContract{
		Name:        "Field",
		Description: "Field Definition & Types - Define form field schemas including field types, validation rules, and display properties. Fields serve as templates that define the structure of form data.",
	},
	DatabaseContract: PrimaryDatabase,
}