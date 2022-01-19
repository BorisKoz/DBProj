package main

import (
	"context"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	delivery2 "repo/internal/pkg/forum/delivery"
	repository2 "repo/internal/pkg/forum/repository"
	"repo/internal/pkg/user/delivery"
	"repo/internal/pkg/user/repository"
)

type DBcfg struct {
	User string
	Host string
	Port int
	Pass string
	Name string
}


func middleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Content-Type", "application/json")
		next(ctx)
	}
}

func main() {
	cfg := DBcfg{
		User: "docker",
		Host: "localhost",
		Port: 5432,
		Pass: "docker",
		Name: "docker",
	}
	r := router.New()
	connString := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s", cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name)
	connConf, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Error().Msgf(err.Error())
		return
	}
	connConf.MaxConns = 100
	connConf.ConnConfig.PreferSimpleProtocol = true

	p, err := pgxpool.ConnectConfig(context.Background(), connConf)

	//handlers live here
	ur := repository.NewUserRep(p)
	delivery.NewUserHandler(r, &ur)

	fr := repository2.NewForumRep(p, &ur)
	delivery2.NewForumHandler(r, &fr)

	if err != nil {
		log.Error().Msgf("error connecting:"+err.Error())
	}
	err = fasthttp.ListenAndServe(":5000", middleware(r.Handler))
	if err != nil {
		log.Error().Msgf("error listening:"+err.Error())
	}
}
