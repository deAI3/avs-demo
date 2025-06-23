package operator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const API_KEY = "1547eae6-c29f-464a-ac29-e27fe94db841"

func getCMCPrice(symbol string) float64 {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://pro-api.coinmarketcap.com/v2/cryptocurrency/quotes/latest", nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	q := url.Values{}
	q.Add("symbol", symbol)
	q.Add("convert", "USD")

	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", API_KEY)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request to server")
		os.Exit(1)
	}
	fmt.Println(resp.Status)
	respBody, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	err = json.Unmarshal(respBody, &res)
	if err != nil {
		fmt.Println("Error unmarshal price: ", err)
		os.Exit(1)
	}
	fmt.Println("data: ", res)

	data := res["data"].(map[string]interface{})
	symbolData := data[symbol].([]interface{})

	fmt.Println("Price of ", symbol, ":", symbolData[0].(map[string]interface{})["quote"].(map[string]interface{})["USD"].(map[string]interface{})["price"])
	return symbolData[0].(map[string]interface{})["quote"].(map[string]interface{})["USD"].(map[string]interface{})["price"].(float64)
}
