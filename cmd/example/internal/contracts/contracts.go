package contracts

import (
	"zbz/cmd/example/internal/models"
	"zbz/lib"
)

// PrimaryDatabase is the shared database contract that all cores can reference
var PrimaryDatabase = zbz.DatabaseContract{
	BaseContract: zbz.BaseContract{
		Name:        "primary",
		Description: "Primary application database for all core business data",
	},
	Key:    "primary",
	Driver: "postgres",
	DSN:    zbz.GetConfig().DSN(), // Use config DSN until we implement encrypted config contracts
}

// UserContract defines the contract for built-in user management
var UserContract = zbz.CoreContract[zbz.User]{
	BaseContract: zbz.BaseContract{
		Name:        "User",
		Description: "Built-in user management for authentication and profile operations",
	},
	DatabaseContract: PrimaryDatabase,
	Handlers: []string{"Get", "Update"}, // Only allow read and update operations
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