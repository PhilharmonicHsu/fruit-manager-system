package controllers

import (
	"context"
	"errors"
	"fmt"
	"fruit-manager-system/models"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"sync"
)

type Fruit struct {
	Name     string `bson:"name" json:"name"`
	Quantity int    `bson:"quantity" json:"quantity"`
}

type Response struct {
	Message   []string `json:"message"`
	IsSuccess bool     `json:"isSuccess"`
}

func fruitWorker(ctx context.Context, orders <-chan Fruit, errorChannel chan error, collection *mongo.Collection, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for order := range orders {
		targetOrder, err := models.GetFruitByName(ctx, collection, order.Name)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				errorChannel <- errors.New(fmt.Sprintf("%s 倉庫無此種類", order.Name))

				return
			} else {
				errorChannel <- errors.New(err.Error())

				return
			}
		}

		remainingQuantity := targetOrder.Quantity - order.Quantity

		if remainingQuantity >= 0 {
			_, updateErr := models.UpdateFruitQuantityByName(ctx, collection, order.Name, remainingQuantity)
			if updateErr != nil {
				errorChannel <- errors.New(updateErr.Error())

				return
			}
		} else {
			errorChannel <- errors.New(fmt.Sprintf("%s 庫存不足", order.Name))

			return
		}

		errorChannel <- nil
	}
}

func restockWorker(ctx context.Context, fruits <-chan Fruit, errorChannel chan error, collection *mongo.Collection, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for fruit := range fruits {
		targetStock, err := models.GetFruitByName(ctx, collection, fruit.Name)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				insertErr := models.InsertFruit(ctx, collection, targetStock)
				if insertErr != nil {
					errorChannel <- errors.New(insertErr.Error())

					return
				}
			} else {
				errorChannel <- errors.New(err.Error())

				return
			}
		}
		_, updateErr := models.UpdateFruitQuantityByName(ctx, collection, fruit.Name, targetStock.Quantity+fruit.Quantity)
		if updateErr != nil {
			errorChannel <- errors.New(updateErr.Error())

			return
		}
	}
}

func HandleOrderFruits(context echo.Context) error {
	var orders []Fruit

	if err := context.Bind(&orders); err != nil {
		return context.JSON(http.StatusUnprocessableEntity, Response{
			Message:   []string{"請求格式錯誤"},
			IsSuccess: false,
		})
	}
	database := context.Get("db").(*mongo.Database)
	collection := database.Collection("fruit")
	ctx := context.Request().Context()

	var waitGroup sync.WaitGroup

	channels := make(map[string]chan Fruit)
	errChannel := make(chan error, len(orders))

	for _, order := range orders {
		channel, exists := channels[order.Name]

		if !exists {
			channel = make(chan Fruit)
			channels[order.Name] = channel
			waitGroup.Add(1)
			go fruitWorker(ctx, channel, errChannel, collection, &waitGroup)
		}

		channel <- order
	}

	for _, channel := range channels {
		close(channel)
	}

	waitGroup.Wait()

	/**
	Because errChannel travels through various channels,
	it cannot be closed until all channels are completed,
	and only after waitGroup.Wait() is completed does it mean that all channels are completed.
	*/
	close(errChannel)

	var errorMessages []string
	for err := range errChannel {
		if err != nil {
			errorMessages = append(errorMessages, err.Error())
		}
	}

	return context.JSON(http.StatusOK, Response{
		Message:   errorMessages,
		IsSuccess: len(errorMessages) == 0,
	})
}

func HandleRestock(context echo.Context) error {
	var restocks []Fruit
	var waitGroup sync.WaitGroup

	if err := context.Bind(&restocks); err != nil {
		return context.JSON(http.StatusUnprocessableEntity, Response{
			Message:   []string{"請求格式錯誤"},
			IsSuccess: false,
		})
	}
	ctx := context.Request().Context()
	database := context.Get("db").(*mongo.Database)
	collection := database.Collection("fruit")

	channels := make(map[string]chan Fruit)
	errChannel := make(chan error, len(restocks))

	for _, restock := range restocks {
		channel, exists := channels[restock.Name]
		if !exists {
			channel = make(chan Fruit)
			channels[restock.Name] = channel
			waitGroup.Add(1)
			go restockWorker(ctx, channel, errChannel, collection, &waitGroup)
		}

		channel <- restock
	}

	for _, channel := range channels {
		close(channel)
	}
	waitGroup.Wait()

	close(errChannel)

	var errorMessages []string
	for err := range errChannel {
		if err != nil {
			errorMessages = append(errorMessages, err.Error())
		}
	}

	return context.JSON(http.StatusOK, Response{
		Message:   errorMessages,
		IsSuccess: len(errorMessages) == 0,
	})
}

func HandleSearchStocks(context echo.Context) error {
	database := context.Get("db").(*mongo.Database)
	collection := database.Collection("fruit")

	stocks, err := models.GetFruits(context.Request().Context(), collection)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.JSON(http.StatusOK, stocks)
}

func HandleUpdateStock(context echo.Context) error {
	var fruit Fruit
	if err := context.Bind(&fruit); err != nil {
		return context.JSON(http.StatusUnprocessableEntity, Response{
			Message:   []string{"請求格式錯誤"},
			IsSuccess: false,
		})
	}

	database := context.Get("db").(*mongo.Database)
	collection := database.Collection("fruit")
	ctx := context.Request().Context()

	targetFruit, err := models.GetFruitByName(ctx, collection, fruit.Name)
	if err != nil {
		return context.JSON(http.StatusUnprocessableEntity, Response{
			Message:   []string{err.Error()},
			IsSuccess: false,
		})
	}

	modifiedCount, updateErr := models.UpdateFruitQuantityByName(ctx, collection, fruit.Name, fruit.Quantity)
	if updateErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, updateErr.Error())
	}

	if modifiedCount == 0 {
		return context.JSON(http.StatusOK, Response{
			Message:   []string{fmt.Sprintf("沒有任何更新，現有庫存: %d", targetFruit.Quantity)},
			IsSuccess: true,
		})
	}

	return context.JSON(http.StatusOK, Response{
		Message:   []string{},
		IsSuccess: true,
	})
}
