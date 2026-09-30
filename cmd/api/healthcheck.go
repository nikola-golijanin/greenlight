package main

import (
	"net/http"
)

type SystemInfo map[string]string

type HealthCheck struct {
	Status     string     `json:"status"`
	SystemInfo SystemInfo `json:"system_info"`
}

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := HealthCheck{
		Status: "available",
		SystemInfo: map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	err := app.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
		return
	}
}
