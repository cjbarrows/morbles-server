package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
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
	Failures  uint16
	Completed bool
}

type player struct {
	ID            uint16
	Name          string
	Admin         bool
	LevelStatuses []levelStatus
}

var levels = []level{
	{ID: 1, Name: "Nothing Doin'", Hint: "Ball drop.", Rows: 2, Columns: 1, StartingBalls: "R", EndingBalls: "R", MapData: " " + " "},
	{ID: 2, Name: "Two Wide", Hint: "Pick a chute.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "  "},
	{ID: 3, Name: "Bumper OK", Hint: "Bumpers bump one left or one right.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "R "},
	{ID: 4, Name: "No Left Turn", Hint: "If a ball goes out of bounds, it's lost forever.", Rows: 2, Columns: 2, StartingBalls: "RG", EndingBalls: "RG", MapData: "  " + "L "},
}

var mockLevelStatuses = []levelStatus{
	{LevelID: 1, Attempts: 1, Failures: 0, Completed: true},
	{LevelID: 2, Attempts: 0, Failures: 0, Completed: false},
	{LevelID: 3, Attempts: 0, Failures: 0, Completed: false},
	{LevelID: 4, Attempts: 2, Failures: 2, Completed: false},
}

var players = []player{
	{ID: 1, Name: "Test Player", Admin: false, LevelStatuses: mockLevelStatuses},
}

func getLevelIDs(theLevels []level) []uint16 {
	var ids []uint16

	for _, level := range theLevels {
		ids = append(ids, level.ID)
	}

	return ids
}

func getBlankLevels() []levelStatus {
	var blankLevels []levelStatus

	for _, level := range levels {
		ls := levelStatus{level.ID, 0, 0, false}
		blankLevels = append(blankLevels, ls)
	}

	return blankLevels
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

func getNextPlayerId(slice []player) uint16 {
	var nextid uint16 = 0
	for _, item := range slice {
		if item.ID > nextid {
			nextid = item.ID
		}
	}
	return nextid + 1
}

func getNextLevelId(slice []level) uint16 {
	var nextid uint16 = 0
	for _, item := range slice {
		if item.ID > nextid {
			nextid = item.ID
		}
	}
	return nextid + 1
}

func addPlayer(name string) {
	nextid := getNextPlayerId(players)
	blankLevels := getBlankLevels()
	newPlayer := player{nextid, name, name == "charlie.barrows@gmail.com", blankLevels}
	players = append(players, newPlayer)
}

func (pl *player) refreshWithLevels(levels []level) {
	for _, lv := range levels {
		found := false
		for _, lvs := range pl.LevelStatuses {
			if lv.ID == lvs.LevelID {
				found = true
			}
		}
		if !found {
			pl.LevelStatuses = append(pl.LevelStatuses, levelStatus{lv.ID, 0, 0, false})
		}
	}
}

func getAuthenticatedPlayer(c *gin.Context) {
	if user, ok := getUser(c); ok {
		for _, pl := range players {
			if pl.Name == user {
				pl.refreshWithLevels(levels)
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

func postLevel(c *gin.Context) {
	var newLevel level
	if err := c.BindJSON(&newLevel); err == nil {
		newLevel.ID = getNextLevelId(levels)
		fmt.Println("new level", newLevel)
		levels = append(levels, newLevel)
		c.JSON(http.StatusOK, newLevel)
		return
	} else {
		// BindJSON already sets a BadRequest
		return
	}
}

func putLevelByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err == nil {
		for key, lvl := range levels {
			if lvl.ID == uint16(id) {
				var updatedLevel level
				if err := c.BindJSON(&updatedLevel); err == nil {
					fmt.Println("updated level", updatedLevel)
					levels[key] = updatedLevel
					c.JSON(http.StatusOK, updatedLevel)
					return
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error updating level %s", err)})
					return
				}
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "level not found"})
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
			session.Set("user", username)
			if err := session.Save(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
			return
		}
	}

	if password == "2020" {
		addPlayer(username)
		session.Set("user", username)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
}

func getUser(c *gin.Context) (string, bool) {
	session := sessions.Default(c)
	if user := session.Get("user"); user != nil {
		return user.(string), true
	}
	return "", false
}

func logout(c *gin.Context) {
	if _, ok := getUser(c); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session := sessions.Default(c)
	session.Delete("user")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func me(c *gin.Context) {
	if user, ok := getUser(c); ok {
		c.JSON(http.StatusOK, gin.H{"user": user})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user"})
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}

func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}

func main() {
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200", "https://morbles.herokuapp.com"}
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT"}
	router.Use(cors.New(config))

	store := cookie.NewStore([]byte("secret"))
	if domain := os.Getenv("CLIENT_DOMAIN"); domain != "" {
		fmt.PrintLn("Domain is " + domain)
		store.Options(sessions.Options{Domain: domain})
	}
	router.Use(sessions.Sessions("thesession", store))

	router.POST("/login", login)

	private := router.Group("/api")
	private.Use(AuthRequired)
	{
		private.GET("/logout", logout)

		private.GET("/me", me)
		private.GET("/status", status)

		private.GET("/levels", getLevels)
		private.GET("/levels/ids", getLevelsIDs)
		private.GET("/levels/:id", getLevelByID)
		private.POST("/levels", postLevel)
		private.PUT("/levels/:id", putLevelByID)

		private.GET("/player", getAuthenticatedPlayer)
		private.GET("/player/:id", getPlayerByID)
		private.PUT("/player/:id", putPlayerById)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router.Run("0.0.0.0:" + port)
}
