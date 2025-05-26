package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		oscall := <-ch
		log.Warn().Msgf("system call:%+v", oscall)
		cancel()
	}()

	r := mux.NewRouter()
	r.HandleFunc("/", handler)

	// start: set up any of your logger configuration here if necessary
	// set global log level to debug
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// create directory logs if not exist
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}
	// open file for log
	file, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed To Open Log File")
	}
	// configuration multiple writer for console and file
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	multiWriter := zerolog.MultiLevelWriter(consoleWriter, file)
	// set output global logger
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()

	// end: set up any of your logger configuration here

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to listen and serve http server")
		}
	}()
	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("failed to shutdown http server gracefully")
	}

	// Simple Logging Example
	log.Info().Msg("Welcome To Instrumentation Class")
	log.Debug().Msg("This is a debug log at application shutdown.")
	log.Error().Msg("This is an error log during application shutdown. (Simulated)")
}

type requestIDCtxKey string

const requestIdKey requestIDCtxKey = "request_id"

func handler(w http.ResponseWriter, r *http.Request) {
	// generate new request_id
	reqID := uuid.New().String()
	// add request_id to context
	ctx := context.WithValue(r.Context(), requestIdKey, reqID)

	// create logger with request_id and function name
	log := log.With().Str(string(requestIdKey), reqID).Str("func", "handler").Logger()
	log.Debug().Msg("Incoming request received.")

	name := r.URL.Query().Get("name")
	res, err := greeting(ctx, name)
	if err != nil {
		log.Error().Err(err).Msg("Error in greeting function")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Info().Msgf("Successfully processed request for name : %s", name)
	w.Write([]byte(res))

	// ctx := r.Context()
	// name := r.URL.Query().Get("name")
	// res, err := greeting(ctx, name)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// w.Write([]byte(res))
}

func greeting(ctx context.Context, name string) (string, error) {
	reqID, ok := ctx.Value(requestIdKey).(string)
	if !ok {
		reqID = "unknown" // fallback if not found
	}

	// create logger with request_id and function name
	log := log.With().Str(string(requestIdKey), reqID).Str("func", "greeting").Logger()
	log.Debug().Msgf("Checking name length for: %s", name)

	if len(name) < 5 {
		log.Warn().Msg("Name is to short.")
		return fmt.Sprintf("Hello %s! Your name is to short\n", name), nil
	}
	log.Info().Msgf("Name length is sufficient for: %s", name)
	return fmt.Sprintf("Hi %s", name), nil
}
