package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// User Model
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com"},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
	{ID: 3, Name: "Alice Johnson", Email: "alice@example.com"},
}

func main() {
	//Create Gin Router
	r := gin.Default()
	//simple route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	//get all users
	r.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, users)
	})

	//get user by id
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		for _, u := range users {
			if id == string(rune(u.ID+'0')) {
				c.JSON(http.StatusOK, u)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	})

	//Create a user
	r.POST("/users", func(c *gin.Context) {
		var newUser User
		if err := c.ShouldBindJSON(&newUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newUser.ID = len(users) + 1
		users = append(users, newUser)
		c.JSON(http.StatusCreated, newUser)
	})

	//Update a user
	r.PUT("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		for i, u := range users {
			if id == string(rune(u.ID+'0')) {
				var updatedUser User
				if err := c.ShouldBindJSON(&updatedUser); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				users[i] = updatedUser
				c.JSON(http.StatusOK, updatedUser)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	})

	//Delete a user
	r.DELETE("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		for i, u := range users {
			if id == string(rune(u.ID+'0')) {
				users = append(users[:i], users[i+1:]...)
				c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	})

	r.Run(":8080")
}
