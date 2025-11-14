package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/LlirikP/pr_dispenser/internal/config"
	"github.com/LlirikP/pr_dispenser/internal/database"
)

type setUserActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type reviewListResponse struct {
	Items []reviewListItem `json:"items"`
}

type reviewListItem struct {
	PrID     string `json:"pr_id"`
	Title    string `json:"title"`
	AuthorID string `json:"author_id"`
	Status   string `json:"status"`
}

func SetUserActiveHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	params := setUserActiveRequest{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		log.Printf("decode err: %v", err)
		return
	}

	if params.UserID == "" {
		RespondWithError(w, "BAD_REQUEST", "user_id required", http.StatusBadRequest)
		return
	}

	_, err := config.ApiCfg.DB.GetUserById(ctx, params.UserID)
	if err != nil {
		RespondWithError(w, "USER_NOT_FOUND", "unknown user", http.StatusNotFound)
		log.Printf("error finding user: %v", err)
		return
	}

	err = config.ApiCfg.DB.SetUserIsActive(ctx, database.SetUserIsActiveParams{
		ID:       params.UserID,
		IsActive: params.IsActive,
	})

	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to update user", http.StatusInternalServerError)
		log.Printf("error updating user: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ReviewListHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		RespondWithError(w, "BAD_REQUEST", "missing user id", http.StatusBadRequest)
		return
	}

	_, err := config.ApiCfg.DB.GetUserById(ctx, userID)
	if err != nil {
		RespondWithError(w, "USER_NOT_FOUND", "unknown user", http.StatusNotFound)
		return
	}

	reviews, err := config.ApiCfg.DB.GetReviewPRs(ctx, userID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to load reviews", http.StatusInternalServerError)
		log.Printf("error loading reviews, %v", err)
		return
	}

	resp := reviewListResponse{}

	for _, r := range reviews {
		resp.Items = append(resp.Items, reviewListItem{
			PrID:     r.PrID,
			Title:    r.PrTitle,
			AuthorID: r.AuthorID,
			Status:   r.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
