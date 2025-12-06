package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client for hash tracking
type Client struct {
	client *redis.Client
	ctx    context.Context
}

// NewClient creates a new Redis client
func NewClient(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Redis client initialized successfully")
	return &Client{
		client: client,
		ctx:    ctx,
	}, nil
}

// HashExists checks if a hash exists in Redis
func (c *Client) HashExists(hash string) (bool, error) {
	key := c.hashKey(hash)
	exists, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists > 0, nil
}

// SetHash stores a hash in Redis with the associated image URL
func (c *Client) SetHash(hash string, imageURL string) error {
	key := c.hashKey(hash)
	err := c.client.Set(c.ctx, key, imageURL, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash: %w", err)
	}
	return nil
}

// GetHash retrieves the image URL associated with a hash
func (c *Client) GetHash(hash string) (string, error) {
	key := c.hashKey(hash)
	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get hash: %w", err)
	}
	return val, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// hashKey returns the Redis key for a hash
func (c *Client) hashKey(hash string) string {
	return fmt.Sprintf("image:hash:%s", hash)
}

