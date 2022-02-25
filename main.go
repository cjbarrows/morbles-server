package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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

var levels = []level{}

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

func getPlayerByID(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))

		if err == nil {
			rows, err := db.Query("SELECT id, name, admin FROM players WHERE id = $1", id)
			if err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error reading player]: %q", err))
				return
			}

			defer rows.Close()
			ok := rows.Next()

			if ok {
				var thisPlayer player
				if err := rows.Scan(&thisPlayer.ID, &thisPlayer.Name, &thisPlayer.Admin); err != nil {
					c.String(http.StatusInternalServerError,
						fmt.Sprintf("Error scanning player: %q", err))
					return
				}
				if err := thisPlayer.refreshWithLevels(db, levels); err != nil {
					c.String(http.StatusInternalServerError,
						fmt.Sprintf("Error reading level status: %q", err))
					return
				}
				c.JSON(http.StatusOK, thisPlayer)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
	}
}

func getNextPlayerId(db *sql.DB) (uint16, error) {
	rows, err := db.Query("SELECT MAX(id) FROM players")
	if err != nil {
		return 0, err
	}

	defer rows.Close()
	ok := rows.Next()

	if ok {
		var maxId uint16
		if err := rows.Scan(&maxId); err != nil {
			return 0, err
		}
		return maxId + 1, nil
	}
	return 0, errors.New("error getting next player id")
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

func addPlayer(db *sql.DB, name string) error {
	if nextid, err := getNextPlayerId(db); err == nil {
		if _, err := db.Exec("INSERT INTO players(id, name, admin) VALUES ($1, $2, false)", nextid, name); err != nil {
			log.Println("Error inserting new player ", err)
		} else {
			return err
		}
	} else {
		return err
	}
	return nil
}

func populateLevels(db *sql.DB) {
	var newLevels []level

	rows, err := db.Query("SELECT id, name, hint, rows, columns, starting_balls, ending_balls, map from levels")
	if err != nil {
		fmt.Println("Error loading levels ", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var id uint16
		var name string
		var hint string
		var numRows uint8
		var numColumns uint8
		var startingBalls string
		var endingBalls string
		var mapData string

		if err := rows.Scan(&id, &name, &hint, &numRows, &numColumns, &startingBalls, &endingBalls, &mapData); err != nil {
			fmt.Println("Error reading level ", err)
			return
		}
		newLevels = append(newLevels, level{id, name, hint, numRows, numColumns, startingBalls, endingBalls, mapData})
	}

	levels = newLevels
}

func (pl *player) refreshWithLevels(db *sql.DB, levels []level) error {
	rows, err := db.Query("SELECT level_id, attempts, failures, completed FROM level_status WHERE player_id = $1", pl.ID)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var levelId uint16
		var attempts uint16
		var failures uint16
		var completed bool
		if err := rows.Scan(&levelId, &attempts, &failures, &completed); err != nil {
			return err
		}
		pl.LevelStatuses = append(pl.LevelStatuses, levelStatus{levelId, attempts, failures, completed})
	}

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

	return nil
}

func getAuthenticatedPlayer(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, ok := getUser(c); ok {
			rows, err := db.Query("SELECT id, name, admin FROM players WHERE name = $1", user)
			if err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error reading player]: %q", err))
				return
			}

			defer rows.Close()
			ok := rows.Next()

			if ok {
				var thisPlayer player
				if err := rows.Scan(&thisPlayer.ID, &thisPlayer.Name, &thisPlayer.Admin); err != nil {
					c.String(http.StatusInternalServerError,
						fmt.Sprintf("Error scanning player: %q", err))
					return
				}
				if err := thisPlayer.refreshWithLevels(db, levels); err != nil {
					c.String(http.StatusInternalServerError,
						fmt.Sprintf("Error reading level status: %q", err))
					return
				}
				c.JSON(http.StatusOK, thisPlayer)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
	}
}

func putPlayerById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var updatedPlayer player
		if err := c.BindJSON(&updatedPlayer); err == nil {
			for _, playerLevel := range updatedPlayer.LevelStatuses {
				result, err := db.Exec("UPDATE level_status SET attempts = $1, failures = $2, completed = $3 WHERE player_id = $4 AND level_id = $5",
					playerLevel.Attempts, playerLevel.Failures, playerLevel.Completed, updatedPlayer.ID, playerLevel.LevelID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error updating player %s", err)})
					return
				}
				if count, err := result.RowsAffected(); err == nil && count == 0 {
					_, err := db.Exec("INSERT INTO level_status(player_id, level_id, attempts, failures, completed) VALUES ($1, $2, $3, $4, $5)",
						updatedPlayer.ID, playerLevel.LevelID, playerLevel.Attempts, playerLevel.Failures, playerLevel.Completed)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error inserting player level status %s", err)})
						return
					}
				}
			}
			fmt.Println("updated player", updatedPlayer)
			c.JSON(http.StatusOK, updatedPlayer)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"message": "player not found"})
	}
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

func login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		username := c.PostForm("username")
		password := c.PostForm("password")

		if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
			return
		}

		if password == "2022" {
			rows, err := db.Query("SELECT id FROM players WHERE name = $1", username)
			if err == nil {
				defer rows.Close()
				ok := rows.Next()

				if ok {
					var thisPlayer player
					if err := rows.Scan(&thisPlayer.ID); err != nil {
						log.Println("Error scanning player ", err)
						c.String(http.StatusInternalServerError,
							fmt.Sprintf("Error scanning player: %q", err))
						return
					}

					session.Set("user", username)
					if err := session.Save(); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
						return
					}
					c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
					return
				} else {
					if err := addPlayer(db, username); err == nil {
						session.Set("user", username)
						if err := session.Save(); err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
							return
						}
						c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
						return
					}
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading player"})
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
	}
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
	// session.Delete("user")
	session.Set("user", "")
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

func setMe(c *gin.Context) {
	session := sessions.Default(c)
	session.Set("user", "a")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully set me"})
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
	databaseUrl := os.Getenv("DATABASE_URL")

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		fmt.Println("Error opening database: ", err)
	}

	populateLevels(db)

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200", "https://morbles.herokuapp.com"}
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT"}
	router.Use(cors.New(config))

	store := cookie.NewStore([]byte("secret"))
	if domain := os.Getenv("CLIENT_DOMAIN"); domain != "" {
		log.Println("Domain is " + domain)
		store.Options(sessions.Options{Domain: domain})
	}
	router.Use(sessions.Sessions("thesession", store))

	router.POST("/login", login(db))

	private := router.Group("/api")
	private.Use(AuthRequired)
	{
		private.GET("/logout", logout)

		private.GET("/me", me)
		private.GET("/setme", setMe)
		private.GET("/status", status)

		private.GET("/levels", getLevels)
		private.GET("/levels/ids", getLevelsIDs)
		private.GET("/levels/:id", getLevelByID)
		private.POST("/levels", postLevel)
		private.PUT("/levels/:id", putLevelByID)

		private.GET("/player", getAuthenticatedPlayer(db))
		private.GET("/player/:id", getPlayerByID(db))
		private.PUT("/player/:id", putPlayerById(db))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router.Run("0.0.0.0:" + port)
}
