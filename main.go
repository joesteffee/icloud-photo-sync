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
	for _, imageURL := range allImageURLs {
		if processedCount >= cfg.MaxItems {
			log.Printf("Reached MAX_ITEMS limit (%d), stopping for this run", cfg.MaxItems)
			break
		}

		// Download and hash the image
		imagePath, hash, err := storageManager.DownloadAndHash(imageURL)
		if err != nil {
			log.Printf("Error downloading image %s: %v", imageURL, err)
			continue
		}

		// Check if we've already processed this image for email
		emailExists, err := redisClient.HashExistsForEmail(hash)
		if err != nil {
			log.Printf("Error checking Redis for email hash %s: %v", hash, err)
			continue
		}

		if emailExists {
			log.Printf("Image with hash %s already processed for email, skipping", hash)
			continue
		}

		// Process new image: email and upload to Google Photos
		emailSuccess := false
		googlePhotosSuccess := false

		// Email the new image
		log.Printf("Emailing new image: %s (hash: %s)", imagePath, hash)
		if err := emailSender.SendImage(imagePath, cfg.SMTPDestination); err != nil {
			log.Printf("Error sending email for image %s: %v", imagePath, err)
		} else {
			emailSuccess = true
		}

		// Upload to Google Photos if configured
		if photosClient != nil && googlePhotosAlbumID != "" {
			// Check if already uploaded to Google Photos
			gphotosExists, err := redisClient.HashExistsForGooglePhotos(hash)
			if err != nil {
				log.Printf("Error checking Redis for Google Photos hash %s: %v", hash, err)
			} else if !gphotosExists {
				log.Printf("Uploading new image to Google Photos: %s (hash: %s)", imagePath, hash)
				if err := photosClient.UploadPhoto(imagePath, googlePhotosAlbumID); err != nil {
					log.Printf("Error uploading to Google Photos for image %s: %v", imagePath, err)
				} else {
					googlePhotosSuccess = true
					// Mark as processed for Google Photos
					if err := redisClient.SetHashForGooglePhotos(hash, imageURL); err != nil {
						log.Printf("Error storing Google Photos hash in Redis: %v", err)
					}
				}
			} else {
				log.Printf("Image with hash %s already uploaded to Google Photos, skipping upload", hash)
				googlePhotosSuccess = true // Already processed
			}
		}

		// Mark as processed for email if email was sent successfully
		if emailSuccess {
			if err := redisClient.SetHashForEmail(hash, imageURL); err != nil {
				log.Printf("Error storing email hash in Redis: %v", err)
				// Continue anyway since email was sent
			}
		}

		processedCount++
		log.Printf("Successfully processed image %s (hash: %s) - Email: %v, Google Photos: %v", 
			imagePath, hash, emailSuccess, googlePhotosSuccess)
	}

	log.Printf("Sync run completed. Processed %d new images", processedCount)
}

