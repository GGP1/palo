package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/GGP1/adak/internal/response"
	"github.com/GGP1/adak/internal/sanitize"
	"github.com/GGP1/adak/internal/token"
	"github.com/GGP1/adak/pkg/shopping/cart"
	"github.com/GGP1/adak/pkg/shopping/ordering"
	"github.com/GGP1/adak/pkg/shopping/payment/stripe"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

// OrderParams holds the parameters for creating a order.
type OrderParams struct {
	Currency string      `json:"currency" validate:"required"`
	Address  string      `json:"address" validate:"required"`
	City     string      `json:"city" validate:"required"`
	Country  string      `json:"country" validate:"required"`
	State    string      `json:"state" validate:"required"`
	ZipCode  string      `json:"zip_code" validate:"required"`
	Date     date        `json:"date" validate:"required"`
	Card     stripe.Card `json:"card" validate:"required"`
}

type date struct {
	Year    int `json:"year" validate:"required,min=2020,max=2100"`
	Month   int `json:"month" validate:"required,min=1,max=12"`
	Day     int `json:"day" validate:"required,min=1,max=31"`
	Hour    int `json:"hour" validate:"required,min=0,max=24"`
	Minutes int `json:"minutes" validate:"required,min=0,max=60"`
}

// OrderingDelete deletes an order.
func (s *API) OrderingDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		_, err := s.orderingClient.Delete(ctx, &ordering.DeleteRequest{OrderID: id})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.HTMLText(w, http.StatusOK, "The order has been deleted successfully.")
	}
}

// OrderingGet finds all the stored orders.
func (s *API) OrderingGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		orders, err := s.orderingClient.Get(ctx, &ordering.GetRequest{})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, orders)
	}
}

// OrderingGetByID retrieves all the orders from the user.
func (s *API) OrderingGetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		order, err := s.orderingClient.GetByID(ctx, &ordering.GetByIDRequest{OrderID: id})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, order)
	}
}

// OrderingGetByUserID retrieves all the orders from the user.
func (s *API) OrderingGetByUserID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uID, _ := r.Cookie("UID")
		ctx := r.Context()

		if err := token.CheckPermits(id, uID.Value); err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		orders, err := s.orderingClient.GetByUserID(ctx, &ordering.GetByUserIDRequest{UserID: id})
		if err != nil {
			response.Error(w, http.StatusNotFound, err)
			return
		}

		response.JSON(w, http.StatusOK, orders)
	}
}

// OrderingNew creates a new order and the payment intent.
func (s *API) OrderingNew() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var oParams OrderParams
		cID, _ := r.Cookie("CID")
		uID, _ := r.Cookie("UID")
		ctx := r.Context()

		if err := json.NewDecoder(r.Body).Decode(&oParams); err != nil {
			response.Error(w, http.StatusBadRequest, err)
		}
		defer r.Body.Close()

		err := validator.New().StructCtx(ctx, oParams)
		if err != nil {
			http.Error(w, err.(validator.ValidationErrors).Error(), http.StatusBadRequest)
			return
		}

		if err := sanitize.Normalize(&oParams.Address, &oParams.City, &oParams.Country, &oParams.Currency, &oParams.State, &oParams.ZipCode); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		// Parse jwt to take the user id
		userID, err := token.ParseFixedJWT(uID.Value)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		// Format date
		deliveryDate := time.Date(oParams.Date.Year, time.Month(oParams.Date.Month), oParams.Date.Day, oParams.Date.Hour, oParams.Date.Minutes, 0, 0, time.Local)

		if deliveryDate.Sub(time.Now()) < 0 {
			response.Error(w, http.StatusBadRequest, errors.New("past dates are not valid"))
			return
		}

		// Fetch the user cart
		cart, err := s.shoppingClient.Get(ctx, &cart.GetRequest{CartID: cID.Value})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		// Create order passing userID, order params, delivery date and the user cart
		order, err := s.orderingClient.New(ctx, &ordering.NewRequest{
			UserID:       userID,
			Currency:     oParams.Currency,
			Address:      oParams.Address,
			City:         oParams.City,
			Country:      oParams.Country,
			State:        oParams.State,
			ZipCode:      oParams.ZipCode,
			DeliveryDate: deliveryDate.Unix(),
			Cart:         cart.Cart,
		})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		// Create payment intent and update the order status
		_, err = stripe.CreateIntent(order.Order.ID, order.Order.CartID, order.Order.Currency, order.Order.Cart.Total, oParams.Card)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		_, err = s.orderingClient.UpdateStatus(ctx, &ordering.UpdateStatusRequest{
			OrderID:     order.Order.ID,
			OrderStatus: string(ordering.PaidState),
		})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		respond := fmt.Sprintf("Thanks for your purchase! Your products will be delivered on %v.", time.Unix(order.Order.DeliveryDate, 0))
		response.HTMLText(w, http.StatusCreated, respond)
	}
}