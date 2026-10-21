package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/config"
	"github.com/Sidd231/4yaar/backend/internal/database"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/Sidd231/4yaar/backend/internal/modules/auth"
	"github.com/Sidd231/4yaar/backend/internal/modules/chat"
	"github.com/Sidd231/4yaar/backend/internal/modules/email"
	"github.com/Sidd231/4yaar/backend/internal/modules/files"
	"github.com/Sidd231/4yaar/backend/internal/modules/invite"
	"github.com/Sidd231/4yaar/backend/internal/modules/users"
	"github.com/Sidd231/4yaar/backend/internal/modules/videocall"
	"github.com/Sidd231/4yaar/backend/internal/router"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	migrateFlag := flag.Bool("migrate", false, "Run database auto-migrations and exit")
	flag.Parse()

	// Initialize structured logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))

	slog.Info("Starting 4Yaar backend service...")

	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load application config", "error", err)
		os.Exit(1)
	}

	// 2. Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		slog.Error("PostgreSQL connection error", "error", err)
		os.Exit(1)
	}

	// 3. Database Migration
	if err := runMigrations(db); err != nil {
		slog.Error("Database migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database schema verified and migrated successfully")

	if *migrateFlag {
		slog.Info("Migration mode completed. Exiting.")
		return
	}

	// 4. Connect to Redis
	redisClient, err := database.NewRedisClient(cfg)
	if err != nil {
		slog.Error("Redis connection error", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// 5. Ensure initial administrator exists in invite-only environment
	if err := ensureInitialAdmin(db, cfg); err != nil {
		slog.Error("Failed to bootstrap initial admin", "error", err)
		os.Exit(1)
	}

	// 6. Initialize Email Service (Amazon SES)
	emailService, err := email.NewEmailService(cfg)
	if err != nil {
		slog.Warn("Failed to initialize AWS SES client, email delivery may be unavailable", "error", err)
	}

	// 7. Initialize Repositories
	authRepo := auth.NewRepository(db, redisClient)
	inviteRepo := invite.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	filesRepo := files.NewRepository(db)

	// 8. Initialize Services
	authService := auth.NewService(authRepo, emailService, cfg)
	inviteService := invite.NewService(inviteRepo, emailService)
	chatService := chat.NewService(chatRepo)

	filesService, err := files.NewService(filesRepo, cfg)
	if err != nil {
		slog.Warn("Failed to initialize AWS S3 client, file uploads may fail", "error", err)
	}

	// 9. Initialize Real-Time Hubs
	chatHub := chat.NewHub(redisClient, chatService)
	go chatHub.Run()

	videoHub := videocall.NewHub()
	go videoHub.Run()

	videoService := videocall.NewService(db, videoHub)

	// 10. Initialize Handlers
	handlers := &router.Handlers{
		Auth:      auth.NewHandler(authService),
		Users:     users.NewHandler(db),
		Invite:    invite.NewHandler(inviteService),
		Chat:      chat.NewHandler(chatService, chatHub, cfg.JWTSecret, cfg.AllowedOrigins),
		Files:     files.NewHandler(filesService),
		VideoCall: videocall.NewHandler(videoService, videoHub, cfg.JWTSecret, cfg.AllowedOrigins),
	}

	// 11. Build HTTP Mux Router
	appRouter := router.NewRouter(cfg, handlers)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      appRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 12. Graceful Shutdown Management
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("HTTP and WebSocket server listening", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		slog.Error("Server error encountered", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		slog.Info("Shutdown signal received", "signal", sig.String())

		// Stop hubs
		chatHub.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Graceful server shutdown failed, forcing close", "error", err)
			_ = server.Close()
		}
		slog.Info("Server stopped cleanly")
	}
}

// runMigrations applies GORM automigrations for all registered models.
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Invite{},
		&models.Room{},
		&models.RoomMember{},
		&models.Message{},
		&models.File{},
	)
}

// ValidateBootstrapCredentials validates bootstrap admin credentials.
func ValidateBootstrapCredentials(email, password string) error {
	if email == "" || password == "" {
		return errors.New("admin bootstrap email and password must both be provided")
	}
	if len(password) < 8 {
		return errors.New("ADMIN_BOOTSTRAP_PASSWORD must be at least 8 characters")
	}
	if password == config.LegacyAdminPassword {
		return errors.New("ADMIN_BOOTSTRAP_PASSWORD cannot use legacy default credentials")
	}
	return nil
}

// ensureInitialAdmin seeds an initial admin if no users exist so the invite system can be bootstrapped.
// Credentials MUST be provided via ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD environment variables.
// If the user table is empty and credentials are not provided, startup will log a warning; the system
// remains usable only once an admin is created through another mechanism (e.g. a one-time CLI command).
func ensureInitialAdmin(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count existing users: %w", err)
	}
	if count > 0 {
		return nil
	}

	bootstrapEmail := cfg.AdminBootstrapEmail
	bootstrapPassword := cfg.AdminBootstrapPassword

	if bootstrapEmail == "" || bootstrapPassword == "" {
		slog.Warn("No users exist and ADMIN_BOOTSTRAP_EMAIL / ADMIN_BOOTSTRAP_PASSWORD are not set. " +
			"Set these environment variables and restart to create the initial administrator.")
		return nil
	}

	if err := ValidateBootstrapCredentials(bootstrapEmail, bootstrapPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(bootstrapPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin bootstrap password: %w", err)
	}

	admin := models.User{
		ID:           uuid.New(),
		Name:         "System Admin",
		Email:        bootstrapEmail,
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create initial admin account: %w", err)
	}

	// Log only the email, never the password
	slog.Info("Bootstrapped initial administrator account", "email", admin.Email, "role", admin.Role)
	return nil
}
