package main

import (
	"github.com/gomodule/redigo/redis"
	"log"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(1)
	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 800; i++ {
		_, err = c.Do("lpop", 1)
		if err != nil {
			log.Fatal(err)
		}
	}

	defer c.Close()
}
