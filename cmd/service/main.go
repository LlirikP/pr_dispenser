package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/LlirikP/pr_dispenser/internal/config"
	"github.com/LlirikP/pr_dispenser/internal/database"
	"github.com/LlirikP/pr_dispenser/internal/handlers"

	"github.com/go-chi/cors"
)

func main() {
	godotenv.Load()

	portStr := os.Getenv("PORT_AUTH")

	if portStr == "" {
		log.Fatal("Could not get PORT from .env file")
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("Could not get db url")
	}

	connection, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Could not connect to the database")
	}

	db := database.New(connection)
	config.ApiCfg = &config.ApiConfig{
		DB: db,
	}

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https//*", "http//*"},
		AllowedMethods:   []string{"OPTIONS", "GET", "POST", "DELETE", "PUT"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1router := chi.NewRouter()

	v1router.Post("/team/add", handlers.CreateTeamHandler)
	v1router.Get("/team/get", handlers.GetTeamHandler)

	v1router.Post("/users/setIsActive", handlers.SetUserActiveHandler)
	v1router.Get("/users/getReview", handlers.GetUserReviewsHandler)

	v1router.Post("/pullRequest/create", handlers.CreatePRHandler)
	v1router.Post("/pullRequest/merge", handlers.MergePRHandler)
	v1router.Post("/pullRequest/reassign", handlers.ReassignReviewerHandler)

	router.Mount("/v1", v1router)

	srv := &http.Server{
		Handler: router,
		Addr:    ":" + portStr,
	}

	log.Printf("Server starting on port %v", portStr)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
