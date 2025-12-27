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

// HashExists checks if a hash exists in Redis (for email - kept for backward compatibility)
func (c *Client) HashExists(hash string) (bool, error) {
	return c.HashExistsForEmail(hash)
}

// SetHash stores a hash in Redis with the associated image URL (for email - kept for backward compatibility)
func (c *Client) SetHash(hash string, imageURL string) error {
	return c.SetHashForEmail(hash, imageURL)
}

// GetHash retrieves the image URL associated with a hash
func (c *Client) GetHash(hash string) (string, error) {
	key := c.hashKey("email", hash)
	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get hash: %w", err)
	}
	return val, nil
}

// HashExistsForEmail checks if a hash exists in Redis for email tracking
func (c *Client) HashExistsForEmail(hash string) (bool, error) {
	key := c.hashKey("email", hash)
	exists, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists > 0, nil
}

// SetHashForEmail stores a hash in Redis with the associated image URL for email tracking
func (c *Client) SetHashForEmail(hash string, imageURL string) error {
	key := c.hashKey("email", hash)
	err := c.client.Set(c.ctx, key, imageURL, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash: %w", err)
	}
	return nil
}

// HashExistsForGooglePhotos checks if a hash exists in Redis for Google Photos tracking
func (c *Client) HashExistsForGooglePhotos(hash string) (bool, error) {
	key := c.hashKey("google_photos", hash)
	exists, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists > 0, nil
}

// SetHashForGooglePhotos stores a hash in Redis with the associated image URL for Google Photos tracking
func (c *Client) SetHashForGooglePhotos(hash string, imageURL string) error {
	key := c.hashKey("google_photos", hash)
	err := c.client.Set(c.ctx, key, imageURL, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// hashKey returns the Redis key for a hash with a prefix
func (c *Client) hashKey(prefix, hash string) string {
	return fmt.Sprintf("image:hash:%s:%s", prefix, hash)
}

