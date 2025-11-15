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
	Members  []struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	} `json:"members"`
}

type teamMemberResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type teamResponse struct {
	TeamName string               `json:"team_name"`
	Members  []teamMemberResponse `json:"members"`
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

	if params.TeamName == "" {
		RespondWithError(w, "BAD_REQUEST", "team_name is required", http.StatusBadRequest)
		return
	}

	_, err := config.ApiCfg.DB.GetTeamByName(ctx, params.TeamName)
	if err == nil {
		RespondWithError(w, "TEAM_EXISTS", fmt.Sprintf("%s already exists", params.TeamName), http.StatusBadRequest)
		log.Printf("team already exists, %v", err)
		return
	}

	teamID := uuid.NewString()

	err = config.ApiCfg.DB.CreateTeam(ctx, database.CreateTeamParams{
		ID:       teamID,
		Teamname: params.TeamName,
	})

	if err != nil {
		http.Error(w, "could not create a team", http.StatusInternalServerError)
		log.Printf("error creating a team, %v", err)
		return
	}

	membersResp := make([]teamMemberResponse, 0, len(params.Members))

	for _, m := range params.Members {
		if m.UserID == "" {
			RespondWithError(w, "BAD_REQUEST", "invalid user id", http.StatusBadRequest)
			return
		}

		err := config.ApiCfg.DB.UpsertUser(ctx, database.UpsertUserParams{
			ID:       m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
			TeamID:   teamID,
		})

		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to upsert user", http.StatusInternalServerError)
			log.Printf("error upserting user, %v", err)
			return
		}

		membersResp = append(membersResp, teamMemberResponse{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	resp := map[string]any{
		"team": teamResponse{
			TeamName: params.TeamName,
			Members:  membersResp,
		},
	}

	RespondWithJSON(w, http.StatusCreated, resp)
}

func GetTeamHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		RespondWithError(w, "BAD_REQUEST", "team name is required", http.StatusBadRequest)
		return
	}

	team, err := config.ApiCfg.DB.GetTeamByName(ctx, teamName)
	if err != nil {
		RespondWithError(w, "NOT_FOUND", "team not found", http.StatusNotFound)
		log.Printf("team not found: %v", err)
		return
	}

	users, err := config.ApiCfg.DB.GetUsersByTeam(ctx, team.ID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "could not get team users", http.StatusInternalServerError)
		log.Printf("error fetching users for team: %v", err)
		return
	}

	resp := teamResponse{
		TeamName: team.Teamname,
		Members:  make([]teamMemberResponse, 0, len(users)),
	}

	for _, u := range users {
		resp.Members = append(resp.Members, teamMemberResponse{
			UserID:   u.ID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}

	RespondWithJSON(w, http.StatusOK, resp)
}
