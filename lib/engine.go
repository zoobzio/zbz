package zbz

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Engine struct {
	R *gin.Engine
	A *Authenticator
	V *validator.Validate
	D *gorm.DB
}

func NewEngine() *Engine {
	// build a router
	r := gin.Default()

	// build an authenticator
	a, err := NewAuthenticator()
	if err != nil {
		log.Fatal(err)
	}

	// build a validator
	v := validator.New()

	// connect to the database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"), os.Getenv("POSTGRES_PORT"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// build the engine
	return &Engine{
		R: r,
		A: a,
		V: v,
		D: db,
	}
}

// Start the engine
func (e *Engine) Start() {
	e.R.Run(fmt.Sprintf(":%s", os.Getenv("API_PORT")))
}
