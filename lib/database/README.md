# Database Adapters

This directory contains database adapters for ZBZ. Adapters allow ZBZ to work with different databases while maintaining a consistent query interface.

## Creating a New Database Adapter

To create an adapter for a new database (e.g., MySQL, SQLite, MongoDB), follow these steps:

### 1. Implement the Database Interface

```go
package mydatabase

import (
    "github.com/jmoiron/sqlx"
    zbz "zbz/lib"
)

type MyDatabaseAdapter struct {
    db *sqlx.DB  // or your database client
    schema zbz.Schema
    
    // Contract metadata
    contractName string
    contractDescription string
}

func NewMyDatabaseAdapter(dsn string) zbz.Database {
    // Initialize your database connection
    db, err := sqlx.Connect("mysql", dsn) // or your driver
    if err != nil {
        panic(err)
    }
    
    return &MyDatabaseAdapter{
        db: db,
        schema: zbz.NewSchema(),
    }
}

func (d *MyDatabaseAdapter) Execute(contract *zbz.MacroContract, params any) (*sqlx.Rows, error) {
    // 1. Get the SQL template from the contract
    sqlTemplate := contract.GetSQL()
    
    // 2. Interpolate parameters (implement macro system)
    finalSQL := d.interpolateSQL(sqlTemplate, params)
    
    // 3. Execute query
    return d.db.Queryx(finalSQL)
}

func (d *MyDatabaseAdapter) CreateTableFromMeta(meta *zbz.Meta) error {
    // Generate CREATE TABLE SQL for your database dialect
    sql := d.generateCreateTableSQL(meta)
    
    _, err := d.db.Exec(sql)
    return err
}

func (d *MyDatabaseAdapter) GetSchema() zbz.Schema {
    return d.schema
}

func (d *MyDatabaseAdapter) ContractName() string {
    return d.contractName
}

func (d *MyDatabaseAdapter) ContractDescription() string {
    return d.contractDescription
}
```

### 2. Implement SQL Generation

Each database has different SQL dialects. Implement table creation for your database:

```go
func (d *MyDatabaseAdapter) generateCreateTableSQL(meta *zbz.Meta) string {
    var sql strings.Builder
    
    sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", 
        strings.ToLower(meta.Name)))
    
    columns := []string{}
    
    for _, field := range meta.FieldMetadata {
        if field.DatabaseColumnName == "-" {
            continue
        }
        
        // Convert Go types to your database types
        dbType := d.convertGoTypeToDBType(field.GoType)
        
        column := fmt.Sprintf("%s %s", field.DatabaseColumnName, dbType)
        
        if field.IsRequired {
            column += " NOT NULL"
        }
        
        columns = append(columns, column)
    }
    
    sql.WriteString(strings.Join(columns, ", "))
    sql.WriteString(", PRIMARY KEY (id)")
    sql.WriteString(")")
    
    return sql.String()
}

func (d *MyDatabaseAdapter) convertGoTypeToDBType(goType string) string {
    // Map Go types to your database types
    switch goType {
    case "string":
        return "VARCHAR(255)"  // MySQL
        // return "TEXT"       // PostgreSQL
        // return "TEXT"       // SQLite
    case "int", "int64":
        return "BIGINT"
    case "time.Time":
        return "DATETIME"      // MySQL
        // return "TIMESTAMP"  // PostgreSQL
    case "bool":
        return "BOOLEAN"
    default:
        return "TEXT"
    }
}
```

### 3. Implement Macro System

ZBZ uses SQL macros for query templates. Implement interpolation for your database:

```go
func (d *MyDatabaseAdapter) interpolateSQL(template string, params any) string {
    // This is a simplified example - implement proper parameter binding
    // for your database to prevent SQL injection
    
    paramMap, ok := params.(map[string]any)
    if !ok {
        return template
    }
    
    result := template
    for key, value := range paramMap {
        placeholder := fmt.Sprintf("{{%s}}", key)
        
        // Convert value to SQL-safe string
        sqlValue := d.toSQLValue(value)
        
        result = strings.ReplaceAll(result, placeholder, sqlValue)
    }
    
    return result
}

func (d *MyDatabaseAdapter) toSQLValue(value any) string {
    switch v := value.(type) {
    case string:
        // Escape quotes for your database
        return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
    case int, int64:
        return fmt.Sprintf("%d", v)
    case bool:
        if v {
            return "TRUE"
        }
        return "FALSE"
    default:
        return fmt.Sprintf("'%v'", v)
    }
}
```

### 4. Integration with ZBZ

```go
package main

import (
    "zbz/lib"
    mydatabase "zbz/lib/database/mydatabase"
)

func main() {
    engine := zbz.NewEngine()
    
    // Create database contract
    dbContract := zbz.DatabaseContract{
        BaseContract: zbz.BaseContract{
            Name: "Primary Database",
            Description: "MySQL database for application data",
        },
        DSN: "user:password@tcp(localhost:3306)/mydb",
    }
    
    // Create your database adapter
    database := mydatabase.NewMyDatabaseAdapter(dbContract.DSN)
    
    // Register with engine
    engine.RegisterDatabase("primary", database)
    
    // Rest of setup...
}
```

## Interface Requirements

Your adapter MUST implement:

- `Execute(contract, params)` - Execute SQL queries from macro contracts
- `CreateTableFromMeta(meta)` - Create tables from ZBZ model metadata
- `GetSchema()` - Return schema interface for validation
- `ContractName()` / `ContractDescription()` - Service metadata

## Database-Specific Considerations

### MySQL
- Use `github.com/go-sql-driver/mysql` driver
- Handle `AUTO_INCREMENT` for ID fields
- Use `DATETIME` for timestamps

### SQLite
- Use `github.com/mattn/go-sqlite3` driver
- Handle file-based database paths
- Use `INTEGER PRIMARY KEY` for auto-increment

### MongoDB
- Implement document-based operations
- Convert SQL-like operations to MongoDB queries
- Handle BSON document structure

### SQL Server
- Use `github.com/denisenkom/go-mssqldb` driver
- Handle `IDENTITY` columns
- Use `NVARCHAR` for Unicode support

## Security Notes

1. **SQL Injection**: Always use proper parameter binding
2. **Validation**: Validate table/column names against schema
3. **Escaping**: Properly escape values for your database dialect

## Testing

Test your adapter with:
- Model creation
- CRUD operations
- Query parameter interpolation
- Schema validation