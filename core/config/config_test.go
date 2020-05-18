package config

import (
	"fmt"
	"testing"
)

func TestLoad(t *testing.T) {
	Load()
	fmt.Printf("%+v", cfg)
}
