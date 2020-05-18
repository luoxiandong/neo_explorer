package rpc

import (
	"fmt"
	"neo_explorer/core/config"
	"testing"
)

func Test_getHeightFrom(t *testing.T) {
	count, err := getHeightFrom("https://explorer.o3node.org:443")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(count)
}

func Test_getHeights(t *testing.T) {
	config.Load()
	heights := getHeights()

	for url, height := range heights {
		fmt.Printf("%s : %d\n", url, height)
	}
}
