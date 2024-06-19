package api_lib_client

import (
	"log"
)

func main() {
	api := NewAPIRetry()

	apiURL := "https://my.api.mockaroo.com/consents/urn:nubank:660adfc8-331f-3ae0-9060-5ce0b629b3e9?key=9fcc6cd0"

	timeRetry := []int{5, 10, 15, 20}

	headers := map[string]string{
		"Authorization": "Bearer token que temos aqui.",
		"Content-Type":  "application/json",
	}

	body, err := api.Get(apiURL, timeRetry, headers)
	if err != nil {
		log.Println("Erro ao fazer a chamada de API (GET):", err)
		return
	}

	log.Println("Resposta da API (GET):", string(body))

}
