package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Product struct {
	ID             int    `json:"id"`
	SKU            string `json:"sku"`
	QuantityOnHand int    `json:"quantity_on_hand"`
}

type Order struct {
	ID        int `json:"id"`
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type Server struct {
	db *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func (s *Server) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if p.QuantityOnHand < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quantity cannot be negative"})
		return
	}

	query := `
		INSERT INTO products (sku, quantity_on_hand)
		VALUES ($1, $2)
		RETURNING id
	`

	err := s.db.QueryRow(query, p.SKU, p.QuantityOnHand).Scan(&p.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if o.Quantity <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quantity must be positive"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE products
		SET quantity_on_hand = quantity_on_hand - $1
		WHERE id = $2
		AND quantity_on_hand >= $1
	`, o.Quantity, o.ProductID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inventory update failed"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to verify inventory"})
		return
	}

	if rowsAffected == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient inventory"})
		return
	}

	err = tx.QueryRow(`
		INSERT INTO orders (product_id, quantity)
		VALUES ($1, $2)
		RETURNING id
	`, o.ProductID, o.Quantity).Scan(&o.ID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "order creation failed"})
		return
	}

	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transaction commit failed"})
		return
	}

	writeJSON(w, http.StatusCreated, o)
}
