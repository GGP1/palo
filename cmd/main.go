package main

import (
	"context"
	"embed"

	"github.com/GGP1/adak/cmd/server"
	"github.com/GGP1/adak/internal/config"
	"github.com/GGP1/adak/internal/logger"
	"github.com/GGP1/adak/pkg/http/rest"
	"github.com/GGP1/adak/pkg/memcached"
	"github.com/GGP1/adak/pkg/postgres"
	"github.com/GGP1/adak/pkg/redis"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

//go:embed static
var staticFS embed.FS

func main() {
	viper.Set("static.fs", staticFS)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.New()
	if err != nil {
		logger.Fatal(err)
	}
	conf.Static.FS = staticFS

	db, err := postgres.Connect(ctx, conf.Postgres)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	mc, err := memcached.Connect(conf.Memcached)
	if err != nil {
		logger.Fatal(err)
	}

	rdb, err := redis.Connect(ctx, conf.Redis)
	if err != nil {
		logger.Fatal(err)
	}
	defer rdb.Close()

	router := rest.NewRouter(conf, db, mc, rdb)
	srv := server.New(conf, router)

	if err := srv.Start(ctx); err != nil {
		logger.Fatal(err)
	}
}
