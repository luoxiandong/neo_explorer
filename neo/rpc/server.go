package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"neo_explorer/core/config"
	"neo_explorer/core/util"
	"strings"
	"sync"
	"time"
)

var (
	// servers stores all neo rpc urls with its height
	// For those (temporarily)unaccessable servers,
	// their height will be set to -1.
	// These servers' heights will be refreshed timely.
	servers map[string]int
	sLock   sync.RWMutex

	// BestHeight indicates current highest height.
	BestHeight util.SafeCounter
)

type ServerInfo struct {
	url    string
	height int
}

func TraceBestHeight() {

	RefreshServers()
	PrintServerStatus()

	for {
		RefreshServers()

		time.Sleep(3 * time.Second)
	}

}

// RefreshServers updates heights of all rpc servers.
func RefreshServers() int {
	// It takes time to get heights.
	serverInfos := getHeights()

	sLock.Lock()

	servers = serverInfos
	bestHeight := 0
	for _, height := range serverInfos {
		if bestHeight < height {
			bestHeight = height
		}
	}
	BestHeight.Set(bestHeight)

	sLock.Unlock()

	return bestHeight
}

func serverUnavailable(url string) {
	sLock.Lock()
	defer sLock.Unlock()

	// Incase server changed(e.g., reloaded dut to config file change).
	if _, ok := servers[url]; ok {
		servers[url] = -1
	}
}

// PrintServerStatus prints rpc host with its current best height.
func PrintServerStatus() {
	sLock.RLock()
	defer sLock.RUnlock()

	for host, height := range servers {
		if height < 0 {
			fmt.Printf("%s: %s\n", host, util.BRed("Failed to get server height"))
		} else {
			fmt.Printf("%s: %d\n", host, height)
		}
	}
}

// getHeights gets current height of all rpc servers
// and returns best height from these servers.
func getHeights() map[string]int {
	// log.Printf("Checking all rpc servers...")
	rpcs := config.GetRPCs()
	c := make(chan ServerInfo, len(rpcs))

	for _, url := range rpcs {
		go func(url string, c chan<- ServerInfo) {
			height, _ := getHeightFrom(url)
			c <- ServerInfo{
				url:    url,
				height: height,
			}
		}(url, c)
	}

	serverInfos := make(map[string]int)

	for range rpcs {
		s := <-c
		serverInfos[s.url] = s.height
	}

	close(c)

	return serverInfos
}

// getServer randomly returns one of rpc servers whose height higher than minHeight.
func getServer(minHeight int) (string, bool) {
	if minHeight < 0 {
		err := fmt.Errorf("minHeight(%d) cannot lower than zero", minHeight)
		panic(err)
	}

	sLock.RLock()
	defer sLock.RUnlock()

	// Suppose all servers are qualified.
	candidates := []string{}

	for url, height := range servers {
		if height >= int(minHeight) {
			// Always select localhost rpc server if valid.
			if strings.Contains(url, "127.0.0.1") ||
				strings.Contains(url, "localhost") {
				candidates = append(candidates, url)
			}

			candidates = append(candidates, url)
		}
	}

	l := len(candidates)
	if l == 0 {
		return "", false
	}

	return candidates[rand.Intn(l)], true
}

// getHeightFrom returns current block index of the given rpc server.
func getHeightFrom(url string) (int, error) {
	params := []interface{}{}
	args := getRPCRequestBody("getblockcount", params)

	respData := BlockCountRespponse{}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer([]byte(args)))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&respData)

	return respData.Result - 1, nil
}
