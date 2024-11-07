package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID             primitive.ObjectID `bson:"_id"`
	Order_Date     time.Time          `json:"order_date" validate:"required"`
	Created_at     time.Time          `json:"created_at"`
	Updated_at     time.Time          `json:"updated_at"`
	Order_id       string             `json:"order_id"`
	User_id        *string            `json:"user_id"`
	Created_by     *string            `json:"created_by"`
	Table_id       *string            `json:"table_id"`
	Table_number   *int               `json:"table_number"`
	Total_amount   float64            `json:"total_amount"`   // Total amount for the order
	Total_quantity int                `json:"total_quantity"` // Total quantity of items in the order
	Status         string             `json:"status" validate:"required,eq=CREATED|eq=INVOICED|eq=PAID"`
	Prepare_Status string             `json:"prepare_status"` // Status: "pending", "preparing", "ready"
}
