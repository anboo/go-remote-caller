package main

import (
	"github.com/go-redis/redis"
	"time"
	"github.com/twinj/uuid"
	"net"
	"bytes"
	"fmt"
	"encoding/json"
	"os"
	"log"
)

type CallerService struct {
	UUID uuid.Uuid
	IP   net.IP
	CountryCode string
	Key string
}


type UnexpectedError struct {
	msg	string
}

func (err *UnexpectedError) Error() string {
	return err.msg
}


type DataProvider interface {
	get(string) ([]byte, error)
	set(string, []byte) error
	has(string) (bool, error)
}


type MemoryDataProvider struct {
	items	map[string] []byte
}

func (this *MemoryDataProvider) get(key string) ([]byte, error) {
	_, err := this.has(key); if err != nil {
		return []byte{}, err
	} else {
		return this.items[key], nil
	}
}

func (this *MemoryDataProvider) set(key string, data []byte) error {
	this.items[key] = data

	return nil
}

func (this *MemoryDataProvider) has(key string) (bool, error) {
	_, ok := this.items[key]

	return ok, nil
}


type RedisDataProvider struct {
	Address  string
	Password string
	DB 		 int
	Client   *redis.Client
}

func (this *RedisDataProvider) getClient() *redis.Client {
	if this.Client == nil {
		this.Client = redis.NewClient(&redis.Options{
			Addr:     this.Address,
			Password: this.Password, // no password set
			DB:       this.DB,  // use default DB
		})
	} else {
		/*
		 * Try restart if connection session is expired
		 */
		_, err := this.Client.Ping().Result(); if err != nil {
			this.Client = nil
			return this.getClient()
		}
	}

	defer this.Client.Close()

	return this.Client
}

func (this *RedisDataProvider) get(key string) ([]byte, error) {
	cl := this.getClient()

	data, err := cl.Get(key).Bytes(); if err != nil {
		return []byte {}, err
	} else {
		return data, nil
	}
}

func (this *RedisDataProvider) set(key string, data []byte) error {
	cl := this.getClient()

	res := cl.Set(key, data, time.Duration(0));

	return res.Err()
}

func (this *RedisDataProvider) has(key string) (bool, error) {
	cl := this.getClient()

	res := cl.Get(key); if res.Err() != nil {
		return false, res.Err()
	}

	if len(res.Val()) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func main() {
	conf := getConfig();

	strg := RedisDataProvider{
		DB: conf.redis.db,
		Address: conf.redis.addr,
		Password: conf.redis.pass,
	}

	strg.set("Nginx", bytes.NewBufferString("Hello from Server").Bytes())

	data, err := strg.get("Nginx"); if err != nil {
		fmt.Printf("Error %s", err.Error())
	} else {
		fmt.Printf("Result %s", data)
	}
}

type Configuration struct {
	redis struct {
		addr string
		db 	 int
		pass string
	}
}

func getConfig() Configuration {
	conf := Configuration{}

	file, err := os.Open("conf.json"); if err != nil {
		log.Printf("Error open configuration file: %s", err)
	}

	json.NewEncoder(file).Encode(&conf)

	return conf
}