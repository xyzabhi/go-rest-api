package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User struct maps directly to the "users" table in Postgres.
// The JSON tags control how the struct is serialized/deserialized in API responses.
type User struct {
	ID        int       `json:"id"`         // primary key
	Name      string    `json:"name"`       // user name
	Email     string    `json:"email"`      // unique email
	CreatedAt time.Time `json:"created_at"` // timestamp when user was created
	UpdatedAt time.Time `json:"updated_at"` // timestamp when user was last updated
}

func main() {
	// Connect to Postgres using pgxpool (see db.go)
	db := ConnectDB()
	defer db.Close()

	// Create a Gin router with default middleware (logger + recovery)
	r := gin.Default()

	// Health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ------------------------------
	// GET /users -> list all users
	// ------------------------------
	r.GET("/users", func(c *gin.Context) {
		// --- Parse query params ---
		limit := 10
		offset := 0
		q := c.Query("q") // search term
		sortBy := c.DefaultQuery("sort", "id")
		order := c.DefaultQuery("order", "asc")

		// Validate limit (default 10, max 100)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		// Validate offset
		if o := c.Query("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		// Validate sortBy
		validSort := map[string]bool{"id": true, "name": true, "email": true}
		if !validSort[sortBy] {
			sortBy = "id"
		}

		// Validate order
		if order != "asc" && order != "desc" {
			order = "asc"
		}

		// --- Build query ---
		query := `
			SELECT id, name, email, created_at, updated_at
			FROM users
		`
		var args []any
		if q != "" {
			// Use ILIKE for case-insensitive search
			query += "WHERE name ILIKE $1 OR email ILIKE $1 "
			args = append(args, "%"+q+"%")
		}

		// ORDER BY + LIMIT/OFFSET
		query += fmt.Sprintf("ORDER BY %s %s LIMIT %d OFFSET %d", sortBy, strings.ToUpper(order), limit, offset)

		// --- Execute query ---
		rows, err := db.Query(c, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			users = append(users, u)
		}

		// --- Return response with metadata ---
		c.JSON(http.StatusOK, gin.H{
			"items":  users,
			"limit":  limit,
			"offset": offset,
			"sort":   sortBy,
			"order":  order,
			"query":  q,
		})
	})

	// --------------------------------
	// GET /users/:id -> get user by ID
	// --------------------------------
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id") // get id from URL path

		var u User
		// Query single user by ID
		err := db.QueryRow(c,
			"SELECT id, name, email, created_at, updated_at FROM users WHERE id=$1",
			id,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Respond with single user object
		c.JSON(http.StatusOK, u)
	})

	// -------------------------------
	// POST /users -> create new user
	// -------------------------------
	r.POST("/users", func(c *gin.Context) {
		// Input struct for request body
		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		// Bind JSON body into input struct
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Insert user into DB and return full user row
		var u User
		err := db.QueryRow(c,
			`INSERT INTO users (name, email)
			 VALUES ($1, $2)
			 RETURNING id, name, email, created_at, updated_at`,
			input.Name, input.Email,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Respond with the created user
		c.JSON(http.StatusCreated, u)
	})

	// ----------------------------------
	// PUT /users/:id -> update user info
	// ----------------------------------
	r.PUT("/users/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Input struct for update payload
		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		// Parse JSON request body
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update user and return updated row
		var u User
		err := db.QueryRow(c,
			`UPDATE users
			 SET name=$2, email=$3, updated_at=now()
			 WHERE id=$1
			 RETURNING id, name, email, created_at, updated_at`,
			id, input.Name, input.Email,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, u)
	})

	// ----------------------------------
	// DELETE /users/:id -> delete a user
	// ----------------------------------
	r.DELETE("/users/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Run DELETE query
		res, err := db.Exec(c, "DELETE FROM users WHERE id=$1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// If no row was deleted, user doesn’t exist
		if res.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Respond with confirmation
		c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
	})

	// Start server on port 8080
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
