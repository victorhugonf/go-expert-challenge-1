package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shopspring/decimal"
)

const findExchangeRateTimeout = 300 * time.Millisecond

type ExchangeRate struct {
	Bid decimal.Decimal `json:"bid"`
}

func main() {
	exchangeRate, err := findExchangeRate()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = saveExchangeRate(exchangeRate)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func findExchangeRate() (*ExchangeRate, error) {
	res, err := findExchangeRateLocalhost()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New("erro ao buscar cotacao")
	}
	defer res.Body.Close()

	exchangeRate, err := readExchangeRate(res.Body)
	if err != nil {
		return nil, err
	}
	return exchangeRate, nil
}

func findExchangeRateLocalhost() (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), findExchangeRateTimeout)
	defer cancel()
	res, err := findExchangeRateLocalhostWithContext(ctx)

	if ctx.Err() != nil {
		return nil, errors.New("timeout ao buscar cotacao")
	}
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("erro ao buscar cotacao")
	}
	return res, err
}

func findExchangeRateLocalhostWithContext(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func readExchangeRate(body io.Reader) (*ExchangeRate, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var exchangeRate ExchangeRate
	err = json.Unmarshal(data, &exchangeRate)
	if err != nil {
		return nil, err
	}
	return &exchangeRate, nil
}

func saveExchangeRate(exchangeRate *ExchangeRate) error {
	f, err := os.Create("./cotacao.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintln("Dólar: ", exchangeRate.Bid))
	if err != nil {
		panic(err)
	}
	return nil
}
