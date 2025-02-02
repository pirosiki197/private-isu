package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/catatsuy/private-isu/webapp/golang/cache"
	extractor "github.com/catatsuy/private-isu/webapp/golang/dynamic_extractor"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func main() {
	log.Println("main")
	extractor.StartServer()
	log.Println("extractor started")

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metricsExporter)
	go func() { log.Fatal(http.ListenAndServe(":10000", mux)) }()
	log.Println("metrics exporter started")

	host := os.Getenv("ISUCONP_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ISUCONP_DB_PORT")
	if port == "" {
		port = "3306"
	}
	_, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Failed to read DB port number from an environment variable ISUCONP_DB_PORT.\nError: %s", err.Error())
	}
	user := os.Getenv("ISUCONP_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("ISUCONP_DB_PASSWORD")
	dbname := os.Getenv("ISUCONP_DB_NAME")
	if dbname == "" {
		dbname = "isuconp"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err = sqlx.Open("mysql+cache", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %s.", err.Error())
	}
	defer db.Close()

	r := chi.NewRouter()

	r.Get("/initialize", getInitialize)
	r.Get("/login", getLogin)
	r.Post("/login", postLogin)
	r.Get("/register", getRegister)
	r.Post("/register", postRegister)
	r.Get("/logout", getLogout)
	r.Get("/", getIndex)
	r.Get("/posts", getPosts)
	r.Get("/posts/{id}", getPostsID)
	r.Post("/", postIndex)
	r.Get("/image/{id}.{ext}", getImage)
	r.Post("/comment", postComment)
	r.Get("/admin/banned", getAdminBanned)
	r.Post("/admin/banned", postAdminBanned)
	r.Get(`/@{accountName:[a-zA-Z]+}`, getAccountName)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.Dir("../public")).ServeHTTP(w, r)
	})

	log.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func metricsExporter(w http.ResponseWriter, _ *http.Request) {
	metrics := cache.ExportMetrics()
	w.Write([]byte(metrics))
}
