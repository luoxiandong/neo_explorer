package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	eParser "github.com/go-errors/errors"
	"github.com/valyala/fasthttp"
	"neo_explorer/core/log"
	"net/http"
	"strings"
	"time"
)

var (
	client = &http.Client{Timeout: 20 * time.Second}
)

// JsonRPCResponse returns rpc response data.
type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
}

func getRPCRequestBody(method string, params []interface{}) string {
	p := ""

	for _, param := range params {
		switch param.(type) {
		case int8, uint8,
			int16, uint16,
			int, uint,
			int32, uint32,
			int64, uint64:
			p += fmt.Sprintf("%d, ", param)
		case string:
			p += fmt.Sprintf("\"%s\", ", param)
		default:
			err := fmt.Errorf("the RPC parameter type must be integer or string. current type=%T, value=%v", param, param)
			panic(err)
		}
	}

	if p != "" {
		p = p[:len(p)-2]
	}

	body := `{
		"jsonrpc": "2.0",
		"method": "` + method + `",
		"params": [
			` + p + `
		],
		"id": 1
	}
	`
	return body
}

func rpcCall(minHeight int, params string, target interface{}) {
	call(minHeight, params, target)
}

func call(minHeight int, params string, target interface{}) {
	requestBody := []byte(params)
	resp := fasthttp.AcquireResponse()
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("POST")
	req.SetBody(requestBody)

	client := &fasthttp.Client{}

	for {
		url, ok := getServer(minHeight)
		if !ok {
			if strings.Contains(params, `"getblock"`) {
				// Exceed the highest block index, return nil target.
				return
			}
			delay := 3
			fmt.Printf("No server's height higher than or equal to %d\nWaiting for %d seconds before retry\n", minHeight, delay)
			time.Sleep(time.Duration(delay) * time.Second)
			PrintServerStatus()
			continue
		}

		req.SetRequestURI(url)
		err := client.Do(req, resp)
		if err != nil {
			log.Error.Println(err)
			serverUnavailable(url)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		break

	}

	bodyBytes := resp.Body()

	err := json.Unmarshal(bodyBytes, target)
	if err != nil {
		log.Error.Println(errors.New(eParser.Wrap(err, 0).ErrorStack()))
		log.Error.Printf("Request body: %v\n", string(requestBody))
		log.Error.Printf("Response: %v\n", string(bodyBytes))
	}
}
