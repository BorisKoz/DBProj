package utils

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"strconv"
)

func Send(code int, data interface{}, ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(code)
	marshalled, err := json.Marshal(data)
	if err != nil {
		ctx.SetStatusCode(500)
		return
	}
	ctx.SetBody(marshalled)
}

func GetQueryString(ctx *fasthttp.RequestCtx, key string) string {
	bytes := ctx.QueryArgs().Peek(key)
	res := string(bytes)
	return res
}

func GetQueryInt(ctx *fasthttp.RequestCtx, key string) (int, error) {
	bytes := ctx.QueryArgs().Peek(key)
	res := string(bytes)
	if res == "" {
		return 0, nil
	}
	intres, err := strconv.Atoi(res)
	if err != nil {
		return -1, err
	}
	return intres, nil
}

func GetQueryBool(ctx *fasthttp.RequestCtx, key string) (bool, error) {
	bytes := ctx.QueryArgs().Peek(key)
	res := string(bytes)
	if res == "" {
		return false, nil
	}
	boolres, err := strconv.ParseBool(res)
	if err != nil {
		return false, err
	}
	return boolres, nil
}