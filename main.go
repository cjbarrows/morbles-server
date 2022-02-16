package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	userkey = "user"
)

type level struct {
	ID            uint16
	Name          string
	Hint          string
	Rows          uint8
	Columns       uint8
	StartingBalls string
	EndingBalls   string
	MapData       string
}

type levelStatus struct {
	LevelID   uint16
	Attempts  uint16
	Completed bool
}

type player struct {
	ID            uint16
	Name          string
	LevelStatuses []levelStatus
}

var levels = []level{
	{ID: 1, Name: "Nothing Doin'", Hint: "Ball drop.", Rows: 2, Columns: 1, StartingBalls: "R", EndingBalls: "R", MapData: " " + " "},
	{ID: 2, Name: "Two Wide", Hint: "Pick a chute.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "  "},
	{ID: 3, Name: "Bumper OK", Hint: "Bumpers bump one left or one right.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "R "},
	{ID: 4, Name: "No Left Turn", Hint: "If a ball goes out of bounds, it's lost forever.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "L "},
}

var levelStatuses = []levelStatus{
	{LevelID: 1, Attempts: 1, Completed: true},
	{LevelID: 2, Attempts: 0, Completed: false},
	{LevelID: 3, Attempts: 0, Completed: false},
	{LevelID: 4, Attempts: 2, Completed: false},
}

var players = []player{
	{ID: 1, Name: "Test Player", LevelStatuses: levelStatuses},
}

func getLevelIDs(theLevels []level) []uint16 {
	var ids []uint16

	for _, level := range theLevels {
		ids = append(ids, level.ID)
	}

	return ids
}

func getLevelsIDs(c *gin.Context) {
	ids := getLevelIDs(levels)

	c.JSON(http.StatusOK, ids)
}

func getLevelByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err == nil {
		for _, level := range levels {
			if level.ID == uint16(id) {
				c.IndentedJSON(http.StatusOK, level)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "level not found"})
}

func getLevels(c *gin.Context) {
	c.JSON(http.StatusOK, levels)
}

func getPlayerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err == nil {
		for _, pl := range players {
			if pl.ID == uint16(id) {
				c.JSON(http.StatusOK, pl)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
}

func getAuthenticatedPlayer(c *gin.Context) {
	if user, ok := getUser(c); ok {
		for _, pl := range players {
			if pl.Name == user {
				c.JSON(http.StatusOK, pl)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
}

func putPlayerById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err == nil {
		for key, pl := range players {
			if pl.ID == uint16(id) {
				var updatedPlayer player
				if err := c.BindJSON(&updatedPlayer); err == nil {
					fmt.Println("updated player", updatedPlayer)
					players[key] = updatedPlayer
					c.JSON(http.StatusOK, updatedPlayer)
					return
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error updating player %s", err)})
					return
				}
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
}

func login(c *gin.Context) {
	session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")

	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	for _, pl := range players {
		if username == pl.Name && password == "2020" {
			session.Set(userkey, username) // In real world usage you'd set this to the users ID
			if err := session.Save(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
			return
		}
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
}

func getUser(c *gin.Context) (string, bool) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	return user.(string), true
}

func logout(c *gin.Context) {
	if _, ok := getUser(c); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session := sessions.Default(c)
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func me(c *gin.Context) {
	user, _ := getUser(c)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}

func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}

func main() {
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte("secret"))))

	router.POST("/login", login)
	router.GET("/logout", logout)

	private := router.Group("/api")
	private.Use(AuthRequired)
	{
		private.GET("/me", me)
		private.GET("/status", status)

		private.GET("/levels", getLevels)
		private.GET("/levels/ids", getLevelsIDs)
		private.GET("/levels/:id", getLevelByID)

		private.GET("/player", getAuthenticatedPlayer)
		private.GET("/player/:id", getPlayerByID)
		private.PUT("/player/:id", putPlayerById)
	}

	router.Run("localhost:8080")
}
