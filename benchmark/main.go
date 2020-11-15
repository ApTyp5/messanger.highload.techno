package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
	"runtime"
	"time"
)

var ctx = context.TODO()
var uri = "mongodb://localhost:27017/"
var dbn = "new"
var col = "messages"
var maxPoolSize uint64 = 100

func main() {
	client := connectNverify()
	showNsetSettings()

	insert12gb(client)
	insertBench(client)

	disconnect(client)
}

func showNsetSettings() {
	fmt.Printf("cpu num: %d\n", runtime.NumCPU())
	runtime.GOMAXPROCS(4)
}

func disconnect(client *mongo.Client) {
	err := client.Disconnect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("bye")
}

func connectNverify() *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(maxPoolSize)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("hi")
	return client
}

func insertBench(client *mongo.Client) {
	start := 0
	stop := 100000
	count := stop - start
	collectionName := "insert"
	collection := client.Database(dbn).Collection(collectionName)

	message := Message{
		Author:    "Name",
		ChatId:    -1,
		CreatedAt: time.Now(),
		Text:      "dog's hotdogs are hot as dogs",
	}

	t := time.Now()
	collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bsonx.Doc{
			{Key: "chat_id", Value: bsonx.Int32(1)},
			{Key: "created_at", Value: bsonx.Int32(1)},
		},
		Options: nil,
	})
	fmt.Printf("index created for %dmcs\n", time.Now().Sub(t).Nanoseconds()/1000)

	var tSum int64 = 0

	for i := start; i < stop; i++ {
		message.ChatId = i

		tStart := time.Now()
		_, err := collection.InsertOne(ctx, message)
		tStop := time.Now()
		tSum += tStop.Sub(tStart).Nanoseconds() / 1000

		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("average time = %dmcs\n", tSum/int64(count))
}

func insert12gb(client *mongo.Client) {
	start := 0
	stop := 100000
	count := stop - start
	collectionName := "insert"
	collection := client.Database(dbn).Collection(collectionName)

	message := Message{
		Author:    "Name",
		ChatId:    -1,
		CreatedAt: time.Now(),
		Text:      "dog's hotdogs are hot as dogs",
	}

	t := time.Now()
	collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bsonx.Doc{
			{Key: "chat_id", Value: bsonx.Int32(1)},
			{Key: "created_at", Value: bsonx.Int32(1)},
		},
		Options: nil,
	})
	fmt.Printf("index created for %dmcs\n", time.Now().Sub(t).Nanoseconds()/1000)

	var tSum int64 = 0

	for i := start; i < stop; i++ {
		fmt.Printf("step %d started\n", i)
		message.ChatId = i
		arr := make([]interface{}, 0, 1)
		for j := 0; j < 4000; j++ {
			arr = append(arr, message)
		}

		tStart := time.Now()
		_, err := collection.InsertMany(ctx, arr)
		tStop := time.Now()
		tSum += tStop.Sub(tStart).Nanoseconds() / 1000

		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("step %d ended\n", i)
	}

	fmt.Printf("average time = %dmcs\n", tSum/int64(count))
}

type Message struct {
	Author    string    `bson:"author_id"`
	ChatId    int       `bson:"chat_id"`
	CreatedAt time.Time `bson:"created_at"`
	Text      string    `bson:"text"`
}
