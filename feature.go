package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type Feature struct {
	logger *log.Logger
	websrv *http.Server
}

type IFeature interface {
	Start()
	Shutdown(context.Context)
}

var _ IFeature = (*Feature)(nil)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

func (f *Feature) Start() {
	f.websrv = &http.Server{Addr: ":80", Handler: http.HandlerFunc(handler)}
	f.logger.Println("[Info] Start Web Server")
	go func() {
		if err := f.websrv.ListenAndServe(); err != nil {
			f.logger.Printf("[Error] %s\n", err)
		}
	}()
}

func (f *Feature) Shutdown(ctx context.Context) {
	f.logger.Println("[Info] Shutdown Web Server")
	if err := f.websrv.Shutdown(ctx); err != nil {
		f.logger.Printf("[Error] %s\n", err)
	}
}
