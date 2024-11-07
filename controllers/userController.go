package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-restaurant-managament/database"
	"go-restaurant-managament/helpers"
	"go-restaurant-managament/models"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Simplified pagination handling
		recordPerPage, err := strconv.Atoi(c.DefaultQuery("recordPerPage", "10"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}
		// startIndex := (page - 1) * recordPerPage

		// Simplified pipeline for inspection
		matchStage := bson.D{{"$match", bson.D{}}}
		projectStage := bson.D{{"$project", bson.D{{"_id", 0}}}} // Show entire document

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{matchStage, projectStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while aggregating user items", "details": err.Error()})
			return
		}

		var allUsers []bson.M
		if err := result.All(ctx, &allUsers); err != nil {
			log.Fatal(err)
		}

		// Output full structure of first document for inspection
		if len(allUsers) > 0 {
			fmt.Printf("First user document: %+v\n", allUsers[0])
		} else {
			fmt.Println("No documents found.")
			c.JSON(http.StatusNotFound, gin.H{"message": "No users found"})
			return
		}

		c.JSON(http.StatusOK, allUsers)
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		userId := c.Param("user_id")

		var user models.User

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		fmt.Println("Signup")
		err := c.BindJSON(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			fmt.Println("ERROR", err.Error())
			return
		}
		fmt.Println("Signup", user)
		validationErr := validate.Struct(&user)
		if validationErr != nil {
			fmt.Println("EE")
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}
		fmt.Println("after validate")
		count, err := userCollection.CountDocuments(ctx, bson.M{"user_id": user.User_id})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
		}
		fmt.Println("after count")
		password := HashPassword(*user.Password)
		user.Password = &password
		fmt.Println("after hasPassword")

		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the phone numer"})
			return
		}
		fmt.Println("after countdoc")
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email or phone number already exists"})
			return
		}
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		fmt.Println("after userId", *user.User_role)
		token, refreshToken, _ := helpers.GenerateAllTokens(*user.Email, *user.Name, user.User_id, *user.User_role)
		user.Token = &token
		user.Refresh_Token = &refreshToken

		resultInsertionNumber, err := userCollection.InsertOne(ctx, user)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("User item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}
func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
		var user models.User
		userId := c.Param("user_id")
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenRegenerationNeeded := false
		var updateObj primitive.D
		if user.Name != nil {
			tokenRegenerationNeeded = true
			updateObj = append(updateObj, bson.E{"name", user.Name})
		}
		if user.Email != nil {
			tokenRegenerationNeeded = true
			updateObj = append(updateObj, bson.E{"email", user.Email})
		}
		if user.User_role != nil {
			tokenRegenerationNeeded = true
			updateObj = append(updateObj, bson.E{"user_role", user.User_role})
		}
		if user.Phone != nil {
			updateObj = append(updateObj, bson.E{"phone", user.Phone})
		}
		if user.Password != nil {
			password := HashPassword(*user.Password)
			user.Password = &password
			updateObj = append(updateObj, bson.E{"password", user.Password})
		}
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", user.Updated_at})

		if tokenRegenerationNeeded {
			token, refreshToken, _ := helpers.GenerateAllTokens(*user.Email, *user.Name, user.User_id, *user.User_role)
			user.Token = &token
			user.Refresh_Token = &refreshToken
		}

		filter := bson.M{"user_id": userId}
		upsert := true
		opts := options.UpdateOptions{
			Upsert: &upsert,
		}
		result, err := userCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", updateObj}},
			&opts,
		)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("error : user update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		var foundUser models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email is incorrect"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		defer cancel()
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if foundUser.Email == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}
		token, refreshToken, _ := helpers.GenerateAllTokens(*foundUser.Email, *foundUser.Name, foundUser.User_id, *foundUser.User_role)
		helpers.UpdateAllTokens(token, refreshToken, foundUser.User_id)
		c.JSON(http.StatusOK, foundUser)
	}
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPassword(userPassword string, providePassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providePassword), []byte(userPassword))
	check := true
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("email or password is incorrect")
		check = false
	}
	return check, msg
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}
var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

func HandleWebSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println("Error during connection upgrade:", err)
			return
		}
		defer conn.Close()

		// Register the new client
		mu.Lock()
		clients[conn] = true
		mu.Unlock()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				mu.Lock()
				delete(clients, conn)
				mu.Unlock()
				break
			}
		}
	}
}

// func notifyClients(orderId string) {
// 	fmt.Println("Notifying clients with orderId:", orderId) // Log order ID to confirm it's correct
// 	mu.Lock()
// 	defer mu.Unlock()
// 	for client := range clients {
// 		message := fmt.Sprintf(`{"event":"newOrder","payload":"%s"}`, orderId)
// 		err := client.WriteMessage(websocket.TextMessage, []byte(message))
// 		if err != nil {
// 			fmt.Println("Error writing message:", err) // Log any errors
// 			client.Close()
// 			delete(clients, client)
// 		}
// 	}
// }
// func notifyWaiter(orderData models.Order) {
// 	fmt.Println("Notifying waiter with orderData:", orderData) // Log order ID to confirm it's correct
// 	mu.Lock()
// 	defer mu.Unlock()
// 	for client := range clients {
// 		message := fmt.Sprintf(`{"event":"prepareStatus","payload":"%s"}`, orderData)
// 		err := client.WriteMessage(websocket.TextMessage, []byte(message))
// 		if err != nil {
// 			fmt.Println("Error writing message:", err) // Log any errors
// 			client.Close()
// 			delete(clients, client)
// 		}
// 	}
// }

// Define a message structure to handle different event types
type Message struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// notifyClients sends a "newOrder" event with the specified orderId
func notifyClients(order models.Order) {
	fmt.Println("Notifying clients with orderId:", order)
	mu.Lock()
	defer mu.Unlock()

	message := Message{
		Event:   "newOrder",
		Payload: order,
	}
	sendMessageToAllClients(message)
}

// notifyWaiter sends a "prepareStatus" event with the specified order data
func notifyWaiter(orderData models.Order) {
	fmt.Println("Notifying waiter with orderData:", orderData.Table_number)
	mu.Lock()
	defer mu.Unlock()

	message := Message{
		Event:   "prepareStatus",
		Payload: orderData,
	}
	sendMessageToAllClients(message)
}

// Helper function to send a JSON message to all clients
func sendMessageToAllClients(message Message) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error marshaling message:", err)
		return
	}

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			fmt.Println("Error writing message:", err)
			client.Close()
			delete(clients, client)
		}
	}
}
