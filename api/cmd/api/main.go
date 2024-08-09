package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/ogabrielrodrigues/ama/api/internal/api"
	"github.com/ogabrielrodrigues/ama/api/internal/store/pg"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s",
		os.Getenv("AMA_DATABASE_USER"),
		os.Getenv("AMA_DATABASE_PASSWORD"),
		os.Getenv("AMA_DATABASE_HOST"),
		os.Getenv("AMA_DATABASE_PORT"),
		os.Getenv("AMA_DATABASE_NAME"),
	))

	if err != nil {
		panic(err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	handler := api.NewHandler(pg.New(pool))

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%s:%s", os.Getenv("AMA_API_HOST"), os.Getenv("AMA_API_PORT")), handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	log.Println("server is running on:", fmt.Sprintf("%s:%s", os.Getenv("AMA_API_HOST"), os.Getenv("AMA_API_PORT")))
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
