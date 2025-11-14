package handlers

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/LlirikP/pr_dispenser/internal/config"
	"github.com/LlirikP/pr_dispenser/internal/database"
	"github.com/google/uuid"
)

type createPRRequest struct {
	Title    string `json:"title"`
	AuthorID string `json:"author_id"`
}

type assignReviewerRequest struct {
	PrID       string `json:"pr_id"`
	ReviewerID string `json:"reviewer_id"`
}

type mergePRRequest struct {
	PrID string `json:"pr_id"`
}

func CreatePRHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	params := createPRRequest{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		log.Printf("error parsing json, %v", err)
		return
	}

	author, err := config.ApiCfg.DB.GetUserById(ctx, params.AuthorID)
	if err != nil {
		RespondWithError(w, "USER_NOT_FOUND", "author not found", http.StatusNotFound)
		log.Printf("error finding user, %v", err)
		return
	}

	_, err = config.ApiCfg.DB.CheckDuplicatePR(ctx, database.CheckDuplicatePRParams{
		AuthorID: params.AuthorID,
		Title:    params.Title,
	})

	if err == nil {
		RespondWithError(w, "PR_EXISTS", "PR with this title and author already exists and is open", http.StatusBadRequest)
		log.Printf("error creating pr due to duplicating, %v", err)
		return
	}

	prID := uuid.NewString()

	err = config.ApiCfg.DB.CreatePR(ctx, database.CreatePRParams{
		ID:       prID,
		Title:    params.Title,
		AuthorID: params.AuthorID,
	})

	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to create PR", http.StatusInternalServerError)
		log.Printf("error creating pr, %v", err)
		return
	}

	teammates, err := config.ApiCfg.DB.GetActiveTeamMembersExceptAuthor(ctx, database.GetActiveTeamMembersExceptAuthorParams{
		TeamID: author.TeamID,
		ID: author.ID,
	})

	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to get team members", http.StatusInternalServerError)
		log.Printf("error getting teammates: %v", err)
		return
	}

	rand.Shuffle(len(teammates), func(i, j int) {
    	teammates[i], teammates[j] = teammates[j], teammates[i]
	})
	
	count := 2
	if len(teammates) < 2 {
		count = len(teammates)
	}

	assigned := make([]string, 0, count)

	for i := 0; i < count; i++ {
		reviewerID := teammates[i]

		err = config.ApiCfg.DB.AddReviewer(ctx, database.AddReviewerParams{
			PrID:       prID,
			ReviewerID: reviewerID,
		})

		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to assign reviewer", http.StatusInternalServerError)
			log.Printf("error assigning reviewers, %v", err)
			return
		}

		err = config.ApiCfg.DB.SetUserIsActive(ctx, database.SetUserIsActiveParams{
			ID:       reviewerID,
			IsActive: false,
		})
		
		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to update reviewer status", http.StatusInternalServerError)
			log.Printf("error updating reviewer status, %v", err)
			return
		}

		assigned = append(assigned, reviewerID)
	}

	resp := struct {
		PRID              string   `json:"pr_id"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	}{
		PRID:              prID,
		AssignedReviewers: assigned,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func AssignReviewerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var params assignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		return
	}

	_, err := config.ApiCfg.DB.GetPRById(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "PR_NOT_FOUND", "unknown PR", http.StatusNotFound)
		return
	}

	_, err = config.ApiCfg.DB.GetUserById(ctx, params.ReviewerID)
	if err != nil {
		RespondWithError(w, "USER_NOT_FOUND", "unknown reviewer", http.StatusNotFound)
		return
	}

	err = config.ApiCfg.DB.AddReviewer(ctx, database.AddReviewerParams{
		PrID:       params.PrID,
		ReviewerID: params.ReviewerID,
	})
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to assign reviewer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func MergePRHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var params mergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		return
	}

	_, err := config.ApiCfg.DB.GetPRById(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "PR_NOT_FOUND", "unknown PR", http.StatusNotFound)
		return
	}

	err = config.ApiCfg.DB.MergePR(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to merge PR", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
