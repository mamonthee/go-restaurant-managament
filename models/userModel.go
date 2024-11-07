package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      *string            `json:"name" validate:"required,min=2,max=100"`
	Password  *string            `json:"password" validate:"required,min=6"`
	Email     *string            `json:"email" validate:"email,required"`
	Avatar    *string            `json:"avatar"`
	Phone     *string            `json:"phone" validate:"required"`
	User_role *string            `json:"user_role" validate:"required,eq=ADMIN|eq=WAITER|eq=KITCHEN|eq=CASHIER"`

	Token         *string   `json:"token"`
	Refresh_Token *string   `json:"refresh_token"`
	Created_at    time.Time `json:"created_at"`
	Updated_at    time.Time `json:"updated_at"`
	User_id       string    `json:"user_id"`
}

// validate:"required,eq=ADMIN|eq=WAITER|KITCHEN|CASHIER"`
