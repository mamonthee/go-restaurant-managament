package controllers

import (
	"context"
	"go-restaurant-managament/database"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"net/http"

	"go-restaurant-managament/models"

	"fmt"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		result, err := tableCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing tables"})
			return
		}
		var allTables []bson.M
		if err := result.All(ctx, &allTables); err != nil {
			log.Fatal(err)
		}
		// c.JSON(http.StatusOK, allTables)
		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Table items fetched successfully",
			"data":    allTables,
		})
	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		tableId := c.Param("table_id")
		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(table)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while fetching table"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": table})

	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			fmt.Println("BIndJSON")
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		table.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()

		validationErr := validate.Struct(table)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": validationErr})
			return
		}
		result, err := tableCollection.InsertOne(ctx, table)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("error:table was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": result})
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("UpdateTable")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		tableId := c.Param("table_id")
		fmt.Println("TableId", tableId)
		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fmt.Println("tablenum", table.Table_number)
		fmt.Println("noofguest", table.Number_of_guests)
		var updateObj primitive.D
		if table.Number_of_guests != nil {
			updateObj = append(updateObj, bson.E{"number_of_guests", table.Number_of_guests})
		}

		if table.Table_number != nil {
			updateObj = append(updateObj, bson.E{"table_number", table.Table_number})
		}
		updateObj = append(updateObj, bson.E{"status", table.Status})
		updateObj = append(updateObj, bson.E{"availiable", table.Availiable})
		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", table.Updated_at})
		filter := bson.M{"table_id": tableId}
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}
		result, err := tableCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("error : table update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
