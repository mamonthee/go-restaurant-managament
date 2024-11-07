package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-restaurant-managament/database"
	"go-restaurant-managament/models"
	"log"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderItemPack struct {
	Table_id       *string
	Table_number   *int
	Created_by     string
	User_id        *string
	Total_amount   float64
	Total_quantity int
	Order_items    []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("GETORDERITEMS")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		result, err := orderItemCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"error:": "error occurred while fetching order items"}})
			return
		}
		var allOrderItems []bson.M
		if err := result.All(ctx, &allOrderItems); err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("OrderId")
		orderId := c.Param("order_id")
		fmt.Println("OrderId", orderId)
		allOrderItems, err := ItemsByOrder(orderId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing order items by order id"})
			return
		}
		c.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) (OrderItem []primitive.M, err error) {
	fmt.Println("ItemsByOrder")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	matchStage := bson.D{{Key: "$match", Value: bson.D{{Key: "order_id", Value: id}}}}
	lookupStage := bson.D{{Key: "$lookup", Value: bson.D{{Key: "from", Value: "food"}, {Key: "localField", Value: "food_id"}, {Key: "foreignField", Value: "food_id"}, {Key: "as", Value: "food"}}}}
	unwindStage := bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$food"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}}

	lookupOrderStage := bson.D{{Key: "$lookup", Value: bson.D{{"from", "order"}, {Key: "localField", Value: "order_id"}, {"foreignField", "order_id"}, {"as", "order"}}}}
	unwindOrderStage := bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$order"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}}

	lookupTableStage := bson.D{{Key: "$lookup", Value: bson.D{{"from", "table"}, {"localField", "order.table_id"}, {"foreignField", "table_id"}, {"as", "table"}}}}
	unwindTableStage := bson.D{{"$unwind", bson.D{{"path", "$table"}, {"preserveNullAndEmptyArrays", true}}}}

	projectStage := bson.D{{"$project", bson.D{

		{"amount", bson.D{{"$multiply", bson.A{"$food.price", "$quantity"}}}},
		{"total_count", 1},
		{"food_name", "$food.name"},
		{"food_image", "$food.food_image"},
		{"table_number", "$table.table_number"},
		{"table_id", "$table.table_id"},
		{"order_id", "$order.order_id"},
		{"price", "$food.price"},
		{"quantity", 1},
	}}}

	groupStage := bson.D{{"$group",
		bson.D{
			{"_id", bson.D{
				{"order_id", "$order_id"},
				{"table_id", "$table_id"},
				{"table_number", "$table_number"}},
			},
			{"payment_due", bson.D{{"$sum", "$amount"}}},
			{"total_count", bson.D{{"$sum", 1}}},
			{"order_items", bson.D{{"$push", "$$ROOT"}}},
		}}}

	projectStage2 := bson.D{
		{"$project", bson.D{
			// {"id", 0},
			{"payment_due", 1},
			{"total_count", 1},
			{"table_number", "$_id.table_number"},
			{"order_items", 1},
		}},
	}

	var orderItems []primitive.M
	result, err := orderItemCollection.Aggregate(
		ctx, mongo.Pipeline{
			matchStage,
			lookupStage,
			unwindStage,
			lookupOrderStage,
			unwindOrderStage,
			lookupTableStage,
			unwindTableStage,
			projectStage,
			groupStage,
			projectStage2,
		})
	defer cancel()
	if err != nil {
		panic(err)
	}
	err = result.All(ctx, &orderItems)
	if err != nil {
		log.Fatal(err)
	}
	return orderItems, err
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		orderItemId := c.Param("order_item_id")
		fmt.Println("OrderItemId", orderItemId)
		var orderItem models.OrderItem
		err := orderItemCollection.FindOne(ctx, bson.M{"order_item_id": orderItemId}).Decode(orderItem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error: error occured while fetching order item"})
		}
		defer cancel()
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		orderItemId := c.Param("order_item_id")
		var orderItem models.OrderItem
		if err := c.BindJSON(orderItem); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var updateObj primitive.D
		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{"quantity", *orderItem.Quantity})
		}
		if orderItem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{"unit_price", *&orderItem.Unit_price})
		}
		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"food_id", *orderItem.Food_id})
		}
		orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.Updated_at})
		filter := bson.M{"order_item_id": orderItemId}
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}
		result, err := orderItemCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("message: order item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var orderItemPack OrderItemPack
		var order models.Order
		fmt.Println("Order Item")
		if err := c.BindJSON(&orderItemPack); err != nil {
			fmt.Println("error", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		order.Order_Date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Table_id = orderItemPack.Table_id
		order.Total_quantity = orderItemPack.Total_quantity
		order.Total_amount = orderItemPack.Total_amount
		order.Prepare_Status = "Pending"
		order.Created_by = &orderItemPack.Created_by
		order.User_id = orderItemPack.User_id
		order.Table_number = orderItemPack.Table_number

		orderItemsToBeInserted := []interface{}{}
		var order_id string

		// err := orderCollection.FindOne(ctx, bson.M{"table_id": orderItemPack.Table_id}).Decode(&order)
		// if err != nil {
		// 	// No order found, so create a new one
		// 	order.Order_Date = time.Now()
		// 	order.Table_id = orderItemPack.Table_id
		order_id = OrderItemOrderCreator(order)
		// } else {
		// 	// Order already exists, use the existing order ID
		// 	order_id = order.Order_id
		// }

		for _, orderItem := range orderItemPack.Order_items {
			fmt.Println("order id", order_id)
			orderItem.Order_id = order_id

			validationErr := validate.Struct(orderItem)
			if validationErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
				return
			}
			orderItem.ID = primitive.NewObjectID()
			orderItem.Order_item_id = orderItem.ID.Hex()
			orderItem.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			var num = toFixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &num
			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}
		fmt.Println(orderItemsToBeInserted...)
		insertItems, err := orderItemCollection.InsertMany(ctx, orderItemsToBeInserted)
		defer cancel()
		if err != nil {
			fmt.Println("InserMoney", err.Error())
			log.Fatal(err)
		}
		var updateTableObj primitive.D
		updateTableObj = append(updateTableObj, bson.E{"availiable", true})
		Updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateTableObj = append(updateTableObj, bson.E{"updated_at", Updated_at})
		filter := bson.M{"table_id": order.Table_id}
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		_, err = orderCollection.UpdateOne(ctx, filter, bson.D{{"$set", updateTableObj}}, &opt)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("error : table update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
			return
		}
		order.Order_id = order_id
		// notifyTable(order.Table_id)
		notifyClients(order)
		response := gin.H{
			"order_id":    order_id,
			"InsertedIDs": insertItems.InsertedIDs,
		}
		c.JSON(http.StatusOK, response)
	}
}

// GetOrdersWithItemsHandler fetches orders with associated order items and calculates total amount
func GetOrdersWithItemsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{"$lookup", bson.D{
			{"from", "orderitems"},
			{"localField", "order_id"},
			{"foreignField", "order_id"},
			{"as", "items"},
		}}},
		{{"$addFields", bson.D{
			{"total_amount", bson.D{
				{"$sum", bson.D{
					{"$map", bson.D{
						{"input", "$items"},
						{"as", "item"},
						{"in", bson.D{
							{"$multiply", bson.A{"$$item.quantity", "$$item.unit_price"}},
						}},
					}},
				}},
			}},
		}}},
	}

	cursor, err := orderCollection.Aggregate(ctx, pipeline)
	if err != nil {
		http.Error(w, "Error fetching orders with items", http.StatusInternalServerError)
		return
	}

	defer cursor.Close(ctx)

	var orders []bson.M
	if err := cursor.All(ctx, &orders); err != nil {
		http.Error(w, "Error decoding orders", http.StatusInternalServerError)
		return
	}

	// Respond with JSON
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
