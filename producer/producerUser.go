package producer

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"ot-prometheus/models"
	"time"
)

func ProducerUser() {
	userPool := []string{"bob", "alice", "jack", "mike", "tiger", "panda", "dog"}
	for {
		postBody, _ := json.Marshal(models.User{
			User: userPool[rand.Intn(len(userPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		http.Post("http://0.0.0.0:8989/user", "application/json", requestBody)
		time.Sleep(time.Second * 2)
	}
}
