package controllers

import (
	"context"
	"fmt"
	"go-restaurant-managament/database"
	"go-restaurant-managament/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")

func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		result, err := menuCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured whil listing the menu items"})
			return
		}
		var allMenus []bson.M
		if err = result.All(ctx, &allMenus); err != nil {
			log.Fatal(err)
		}
		// c.JSON(http.StatusOK, allMenus)
		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Menu items fetched successfully",
			"data":    allMenus,
		})
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		menuId := c.Param("menu_id")
		var menu models.Menu

		err := menuCollection.FindOne(ctx, bson.M{"menu_id": menuId}).Decode(&menu)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occured while fetching the menu"})
		}

		c.JSON(http.StatusOK, menu)

	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("Create menu")
		var menu models.Menu
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		if err := c.BindJSON(&menu); err != nil {
			fmt.Println("ErrBindingjson")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		validatorErr := validate.Struct(&menu)
		fmt.Println(menu.Category, "//", menu.Name)

		if validatorErr != nil {
			fmt.Println("Valtidation error", validatorErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": validatorErr})
			return
		}
		menu.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menu.ID = primitive.NewObjectID()
		menu.Menu_id = menu.ID.Hex()

		result, insertErr := menuCollection.InsertOne(ctx, menu)
		defer cancel()
		if insertErr != nil {
			fmt.Println("InsertError", insertErr)
			msg := fmt.Sprintf("Menu item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var menu models.Menu
		fmt.Println("UpdateMenu")
		if err := c.BindJSON(&menu); err != nil {
			fmt.Println("BindJson", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		menuId := c.Query("menu_id")
		fmt.Println("MENUID", menuId)
		filter := bson.M{"menu_id": menuId}
		fmt.Println("UpdateMenu00")
		var updateObj primitive.D
		fmt.Println("UpdateMenu-0")
		// if menu.Start_Date != nil && menu.End_Date != nil {
		fmt.Println("UpdateMenu1")
		// if !inTimeSpan(*menu.Start_Date, *menu.End_Date, time.Now()) {
		// 	msg := "kindly retype the time"
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		// 	defer cancel()
		// 	return
		// }
		fmt.Println("UpdateMenu2")
		updateObj = append(updateObj, bson.E{"start_date", menu.Start_Date})
		updateObj = append(updateObj, bson.E{"end_date", menu.End_Date})

		if menu.Name != "" {
			updateObj = append(updateObj, bson.E{"name", menu.Name})
		}
		fmt.Println("UpdateMenu3")
		if menu.Category != "" {
			updateObj = append(updateObj, bson.E{"category", menu.Category})
		}
		menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", menu.Updated_at})
		fmt.Println("UpdateMenu4")
		upset := true

		opt := options.UpdateOptions{
			Upsert: &upset,
		}
		fmt.Println("BeforeUpdate")
		result, err := menuCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)
		if err != nil {
			fmt.Println("ERROR")
			msg := "Menu update failed "
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()
		fmt.Println("Finish")
		c.JSON(http.StatusOK, result)
		// }
	}

}
func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}
