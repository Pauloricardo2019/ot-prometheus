package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	url := "http://localhost:8989/product" // URL do endpoint /product

	// Definindo a estrutura do payload JSON
	product := struct {
		Product string `json:"product"`
	}{
		Product: "example_product_id",
	}

	// Criando um cliente HTTP
	client := http.Client{}

	// Loop infinito para enviar requisições repetidamente
	for {
		// Convertendo struct para JSON
		jsonData, err := json.Marshal(product)
		if err != nil {
			log.Fatalf("Erro ao codificar JSON: %v", err)
		}

		// Criando uma requisição POST com o payload JSON
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Fatalf("Erro ao criar requisição HTTP: %v", err)
		}

		// Definindo o tipo de conteúdo do cabeçalho como aplicação/json
		req.Header.Set("Content-Type", "application/json")

		// Realizando a requisição HTTP
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Erro ao enviar requisição HTTP: %v", err)
		}
		defer resp.Body.Close()

		// Verificando o código de status da resposta
		if resp.StatusCode != http.StatusOK {
			log.Printf("Erro na resposta da API: %s", resp.Status)
		} else {
			log.Printf("Requisição bem-sucedida. Status: %s", resp.Status)
		}

		// Aguardando 1 segundo antes de enviar a próxima requisição
		time.Sleep(1 * time.Second)
	}
}
