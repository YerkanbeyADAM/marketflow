// internal/ports/outbound/exchange_test.go
package livesource

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"marketflow/internal/app/model"

	"github.com/stretchr/testify/assert"
)

func startFakeExchangeServer(t *testing.T, addr string, data []model.MarketData) {
	ln, err := net.Listen("tcp", addr)
	assert.NoError(t, err)

	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		assert.NoError(t, err)
		defer conn.Close()

		writer := bufio.NewWriter(conn)
		for _, d := range data {
			b, _ := json.Marshal(d)
			writer.Write(b)
			writer.WriteString("\n")
			writer.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

func TestRealDataSource_Stream(t *testing.T) {
	addr := "127.0.0.1:41000"

	fakeData := []model.MarketData{
		{Symbol: "BTCUSDT", Price: 50000.1, Timestamp: time.Now()},
		{Symbol: "ETHUSDT", Price: 3000.5, Timestamp: time.Now()},
	}

	startFakeExchangeServer(t, addr, fakeData)

	source := &RealDataSource{
		Exchanges: []string{addr},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	out := make(chan model.MarketData, 10)

	err := source.Stream(ctx, out)
	assert.NoError(t, err)

	collected := []model.MarketData{}
	for len(out) > 0 {
		collected = append(collected, <-out)
	}

	assert.GreaterOrEqual(t, len(collected), 2)
	assert.Equal(t, "BTCUSDT", collected[0].Symbol)
}
