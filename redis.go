package redislock

import (
	"context"
	"errors"
	"time"
)
import "github.com/redis/go-redis/v9"

type LockClient interface {
	SetNEX(ctx context.Context, key string, value string, expiration int64) (int64, error)
	Even(ctx context.Context, src string, keys []string, Values []interface{}) (interface{}, error)
}

type Client struct {
	client *redis.Client
	ClientOptions
}

var _ LockClient = (*Client)(nil)

func NewClient(network string, addr string, password string, options ...ClientOption) *Client {
	c := &Client{
		ClientOptions: ClientOptions{
			network:  network,
			addr:     addr,
			password: password,
		},
	}
	for _, option := range options {
		option(&c.ClientOptions)
	}
	repairClient(&c.ClientOptions)

	r := redis.NewClient(&redis.Options{
		Network:    c.network,
		Addr:       c.addr,
		Password:   c.password,
		MaxRetries: c.maxRetries,
	})
	c.client = r
	return c
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if len(key) == 0 {
		return "", errors.New("key is empty")
	}
	return c.client.Do(ctx, "GET", key).Text()
}

func (c *Client) SetEX(ctx context.Context, key string, value string, expireSeconds int64) (int64, error) {
	if len(key) == 0 || len(value) == 0 {
		return -1, errors.New("key or value is empty")
	}
	val, err := c.client.SetEx(ctx, key, value, time.Duration(expireSeconds)).Result()
	if err != nil {
		return -1, err
	}
	if val == "OK" {
		return 1, nil
	}
	return 0, err
}

func (c *Client) SetNEX(ctx context.Context, key string, value string, expiration int64) (int64, error) {
	if len(key) == 0 || len(value) == 0 {
		return -1, errors.New("redis set keyNX or value can't be empty")
	}
	val, err := c.client.Do(ctx, "SET", key, value, "EX", expiration, "NX").Result()
	if err != nil {
		return -1, err
	}
	if respString, ok := val.(string); ok && respString == "OK" {
		return 1, nil
	}
	return 0, nil
}

func (c *Client) Even(ctx context.Context, src string, keys []string, Values []interface{}) (interface{}, error) {
	var script = redis.NewScript(src)
	return script.Run(ctx, c.client, keys, Values...).Result()
}
