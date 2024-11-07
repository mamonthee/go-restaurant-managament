package controllers

import (
	"context"
	"fmt"
	"go-restaurant-managament/database"
	"go-restaurant-managament/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		result, err := invoiceCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while getting invoices"})
			return
		}
		var allInvoices []bson.M
		if err = result.All(ctx, &allInvoices); err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		invoiceId := c.Param("invoice_id")
		var invoice models.Invoice
		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while fetching invoice item"})
			return
		}

		var invoiceView InvoiceViewFormat

		allOrderItems, err := ItemsByOrder(invoice.Order_id)
		invoiceView.Order_id = invoice.Order_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		invoiceView.Payment_method = "null"
		if invoiceView.Payment_method != "" {
			invoiceView.Payment_method = *invoice.Payment_method
		}

		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = *&invoice.Payment_status
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]
		c.JSON(http.StatusOK, invoiceView)
	}
}
func GetInvoiceByDate() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Parse startDate and endDate from request parameters
		startDateStr := c.Param("startDate")
		endDateStr := c.Param("endDate")
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}

		// Aggregation pipeline
		match := bson.D{{"$match", bson.D{{"created_at", bson.D{{"$gte", startDate}, {"$lte", endDate}}}}}}

		lookup := bson.D{{"$lookup", bson.D{
			{"from", "order"},
			{"localField", "order_id"},
			{"foreignField", "order_id"},
			{"as", "order"},
		}}}

		unwind := bson.D{{"$unwind", bson.D{{"path", "$order"}, {"preserveNullAndEmptyArrays", true}}}}

		group := bson.D{{"$group", bson.D{
			{"_id", "$_id"}, // Group by invoice ID
			{"total_amount", bson.D{{"$sum", "$order.total_amount"}}},
			{"table_number", bson.D{{"$first", "$order.table_number"}}},
			{"invoice_data", bson.D{{"$first", "$$ROOT"}}}, // Preserve original invoice data
		}}}

		project := bson.D{{"$project", bson.D{
			{"invoice_id", "$invoice_data._id"},
			{"invoice_number", "$invoice_data.invoice_number"},
			{"order_id", "$invoice_data.order_id"},
			{"payment_method", "$invoice_data.payment_method"},
			{"payment_status", "$invoice_data.payment_status"},
			{"payment_due_date", "$invoice_data.payment_due_date"},
			{"created_at", "$invoice_data.created_at"},
			{"updated_at", "$invoice_data.updated_at"},
			{"table_number", "$table_number"},
			{"total_amount", bson.D{{"$ifNull", []interface{}{"$total_amount", 0}}}}, // Default to 0 if null
		}}}

		// Run the pipeline
		pipeline := mongo.Pipeline{match, lookup, unwind, group, project}
		cursor, err := invoiceCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching invoices"})
			return
		}

		var invoices []bson.M
		if err := cursor.All(ctx, &invoices); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding invoices"})
			return
		}

		// Return the aggregated result
		c.JSON(http.StatusOK, invoices)
	}
}

//	func getDailySequence(date string) int {
//		// TODO: Implement logic to query your database for the last invoice number created on the given date
//		// For demonstration, we'll assume this function returns a sequence number.
//		return 1 // Replace with actual logic to get the current sequence number
//	}
func getLastInvoiceNumber(date string) (string, error) {
	// Define a filter for the invoice number that matches the date
	filter := bson.M{
		"invoice_number": bson.M{
			"$regex": fmt.Sprintf("^INV-%s-", date), // Use regex to match the invoice number format
		},
	}

	// Find the last invoice by sorting on createdAt in descending order
	options := options.FindOne().SetSort(bson.M{"created_at": -1})
	fmt.Println("OPTIONS")
	fmt.Println(options)

	var lastInvoice models.Invoice
	err := invoiceCollection.FindOne(context.TODO(), filter, options).Decode(&lastInvoice)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil // No invoices for today
		}
		return "", err // Handle other errors
	}

	return lastInvoice.Invoice_number, nil
}

func generateInvoiceNumber() (string, error) {
	prefix := "INV"
	date := time.Now().Format("20060102") // Format date as YYYYMMDD

	lastInvoiceNumber, err := getLastInvoiceNumber(date)
	if err != nil {
		return "", err // Handle the error as necessary
	}

	var sequence int
	if lastInvoiceNumber == "" {
		sequence = 1 // First invoice of the day
	} else {
		parts := strings.Split(lastInvoiceNumber, "-")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid invoice number format")
		}

		sequenceStr := parts[2]
		seq, err := strconv.Atoi(sequenceStr)
		if err != nil {
			return "", err
		}
		sequence = seq + 1 // Increment the sequence number
	}

	// Format the sequence as a 4-digit string
	sequenceStr := fmt.Sprintf("%04d", sequence)
	invoiceNumber := fmt.Sprintf("%s-%s-%s", prefix, date, sequenceStr)

	return invoiceNumber, nil
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
		var invoice models.Invoice
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Set default payment status to "PENDING" if not provided
		if invoice.Payment_status == nil || *invoice.Payment_status == "" {
			status := "PENDING"
			invoice.Payment_status = &status
		}
		// status := "PENDING"
		// if invoice.Payment_status != nil {
		// 	invoice.Payment_status = &status
		// }
		invoice.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))
		invoice.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()
		invoiceNumber, err := generateInvoiceNumber()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		invoice.Invoice_number = invoiceNumber

		validateError := validate.Struct(&invoice)
		if validateError != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validateError.Error()})
		}
		result, err := invoiceCollection.InsertOne(ctx, invoice)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("invoice item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		fmt.Println("Paid status", *invoice.Payment_status)
		// Check if Order needs to be updated based on invoice payment status
		if *invoice.Payment_status == "PAID" {
			err = UpdateOrderStatus(ctx, invoice.Order_id, "PAID")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order status"})
				return
			}
		}
		c.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		invoiceId := c.Param("invoice_id")
		var invoice models.Invoice
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var updateObj primitive.D
		if invoice.Payment_method != nil {
			updateObj = append(updateObj, bson.E{"payment_method", invoice.Payment_method})
		}

		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
			updateObj = append(updateObj, bson.E{"payment_status", invoice.Payment_status})
		}

		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", invoice.Updated_at})
		upsert := true
		opts := options.UpdateOptions{
			Upsert: &upsert,
		}
		filter := bson.M{"invoice_id": invoiceId}

		result, updateErr := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opts,
		)
		defer cancel()
		if updateErr != nil {
			msg := fmt.Sprintf("message: invoice item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
