package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/url"
)

type mongodbStatus struct {
	baseStatus
	bson.M
}

func MongodbStatus(healthURL *url.URL, client *mongo.Client) (out mongodbStatus, err error) {
	// Send a ping to confirm a successful connection
	var result bson.M
	err = client.Database("admin").RunCommand(context.Background(), bson.D{{"buildInfo", 1}}).Decode(&result)
	if err != nil {
		return
	}
	out = mongodbStatus{
		baseStatus: baseStatus{},
		M:          result,
	}
	/*
		client := &http.Client{}
		log.Println("Getting health at: ", healthURL.String())
		req, err := http.NewRequest("GET", healthURL.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				log.AddError(err).Error("while closing response body")
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		name := strings.TrimPrefix(string(st), "website_")
		status := "DOWN"
		if strings.Contains(strings.ToLower(string(body)), name) {
			status = "UP"
		}
		out = baseStatus{
			Status:  status,
			Name:    name,
			Version: version,
			IP:      GetOutboundIP(),
			Now:     time.Now(),
		}
	*/
	return
}
