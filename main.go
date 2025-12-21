package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
	"github.com/jsteffee/icloud-photo-sync/pkg/email"
	"github.com/jsteffee/icloud-photo-sync/pkg/redis"
	"github.com/jsteffee/icloud-photo-sync/pkg/scraper"
	"github.com/jsteffee/icloud-photo-sync/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	redisClient, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	storageManager, err := storage.NewManager(cfg.ImageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	emailSender, err := email.NewSender(cfg.SMTPConfig)
	if err != nil {
		log.Fatalf("Failed to initialize email sender: %v", err)
	}

	// Create scrapers for each album URL
	albumScrapers := make([]*scraper.Scraper, 0, len(cfg.AlbumURLs))
	for _, albumURL := range cfg.AlbumURLs {
		albumScrapers = append(albumScrapers, scraper.NewScraper(albumURL))
	}

	log.Printf("Starting iCloud Photo Sync Service")
	log.Printf("Album URLs: %v", cfg.AlbumURLs)
	log.Printf("Number of albums: %d", len(cfg.AlbumURLs))
	log.Printf("Run interval: %d seconds", cfg.RunInterval)
	log.Printf("Max items per run: %d", cfg.MaxItems)
	log.Printf("Image directory: %s", cfg.ImageDir)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run initial sync
	runSync(albumScrapers, storageManager, redisClient, emailSender, cfg)

	// Set up ticker for periodic runs
	ticker := time.NewTicker(time.Duration(cfg.RunInterval) * time.Second)
	defer ticker.Stop()

	// Main loop
	for {
		select {
		case <-ticker.C:
			runSync(albumScrapers, storageManager, redisClient, emailSender, cfg)
		case <-sigChan:
			log.Println("Received shutdown signal, exiting...")
			return
		}
	}
}

func runSync(
	albumScrapers []*scraper.Scraper,
	storageManager *storage.Manager,
	redisClient *redis.Client,
	emailSender *email.Sender,
	cfg *config.Config,
) {
	log.Println("Starting sync run...")

	// Collect image URLs from all albums
	var allImageURLs []string
	for i, albumScraper := range albumScrapers {
		imageURLs, err := albumScraper.GetImageURLs()
		if err != nil {
			log.Printf("Error scraping album %d: %v", i+1, err)
			continue
		}
		log.Printf("Found %d image URLs in album %d", len(imageURLs), i+1)
		allImageURLs = append(allImageURLs, imageURLs...)
	}

	log.Printf("Found %d total image URLs across all albums", len(allImageURLs))

	emailedCount := 0
	for _, imageURL := range allImageURLs {
		if emailedCount >= cfg.MaxItems {
			log.Printf("Reached MAX_ITEMS limit (%d), stopping for this run", cfg.MaxItems)
			break
		}

		// Download and hash the image
		imagePath, hash, err := storageManager.DownloadAndHash(imageURL)
		if err != nil {
			log.Printf("Error downloading image %s: %v", imageURL, err)
			continue
		}

		// Check if we've already processed this image
		exists, err := redisClient.HashExists(hash)
		if err != nil {
			log.Printf("Error checking Redis for hash %s: %v", hash, err)
			continue
		}

		if exists {
			log.Printf("Image with hash %s already processed, skipping", hash)
			continue
		}

		// Email the new image
		log.Printf("Emailing new image: %s (hash: %s)", imagePath, hash)
		if err := emailSender.SendImage(imagePath, cfg.SMTPDestination); err != nil {
			log.Printf("Error sending email for image %s: %v", imagePath, err)
			continue
		}

		// Mark as processed in Redis
		if err := redisClient.SetHash(hash, imageURL); err != nil {
			log.Printf("Error storing hash in Redis: %v", err)
			// Continue anyway since email was sent
		}

		emailedCount++
		log.Printf("Successfully processed image %s (hash: %s)", imagePath, hash)
	}

	log.Printf("Sync run completed. Emailed %d new images", emailedCount)
}

