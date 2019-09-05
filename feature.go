package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kazufusa/go-service-example/llog"
)

type Feature struct {
	elog   *llog.Logger
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
	f.elog.Info("Start Web Server")
	go func() {
		if err := f.websrv.ListenAndServe(); err != nil {
			f.elog.Error(err)
		}
	}()
}

func (f *Feature) Shutdown(ctx context.Context) {
	f.elog.Info("Shutdown Web Server")
	if err := f.websrv.Shutdown(ctx); err != nil {
		f.elog.Error(err)
	}
}
