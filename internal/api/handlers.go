package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tanq16/expenseowl/internal/config"
	"github.com/tanq16/expenseowl/internal/storage"
	"github.com/tanq16/expenseowl/internal/web"
)

type Handler struct {
	storage storage.Storage
	config  *config.Config
}

func NewHandler(s storage.Storage, cfg *config.Config) *Handler {
	return &Handler{
		storage: s,
		config:  cfg,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ExpenseRequest struct {
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
}

type ConfigResponse struct {
	Categories []string `json:"categories"`
	Currency   string   `json:"currency"`
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	response := ConfigResponse{
		Categories: h.config.Categories,
		Currency:   h.config.Currency,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) EditCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	var categories []string
	if err := json.NewDecoder(r.Body).Decode(&categories); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		log.Printf("HTTP ERROR: Failed to decode request body: %v\n", err)
		return
	}
	h.config.UpdateCategories(categories)
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
	log.Println("HTTP: Updated categories")
}

func (h *Handler) EditCurrency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	var currency string
	if err := json.NewDecoder(r.Body).Decode(&currency); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		log.Printf("HTTP ERROR: Failed to decode request body: %v\n", err)
		return
	}
	h.config.UpdateCurrency(currency)
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
	log.Println("HTTP: Updated currency")
}

func (h *Handler) AddExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	var req ExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		log.Printf("HTTP ERROR: Failed to decode request body: %v\n", err)
		return
	}
	if !req.Date.IsZero() {
		req.Date = req.Date.UTC()
	}
	expense := &config.Expense{
		Name:     req.Name,
		Category: req.Category,
		Amount:   req.Amount,
		Date:     req.Date,
	}
	if err := expense.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		log.Printf("HTTP ERROR: Failed to validate expense: %v\n", err)
		return
	}
	if err := h.storage.SaveExpense(expense); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to save expense"})
		log.Printf("HTTP ERROR: Failed to save expense: %v\n", err)
		return
	}
	writeJSON(w, http.StatusOK, expense)
}

func (h *Handler) GetExpenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	expenses, err := h.storage.GetAllExpenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
		log.Printf("HTTP ERROR: Failed to retrieve expenses: %v\n", err)
		return
	}
	writeJSON(w, http.StatusOK, expenses)
}

func (h *Handler) ServeTableView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := web.ServeTemplate(w, "table.html"); err != nil {
		http.Error(w, "Failed to serve template", http.StatusInternalServerError)
		log.Printf("HTTP ERROR: Failed to serve template: %v\n", err)
		return
	}
}

func (h *Handler) ServeSettingsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := web.ServeTemplate(w, "settings.html"); err != nil {
		http.Error(w, "Failed to serve template", http.StatusInternalServerError)
		log.Printf("HTTP ERROR: Failed to serve template: %v\n", err)
		return
	}
}

func (h *Handler) ServeAPISetupView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := web.ServeTemplate(w, "api-setup.html"); err != nil {
		http.Error(w, "Failed to serve template", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID parameter is required"})
		log.Println("HTTP ERROR: ID parameter is required")
		return
	}
	if err := h.storage.DeleteExpense(id); err != nil {
		if err == storage.ErrExpenseNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Expense not found"})
			log.Printf("HTTP ERROR: Expense not found: %v\n", err)
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete expense"})
		log.Printf("HTTP ERROR: Failed to delete expense: %v\n", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
	log.Printf("HTTP: Deleted expense with ID %s\n", id)
}

// Static Handler
func (h *Handler) ServeStaticFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	if err := web.ServeStatic(w, r.URL.Path); err != nil {
		http.Error(w, "Failed to serve static file", http.StatusInternalServerError)
		log.Printf("HTTP ERROR: Failed to serve static file %s: %v\n", r.URL.Path, err)
		return
	}
}

// Export handlers
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	expenses, err := h.storage.GetAllExpenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
		log.Printf("HTTP ERROR: Failed to retrieve expenses: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=expenses.csv")
	// write CSV data
	w.Write([]byte("ID,Name,Category,Amount,Date\n"))
	for _, expense := range expenses {
		line := fmt.Sprintf("%s,%s,%s,%.2f,%s\n",
			expense.ID,
			strings.ReplaceAll(expense.Name, ",", ";"), // Replace , in name with ;
			expense.Category,
			expense.Amount,
			expense.Date.Format("2006-01-02 15:04:05"),
		)
		w.Write([]byte(line))
	}
	log.Println("HTTP: Exported expenses to CSV")
}

func (h *Handler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Println("HTTP ERROR: Method not allowed")
		return
	}
	expenses, err := h.storage.GetAllExpenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
		log.Printf("HTTP ERROR: Failed to retrieve expenses: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=expenses.json")
	// Pretty print the JSON data for better readability
	jsonData, err := json.MarshalIndent(expenses, "", "    ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON data", http.StatusInternalServerError)
		log.Printf("HTTP ERROR: Failed to marshal JSON data: %v\n", err)
		return
	}
	w.Write(jsonData)
	log.Println("HTTP: Exported expenses to JSON")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
