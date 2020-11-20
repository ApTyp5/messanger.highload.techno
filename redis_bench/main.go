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

	for i := 0; i < 20; i++ {
		_, err = c.Do("rpush", 1, 2, 12312312, "лорем ипсум долорес сит амиет аурум")
		if err != nil {
			log.Fatal(err)
		}
	}

	defer c.Close()
}
