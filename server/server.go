package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const findExchangeRateTimeout = 200 * time.Millisecond
const saveExchangeRateTimeout = 10 * time.Millisecond

type ExchangeRate struct {
	UUID uuid.UUID `gorm:"primaryKey"`
	Bid  decimal.Decimal
	gorm.Model
}

func (exchangeRate *ExchangeRate) ToExchangeRateResponse() *ExchangeRateResponse {
	return &ExchangeRateResponse{Bid: exchangeRate.Bid}
}

type ExchangeRateResponse struct {
	Bid decimal.Decimal `json:"bid"`
}

type ExchangeRateAwesomeapi struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

func (exchangeRate *ExchangeRateAwesomeapi) ToExchangeRate() (*ExchangeRate, error) {
	bid, err := decimal.NewFromString(exchangeRate.USDBRL.Bid)
	if err != nil {
		return nil, err
	}
	return &ExchangeRate{UUID: uuid.New(), Bid: bid}, nil
}

func main() {
	err := dbMigrate()
	if err != nil {
		panic(err.Error())
	}
	http.HandleFunc("/cotacao", exchangeRateHandler)
	http.ListenAndServe(":8080", nil)
}

func exchangeRateHandler(w http.ResponseWriter, r *http.Request) {
	exchangeRate, err := findExchangeRate()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(exchangeRate.ToExchangeRateResponse())
}

func findExchangeRate() (*ExchangeRate, error) {
	res, err := findExchangeRateInAwesomeapi()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	exchangeRate, err := readExchangeRate(res.Body)
	if err != nil {
		return nil, err
	}

	err = saveExchangeRate(exchangeRate)
	if err != nil {
		return nil, err
	}

	return exchangeRate, nil
}

func findExchangeRateInAwesomeapi() (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), findExchangeRateTimeout)
	defer cancel()
	res, err := findExchangeRateInAwesomeapiWithContext(ctx)

	if ctx.Err() != nil {
		return nil, errors.New("timeout ao buscar cotacao")
	}
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("erro ao buscar cotacao")
	}
	return res, err
}

func findExchangeRateInAwesomeapiWithContext(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
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
	var exchangeRate ExchangeRateAwesomeapi
	err = json.Unmarshal(data, &exchangeRate)
	if err != nil {
		return nil, err
	}
	return exchangeRate.ToExchangeRate()
}

func saveExchangeRate(exchangeRate *ExchangeRate) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), saveExchangeRateTimeout)
	defer cancel()

	db.WithContext(ctx).Create(&exchangeRate)

	if ctx.Err() != nil {
		return errors.New("timeout ao salvar cotacao")
	}
	if err != nil {
		log.Println(err.Error())
		return errors.New("erro ao salvar cotacao")
	}
	return nil
}

func dbOpen() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
}

func dbMigrate() error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&ExchangeRate{})
	if err != nil {
		return err
	}
	return nil
}
