package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/LlirikP/pr_dispenser/internal/config"
	"github.com/LlirikP/pr_dispenser/internal/database"
	"github.com/google/uuid"
)

type createTeamRequest struct {
	TeamName string `json:"team_name"`
}

func CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	params := createTeamRequest{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		log.Printf("error parsing json, %v", err)
		return
	}
	
	_, err := config.ApiCfg.DB.GetTeamByName(ctx, params.TeamName)
	if err != nil {
		RespondWithError(w, "TEAM_EXISTS", fmt.Sprintf("%s already exists", params.TeamName), http.StatusBadRequest)
		log.Printf("team already exists, %v", err)
		return
	}

	id := uuid.New()

	err = config.ApiCfg.DB.CreateTeam(ctx, database.CreateTeamParams{
		ID: id,
		Teamname: params.TeamName,
	})

	if err != nil {
		http.Error(w, "could not create a team", http.StatusInternalServerError)
		log.Printf("error creating a team, %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func GetTeamHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "invalid team name", http.StatusBadRequest)
		return
	}

	team, err := config.ApiCfg.DB.GetTeamByName(r.Context(), teamName)
	if err != nil {
		RespondWithError(w, "NOT_FOUND", "team not found", http.StatusNotFound)
		return
	}
}