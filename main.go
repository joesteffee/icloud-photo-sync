package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
	"github.com/jsteffee/icloud-photo-sync/pkg/email"
	"github.com/jsteffee/icloud-photo-sync/pkg/photos"
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

	// Initialize Google Photos client if configured
	var photosClient *photos.Client
	if cfg.GooglePhotosConfig != nil {
		photosClient, err = photos.NewClient(cfg.GooglePhotosConfig)
		if err != nil {
			log.Fatalf("Failed to initialize Google Photos client: %v", err)
		}
		log.Printf("Google Photos integration enabled for album: %s", cfg.GooglePhotosConfig.AlbumName)
	} else {
		log.Printf("Google Photos integration disabled (no configuration provided)")
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
	runSync(albumScrapers, storageManager, redisClient, emailSender, photosClient, cfg)

	// Set up ticker for periodic runs
	ticker := time.NewTicker(time.Duration(cfg.RunInterval) * time.Second)
	defer ticker.Stop()

	// Main loop
	for {
		select {
		case <-ticker.C:
			runSync(albumScrapers, storageManager, redisClient, emailSender, photosClient, cfg)
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
	photosClient *photos.Client,
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

	// Get Google Photos album ID if configured (cache it for the run)
	// With new API scopes, the album will be created if it doesn't exist
	var googlePhotosAlbumID string
	if photosClient != nil {
		albumID, err := photosClient.GetOrCreateAlbumID()
		if err != nil {
			log.Printf("Error getting/creating Google Photos album: %v. Google Photos sync will be skipped for this run.", err)
			photosClient = nil // Disable Google Photos for this run
		} else {
			googlePhotosAlbumID = albumID
			log.Printf("Using Google Photos album ID: %s", googlePhotosAlbumID)
		}
	}

	processedCount := 0
	log.Printf("Starting to process %d image URLs", len(allImageURLs))
	for i, imageURL := range allImageURLs {
		if processedCount >= cfg.MaxItems {
			log.Printf("Reached MAX_ITEMS limit (%d), stopping for this run", cfg.MaxItems)
			break
		}

		log.Printf("Processing image %d/%d: %s", i+1, len(allImageURLs), imageURL)

		// Download and hash the image (high-quality version only - original or medium)
		// The scraper ensures only high-quality images are selected (skips thumbnails)
		// This same high-quality image will be used for both email and Google Photos
		imagePath, hash, err := storageManager.DownloadAndHash(imageURL)
		if err != nil {
			log.Printf("Error downloading image %s: %v", imageURL, err)
			continue
		}
		log.Printf("Downloaded and hashed image: %s (hash: %s)", imagePath, hash)

		// Check processing status for both email and Google Photos independently
		emailExists, err := redisClient.HashExistsForEmail(hash)
		if err != nil {
			log.Printf("Error checking Redis for email hash %s: %v", hash, err)
			continue
		}
		log.Printf("Email tracking check for hash %s: exists=%v", hash, emailExists)

		gphotosExists := false
		if photosClient != nil && googlePhotosAlbumID != "" {
			var err2 error
			gphotosExists, err2 = redisClient.HashExistsForGooglePhotos(hash)
			if err2 != nil {
				log.Printf("Error checking Redis for Google Photos hash %s: %v", hash, err2)
			} else {
				log.Printf("Google Photos tracking check for hash %s: exists=%v", hash, gphotosExists)
			}
		}

		// Skip if already processed for both services
		if emailExists && (photosClient == nil || gphotosExists) {
			log.Printf("Image with hash %s already processed for all services, skipping", hash)
			continue
		}

		// Process image for email and/or Google Photos as needed
		// Both services use the same high-quality downloaded image file
		emailSuccess := false
		googlePhotosSuccess := false

		// Email the image if not already emailed
		if !emailExists {
			log.Printf("Emailing high-quality image: %s (hash: %s)", imagePath, hash)
			if err := emailSender.SendImage(imagePath, cfg.SMTPDestination); err != nil {
				log.Printf("Error sending email for image %s: %v", imagePath, err)
			} else {
				emailSuccess = true
				// Mark as processed for email
				if err := redisClient.SetHashForEmail(hash, imageURL); err != nil {
					log.Printf("Error storing email hash in Redis: %v", err)
				}
			}
		} else {
			log.Printf("Image with hash %s already emailed, skipping email", hash)
			emailSuccess = true // Already processed
		}

		// Upload to Google Photos if configured and not already uploaded
		if photosClient != nil && googlePhotosAlbumID != "" && !gphotosExists {
			log.Printf("Uploading high-quality image to Google Photos: %s (hash: %s)", imagePath, hash)
			if err := photosClient.UploadPhoto(imagePath, googlePhotosAlbumID); err != nil {
				log.Printf("Error uploading to Google Photos for image %s: %v", imagePath, err)
			} else {
				googlePhotosSuccess = true
				// Mark as processed for Google Photos
				if err := redisClient.SetHashForGooglePhotos(hash, imageURL); err != nil {
					log.Printf("Error storing Google Photos hash in Redis: %v", err)
				}
			}
		} else if photosClient != nil && googlePhotosAlbumID != "" && gphotosExists {
			log.Printf("Image with hash %s already uploaded to Google Photos, skipping upload", hash)
			googlePhotosSuccess = true // Already processed
		}

		// Only count as processed if we actually did something new
		if emailSuccess || googlePhotosSuccess {
			processedCount++
			log.Printf("Successfully processed image %s (hash: %s) - Email: %v, Google Photos: %v", 
				imagePath, hash, emailSuccess, googlePhotosSuccess)
		} else {
			log.Printf("Failed to process image %s (hash: %s) for both email and Google Photos - Email: %v, Google Photos: %v", 
				imagePath, hash, emailSuccess, googlePhotosSuccess)
		}
	}

	log.Printf("Sync run completed. Processed %d new images", processedCount)
}

