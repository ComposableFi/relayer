package substrate_test

import (
	"context"
	"testing"
)

func TestQueryLatestHeight(t *testing.T) {
	p, err := getTestProvider()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	height, err := p.QueryLatestHeight(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	if height <= 0 {
		t.Errorf("latest height should be greater than genesis height")
	}
}
