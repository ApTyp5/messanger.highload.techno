package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
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

	findDocsReadAll(client)
	//insert10milMessages(client)
	//insertBench(client)

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
	stop := 939600000
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
	_, _ = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
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

func findDocs(client *mongo.Client) {
	collectionName := "insert"
	collection := client.Database(dbn).Collection(collectionName)
	t, _ := time.Parse(time.RFC3339, "2020-11-17T11:09:01.920Z")
	filter := bson.M{"created_at": bson.M{"$lt": t}, "chat_id": bson.M{"$lt": 30}}
	message := Message{}

	curs, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	for curs.Next(ctx) == true {
		err := curs.Decode(&message)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("author: %s, chat_id: %d, created_at: %s, text: %s\n", message.Author, message.ChatId, message.CreatedAt.String(), message.Text)
	}
}

func findDocsReadAll(client *mongo.Client) {
	collectionName := "insert"
	collection := client.Database(dbn).Collection(collectionName)
	//t, _ := time.Parse(time.RFC3339, "2020-11-17T11:09:01.920Z")
	//filter := bson.M{"created_at": bson.M{"$lt": t}, "chat_id": bson.M{"$lt": 30}}
	messages := make([]Message, 0, 100)

	for i := 0; i < 10000000; i += 30 {
		filter := bson.M{"chat_id": bson.M{"$gt": i, "$lt": i + 30}}

		curs, err := collection.Find(ctx, filter)
		if err != nil {
			log.Fatal(err)
		}

		err = curs.All(ctx, &messages)
		if err != nil {
			log.Fatal(err)
		}

		if (i % 1000) == 0 {
			fmt.Printf("step %d\n", i)
		}
	}
}

func insert10milMessages(client *mongo.Client) {
	start := 0
	stop := 10000000
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
	_, _ = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
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
		message.CreatedAt = time.Now()

		tStart := time.Now()
		_, err := collection.InsertOne(ctx, message)
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
