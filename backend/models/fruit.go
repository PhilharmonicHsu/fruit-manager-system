package models

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type Fruit struct {
	Name     string `bson:"name" json:"name"`
	Quantity int    `bson:"quantity" json:"quantity"`
}

func InsertFruit(ctx context.Context, collection *mongo.Collection, fruit *Fruit) error {
	_, err := collection.InsertOne(ctx, fruit)

	return err
}

func GetFruits(ctx context.Context, collection *mongo.Collection) ([]Fruit, error) {
	var stocks []Fruit

	projection := bson.D{{"_id", 0}}
	filter := bson.D{}

	cursor, err := collection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("關閉 cursor 時發生錯誤: %v", err)
		}
	}()

	for cursor.Next(ctx) {
		var order Fruit
		if err := cursor.Decode(&order); err != nil {
			return nil, err
		}

		stocks = append(stocks, order)
	}

	return stocks, nil
}

func GetFruitByName(ctx context.Context, collection *mongo.Collection, name string) (*Fruit, error) {
	var fruit Fruit
	err := collection.FindOne(ctx, bson.M{"name": name}).Decode(&fruit)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New(fmt.Sprintf("查無此水果: %s", fruit.Name))
		}
		return nil, err
	}

	return &fruit, nil
}

func UpdateFruitQuantityByName(ctx context.Context, collection *mongo.Collection, name string, quantity int) (int64, error) {
	result, err := collection.UpdateOne(ctx, bson.M{"name": name}, bson.M{"$set": bson.M{"quantity": quantity}})
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}
