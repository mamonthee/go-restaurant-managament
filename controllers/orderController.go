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

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing order items"})
		}
		var allOrders []bson.M
		if err := result.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		orderId := c.Param("order_id")
		var order models.Order

		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing order items"})
			return
		}
		c.JSON(http.StatusOK, order)
	}
}

func CreatedOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var order models.Order
		var table models.Table

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(&order)
		if validationErr != nil {
			fmt.Println("validationError")
			c.JSON(http.StatusBadGateway, gin.H{"error": validationErr.Error()})
			return
		}
		fmt.Println("tableId", *order.Table_id)
		tableId := order.Table_id
		if order.Table_id != nil {
			if err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(&table); err != nil {
				msg := fmt.Sprintf("message: table was not found")
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
		}
		created_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Created_at = created_at
		order.Updated_at = updated_at

		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		result, err := orderCollection.InsertOne(ctx, order)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("message: order item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, result)

	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var table models.Table
		var order models.Order
		orderId := c.Param("order_id")
		fmt.Println("OrderId", orderId)
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var updateObj primitive.D

		if order.Order_id != "" {
			updateObj = append(updateObj, bson.E{"order_id", order.Order_id})
		}
		if order.Prepare_Status != "" {
			updateObj = append(updateObj, bson.E{"prepare_status", order.Prepare_Status})
		}
		if order.Table_id != nil {
			err := orderCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			defer cancel()
			if err != nil {
				msg := fmt.Sprintf("message:table was not found")
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
		}
		updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", updated_at})

		upsert := true
		filter := bson.M{"order_id": orderId}

		opts := options.UpdateOptions{
			Upsert: &upsert,
		}
		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opts,
		)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("order time update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		notifyWaiter(order)
		c.JSON(http.StatusOK, result)

	}
}
func OrderItemOrderCreator(order models.Order) string {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	// order.Total_amount = order.Total_amount
	// order.Total_quantity = order.Total_quantity
	order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()
	order.Status = "CREATED"
	order.Created_by = order.Created_by
	fmt.Println("USERID", order.User_id)

	orderCollection.InsertOne(ctx, order)
	defer cancel()
	return order.Order_id
}

// UpdateOrderStatus updates the status of an order based on the order ID.
func UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	// Define filter and update object
	filter := bson.M{"order_id": orderID}
	update := bson.D{
		{"$set", bson.D{
			{"status", status},
			{"updated_at", time.Now()},
		}},
	}

	// Update the order in the database
	_, err := orderCollection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(false))
	if err != nil {
		fmt.Printf("failed to update order status: %v\n", err)
		return err
	}
	return nil
}

type OrderWithItems struct {
	Order      models.Order       `bson:",inline"` // Embed Order struct
	OrderItems []models.OrderItem `json:"order_items"`
}

func GetAllOrdersWithItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var allOrders []models.Order
		var ordersWithItems []OrderWithItems

		// Fetch all orders
		cursor, err := orderCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}

		for _, order := range allOrders {
			// var order models.Order
			// if err := cursor.Decode(&order); err != nil {
			// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			// 	return
			// }
			fmt.Println("CTX", order)

			// Fetch associated order items for each order
			var orderItems []models.OrderItem
			orderItemCursor, err := orderItemCollection.Find(ctx, bson.M{"order_id": order.Order_id})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			for orderItemCursor.Next(ctx) {
				var orderItem models.OrderItem
				if err := orderItemCursor.Decode(&orderItem); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				orderItems = append(orderItems, orderItem)
			}
			orderItemCursor.Close(ctx)

			// Create a combined struct
			ordersWithItems = append(ordersWithItems, OrderWithItems{
				Order:      order,
				OrderItems: orderItems,
			})
		}

		if err := cursor.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ordersWithItems)
	}
}

type OrderWithItemsAndFood struct {
	Order_id     string              `json:"order_id"`
	Table_number *string             `json:"table_number"`
	OrderItems   []OrderItemWithFood `json:"order_items"`
}
type OrderItemWithFood struct {
	Food_id    *string  `json:"food_id"`
	Food_name  *string  `json:"food_name"`
	Quantity   *uint64  `json:"quantity"`
	Unit_price *float64 `json:"unit_price"`
}

func GetAllOrdersWithItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Match stage
		match := bson.D{{"$match", bson.D{{"prepare_status", bson.D{{"$ne", "Ready"}}}}}}

		// Lookup to join Order with OrderItems
		lookupOrderItems := bson.D{{"$lookup", bson.D{
			{"from", "orderItem"},
			{"localField", "order_id"},
			{"foreignField", "order_id"},
			{"as", "order_items"},
		}}}

		// Unwind order_items to prepare for further joining
		unwindOrderItems := bson.D{{"$unwind", bson.D{
			{"path", "$order_items"},
			{"preserveNullAndEmptyArrays", true},
		}}}

		// Lookup to join OrderItems with Food collection
		lookupFood := bson.D{{"$lookup", bson.D{
			{"from", "food"},
			{"localField", "order_items.food_id"},
			{"foreignField", "food_id"},
			{"as", "food_details"},
		}}}

		// Unwind food_details to get individual food item data
		unwindFoodDetails := bson.D{{"$unwind", bson.D{
			{"path", "$food_details"},
			{"preserveNullAndEmptyArrays", true},
		}}}

		// Additional lookup to join Order with Table to get table_number
		lookupTable := bson.D{{"$lookup", bson.D{
			{"from", "table"},
			{"localField", "table_id"},
			{"foreignField", "table_id"},
			{"as", "table_details"},
		}}}

		// Unwind table_details to access table_number
		unwindTableDetails := bson.D{{"$unwind", bson.D{
			{"path", "$table_details"},
			{"preserveNullAndEmptyArrays", true},
		}}}

		// Group stage to aggregate order items and format response
		group := bson.D{{"$group", bson.D{
			{"_id", bson.D{
				{"order_id", "$order_id"},
				{"order_date", "$order_date"},
				{"user_id", "$user_id"},
				{"created_by", "$created_by"},
				{"created_at", "$created_at"},
				{"updated_at", "$updated_at"},
				{"table_id", "$table_id"},
				{"total_amount", "$total_amount"},
				{"total_quantity", "$total_quantity"},
				{"status", "$status"},
				{"table_number", "$table_details.table_number"},
			}},
			{"order_items", bson.D{{"$push", bson.D{
				{"ID", "$order_items.ID"},
				{"order_item_id", "$order_items.order_item_id"},
				{"order_id", "$order_items.order_id"},
				{"quantity", "$order_items.quantity"},
				{"unit_price", "$order_items.unit_price"},
				{"created_at", "$order_items.created_at"},
				{"updated_at", "$order_items.updated_at"},
				{"food_id", "$order_items.food_id"},
				{"food_name", "$food_details.name"},
				{"status", "$order_items.status"},
			}}}},
		}}}

		// Project stage to finalize the desired structure
		project := bson.D{
			{"$project", bson.D{
				{"Order", bson.D{
					{"ID", "$_id.order_id"},
					{"order_date", "$_id.order_date"},
					{"user_id", "$_id.user_id"},
					{"created_by", "$_id.created_by"},
					{"created_at", "$_id.created_at"},
					{"updated_at", "$_id.updated_at"},
					{"order_id", "$_id.order_id"},
					{"table_id", "$_id.table_id"},
					{"table_number", "$_id.table_number"},
					{"total_amount", "$_id.total_amount"},
					{"total_quantity", "$_id.total_quantity"},
					{"status", "$_id.status"},
				}},
				{"order_items", "$order_items"},
			}},
		}

		// Execute aggregation pipeline
		cursor, err := orderCollection.Aggregate(
			ctx, mongo.Pipeline{
				match,
				lookupOrderItems,
				unwindOrderItems,
				lookupFood,
				unwindFoodDetails,
				lookupTable,
				unwindTableDetails,
				group,
				project,
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		// Decode result
		var ordersWithItems []bson.M
		if err := cursor.All(ctx, &ordersWithItems); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, ordersWithItems)
	}
}
