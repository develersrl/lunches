package brain

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type Brain struct {
	client *redis.Client
}

func New(uri string) *Brain {
	// redis://h:password@url:port
	var url, pass string
	if strings.Contains(uri, "redis://h:") {
		uri = strings.Replace(uri, "redis://h:", "", 1)
		ur := strings.Split(uri, "@")
		pass = ur[0]
		url = ur[1]
	} else {
		url = uri
		pass = ""
	}

	client := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: pass, // no password set
		DB:       0,    // use default DB
	})

	start := time.Now()
	pong, err := client.Ping().Result()
	d := time.Since(start)
	log.Printf("%s, %v", pong, d)

	if err != nil {
		log.Println("Could not ping redis url: ", err)
	}

	return &Brain{client: client}
}

func (b *Brain) Set(key string, val interface{}) error {
	encoded, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return b.client.Set(key, encoded, 0).Err()
}
func (b *Brain) Read(key string) (string, error) {
	val, err := b.client.Get(key).Result()

	if err != nil {
		return "", err
	}

	return val, nil
}

func (b *Brain) Get(key string, q interface{}) error {

	val, err := b.Read(key)

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), q)
}

func (b *Brain) Close() error {
	return b.client.Close()
}
