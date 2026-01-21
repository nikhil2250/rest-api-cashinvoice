package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task-management-api/config"
	"task-management-api/database"
	"task-management-api/handler"
	"task-management-api/repository"
	"task-management-api/service"
	"task-management-api/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize configuration
	config := config.LoadConfig()

	// Initialize MongoDB
	db, err := database.InitDB(config)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure database connection is closed on exit
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := db.Close(shutdownCtx); err != nil {
			log.Printf("Error closing database connection: %v", err)
		} else {
			log.Println("Database connection closed")
		}
	}()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, config.JWTSecret)
	taskService := service.NewTaskService(taskRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	taskHandler := handler.NewTaskHandler(taskService, authService)

	// Setup router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/login", authHandler.Login).Methods("POST")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	}).Methods("GET")

	// Protected routes
	api := router.PathPrefix("/tasks").Subrouter()
	api.Use(authService.AuthMiddleware)
	api.HandleFunc("", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("", taskHandler.ListTasks).Methods("GET")
	api.HandleFunc("/{id}", taskHandler.GetTask).Methods("GET")
	api.HandleFunc("/{id}", taskHandler.DeleteTask).Methods("DELETE")

	// Start background worker
	taskWorker := service.NewTaskWorker(taskRepo, config.AutoCompleteMinutes)
	go taskWorker.Start(ctx)

	// Setup server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", config.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal("Server failed to start:", err)
	case sig := <-quit:
		log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
	}

	// Cancel context to stop background workers
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		if err := srv.Close(); err != nil {
			log.Fatal("Error closing server:", err)
		}
	}

	log.Println("Server exited gracefully")
}
