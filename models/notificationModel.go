package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Notification struct {
	ID        primitive.ObjectID `bson:"_id"`
	User_role string             `json:"user_role"`
	User_id   string             `json:"user_id"`
	Order_id  string             `json:"order_id"`
	Is_read   bool               `json:"is_read"`
}
