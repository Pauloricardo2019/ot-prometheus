package producer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"ot-prometheus/models"
	"time"
)

func ProducerProduct() {
	productPool := []string{"camiseta", "blusa", "calça", "jaqueta", "camisa"}
	for {
		postBody, _ := json.Marshal(models.Product{
			Product: productPool[rand.Intn(len(productPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		_, err := http.Post("http://0.0.0.0:8989/product", "application/json", requestBody)
		if err != nil {
			fmt.Println("error on send post product", err)
		}
		time.Sleep(time.Second * 2)
	}
}
