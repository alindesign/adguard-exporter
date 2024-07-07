package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"log"

	"github.com/alindesign/adguard-exporter/internal/adguard"
	"github.com/alindesign/adguard-exporter/internal/config"
	"github.com/alindesign/adguard-exporter/internal/http"
	"github.com/alindesign/adguard-exporter/internal/metrics"
	"github.com/alindesign/adguard-exporter/internal/worker"
)

func main() {
	metrics.Setup()

	configuration, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	var clients []*adguard.Client
	for _, client := range configuration.Clients {
		clients = append(clients, adguard.NewClient(client))
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	server := http.NewHttp(configuration)
	go func() {
		err := server.Serve()
		if err != nil {
			sigs <- syscall.SIGTERM
			log.Printf("Error starting server: %v", err)
		}
	}()
	go worker.Work(ctx, configuration.Interval, clients)

	<-sigs
	if err := server.Stop(ctx); err != nil {
		panic(err)
	}
}
