package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"multiplexer.lavrentev.dev/internal/web"
)

type Config struct {
	Web web.Config
}

func main() {
	cfg := Config{
		Web: web.Config{
			Addr: ":8000",
		},
	}

	log.Println("init server")
	server, err := web.NewServer(cfg.Web)
	if err != nil {
		log.Fatal(err)
	}

	serverError := make(chan error, 1)
	go func() {
		log.Println("start the server")
		serverError <- server.Start()
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-term:
		log.Println("got an interrupt signal")
	case err := <-serverError:
		log.Printf("server is down %v\n", err)
	}

	log.Println("shutdown")
	if err := server.Shutdown(); err != nil {
		log.Printf("Successfully failed to shutdown server gracefully: %v\n", err)
	}
}
