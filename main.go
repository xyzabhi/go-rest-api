package main

import (
	"net/http"

	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	//Create Gin Router
	r := gin.Default()

	//Connect to DB
	db := ConnectDB()
	//Always close db connections when app exits
	defer db.Close()
	//simple route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	//get all users from DB
	r.GET("/users", func(c *gin.Context) {
		//Run select query
		rows, err := db.Query(c, "SELECT id,name,email,created_at,updated_at FROM users ORDER BY id ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		//Always close rows when done
		defer rows.Close()
		//Build response slice
		var users []map[string]any
		for rows.Next() {
			var id int
			var name, email string
			var createdAt, updatedAt time.Time
			//Scan rows into go vars
			err := rows.Scan(&id, &name, &email, &createdAt, &updatedAt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			//Append users to response slice
			users = append(users, map[string]any{
				"id": id, "name": name, "email": email, "created_at": createdAt, "updated_at": updatedAt})
		}
		c.JSON(http.StatusOK, users)
	})

	//get user by id
	r.GET("/users/:id", func(c *gin.Context) {

	})

	//Create a user
	r.POST("/users", func(c *gin.Context) {
		//Struct input payload
		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		//Bind the JSON to struct
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Insert user into DB, return the new ID
		var id int
		err := db.QueryRow(c, "INSERT INTO users (name,email) VALUES ($1,$2) RETURNING id", input.Name, input.Email).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		//Return the new user with the ID
		c.JSON(http.StatusCreated, gin.H{"id": id, "name": input.Name, "email": input.Email})
	})

	//Update a user
	r.PUT("/users/:id", func(c *gin.Context) {

	})

	//Delete a user
	r.DELETE("/users/:id", func(c *gin.Context) {

	})

	r.Run(":8080")
}

// connectDB establishes a connection to the PostgreSQL database
func ConnectDB() *pgxpool.Pool {
	//DB connection string from eniv
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://app:app@localhost:15432/appdb?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	//test the connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database")
	return pool
}
