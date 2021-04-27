package rest

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/adak/internal/response"
	"github.com/GGP1/adak/internal/sanitize"
	"github.com/GGP1/adak/pkg/shop"
	"github.com/go-playground/validator/v10"

	"github.com/go-chi/chi"
)

// ShopCreate creates a new shop and saves it.
func (s *API) ShopCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sp shop.Shop
		ctx := r.Context()

		if err := json.NewDecoder(r.Body).Decode(&sp); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validator.New().StructCtx(ctx, &sp); err != nil {
			http.Error(w, err.(validator.ValidationErrors).Error(), http.StatusBadRequest)
			return
		}

		_, err := s.shopClient.Create(ctx, &shop.CreateRequest{Shop: &sp})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusCreated, &sp)
	}
}

// ShopDelete removes a shop.
func (s *API) ShopDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		_, err := s.shopClient.Delete(ctx, &shop.DeleteRequest{ID: id})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.HTMLText(w, http.StatusOK, "Shop deleted successfully.")
	}
}

// ShopGet lists all the shops.
func (s *API) ShopGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		get, err := s.shopClient.Get(ctx, &shop.GetRequest{})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, get.Shops)
	}
}

// ShopGetByID lists the shop with the id requested.
func (s *API) ShopGetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		getByID, err := s.shopClient.GetByID(ctx, &shop.GetByIDRequest{ID: id})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, getByID.Shop)
	}
}

// ShopSearch looks for the products with the given value.
func (s *API) ShopSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := chi.URLParam(r, "query")
		ctx := r.Context()

		if err := sanitize.Normalize(&query); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		search, err := s.shopClient.Search(ctx, &shop.SearchRequest{Search: query})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, search.Shops)
	}
}

// ShopUpdate updates the shop with the given id.
func (s *API) ShopUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sp shop.Shop
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		if err := json.NewDecoder(r.Body).Decode(&sp); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		_, err := s.shopClient.Update(ctx, &shop.UpdateRequest{Shop: &sp, ID: id})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.HTMLText(w, http.StatusOK, "Shop updated successfully.")
	}
}