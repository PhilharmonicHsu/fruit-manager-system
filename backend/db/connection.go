package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var Client *mongo.Client

func InitMongo(uri string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOption := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOption)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("無法連線到 MongoDB:", err)
	}

	Client = client

	log.Println("成功連線到 MongoDB")
}
