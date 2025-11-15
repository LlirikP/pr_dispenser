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
)

type createPRRequest struct {
	PrID     string `json:"pull_request_id"`
	Title    string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

type assignReviewerRequest struct {
	PrID       string `json:"pull_request_id"`
	ReviewerID string `json:"old_user_id"`
}

type mergePRRequest struct {
	PrID string `json:"pull_request_id"`
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
		RespondWithError(w, "PR_EXISTS", "PR already exists", http.StatusConflict)
		log.Printf("error creating pr due to duplicating, %v", err)
		return
	}

	err = config.ApiCfg.DB.CreatePR(ctx, database.CreatePRParams{
		ID:       params.PrID,
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
		ID:     author.ID,
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
			PrID:       params.PrID,
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
		PRID              string   `json:"pull_request_id"`
		Title             string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	}{
		PRID:              params.PrID,
		Title:             params.Title,
		AuthorID:          params.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: assigned,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)

	if err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

func AssignReviewerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	params := assignReviewerRequest{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		return
	}

	pr, err := config.ApiCfg.DB.GetPRById(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "PR_NOT_FOUND", "unknown PR", http.StatusNotFound)
		return
	}

	if pr.Status == "MERGED" {
		RespondWithError(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
		return
	}

	reviewer, err := config.ApiCfg.DB.GetUserById(ctx, params.ReviewerID)
	if err != nil {
		RespondWithError(w, "USER_NOT_FOUND", "user not found", http.StatusNotFound)
		log.Printf("error finding user, %v", err)
		return
	}

	assigned, err := config.ApiCfg.DB.IsReviewerAssigned(ctx, database.IsReviewerAssignedParams{
		PrID:       params.PrID,
		ReviewerID: params.ReviewerID,
	})

	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to check reviewer assignment", http.StatusInternalServerError)
		return
	}

	if !assigned {
		RespondWithError(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
		return
	}

	teammates, err := config.ApiCfg.DB.GetActiveTeamMembersExceptAuthor(
		ctx,
		database.GetActiveTeamMembersExceptAuthorParams{
			TeamID: reviewer.TeamID,
			ID:     reviewer.ID,
		},
	)

	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to load team members", http.StatusInternalServerError)
		return
	}

	newReviewerID := ""

	for _, candidate := range teammates {
		cid := candidate

		if cid == reviewer.ID {
			continue
		}

		alreadyAssigned, err := config.ApiCfg.DB.IsReviewerAssigned(ctx, database.IsReviewerAssignedParams{
			PrID:       params.PrID,
			ReviewerID: cid,
		})
		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to check candidate", http.StatusInternalServerError)
			return
		}

		if alreadyAssigned {
			continue
		}

		newReviewerID = cid
		break
	}

	if newReviewerID == "" {
		RespondWithError(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
		return
	}

	err = config.ApiCfg.DB.DeleteReviewer(ctx, database.DeleteReviewerParams{
		PrID:       params.PrID,
		ReviewerID: params.ReviewerID,
	})
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to remove old reviewer", http.StatusInternalServerError)
		return
	}

	err = config.ApiCfg.DB.AddReviewer(ctx, database.AddReviewerParams{
		PrID:       params.PrID,
		ReviewerID: newReviewerID,
	})
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to assign new reviewer", http.StatusInternalServerError)
		return
	}

	reviewers, err := config.ApiCfg.DB.GetReviewersByPR(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to load reviewers", http.StatusInternalServerError)
		return
	}

	resp := struct {
		PR struct {
			ID                string   `json:"pull_request_id"`
			Title             string   `json:"pull_request_name"`
			AuthorID          string   `json:"author_id"`
			Status            string   `json:"status"`
			AssignedReviewers []string `json:"assigned_reviewers"`
		} `json:"pr"`
		ReplacedBy string `json:"replaced_by"`
	}{}

	resp.PR.ID = pr.ID
	resp.PR.Title = pr.Title
	resp.PR.AuthorID = pr.AuthorID
	resp.PR.Status = pr.Status
	resp.PR.AssignedReviewers = reviewers
	resp.ReplacedBy = newReviewerID

	RespondWithJSON(w, http.StatusOK, resp)
	w.WriteHeader(http.StatusOK)
}

func MergePRHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	params := mergePRRequest{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondWithError(w, "BAD_JSON", "invalid json", http.StatusBadRequest)
		log.Printf("error parsing json, %v", err)
		return
	}

	pr, err := config.ApiCfg.DB.GetPRById(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "PR_NOT_FOUND", "unknown PR", http.StatusNotFound)
		log.Printf("error finding pr, %v", err)
		return
	}

	if pr.Status == "MERGED" {
		reviewers, err := config.ApiCfg.DB.GetReviewersByPR(ctx, params.PrID)
		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to load reviewers", http.StatusInternalServerError)
			log.Printf("error loading reviewers, %v", err)
			return
		}

		resp := map[string]any{
			"pr": map[string]any{
				"pull_request_id":    pr.ID,
				"pull_request_name":  pr.Title,
				"author_id":          pr.AuthorID,
				"status":             pr.Status,
				"assigned_reviewers": reviewers,
				"mergedAt":           pr.MergedAt,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Printf("error encoding response: %v", err)
		}
		return
	}

	err = config.ApiCfg.DB.MergePR(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to merge PR", http.StatusInternalServerError)
		log.Printf("error merging pr, %v", err)
		return
	}

	reviewers, err := config.ApiCfg.DB.GetReviewersByPR(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to load reviewers", http.StatusInternalServerError)
		log.Printf("error loading reviewers, %v", err)
		return
	}

	for _, reviewerID := range reviewers {
		err = config.ApiCfg.DB.SetUserIsActive(ctx, database.SetUserIsActiveParams{
			ID:       reviewerID,
			IsActive: true,
		})

		if err != nil {
			RespondWithError(w, "DB_ERROR", "failed to change reviewer's status", http.StatusInternalServerError)
			log.Printf("error changind status, %v", err)
			return
		}
	}

	updatedPR, err := config.ApiCfg.DB.GetPRById(ctx, params.PrID)
	if err != nil {
		RespondWithError(w, "DB_ERROR", "failed to reload PR", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"pr": map[string]any{
			"pull_request_id":    updatedPR.ID,
			"pull_request_name":  updatedPR.Title,
			"author_id":          updatedPR.AuthorID,
			"status":             updatedPR.Status,
			"assigned_reviewers": reviewers,
			"mergedAt":           updatedPR.MergedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("error encoding response: %v", err)
	}
}
