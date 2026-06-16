package lets

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed templates/*.html
var templateFS embed.FS

type App struct {
	db        *gorm.DB
	templates *template.Template
	now       func() time.Time
}

type LunchRequest struct {
	ID                uint      `gorm:"primaryKey"`
	SelectedDate      time.Time `gorm:"uniqueIndex"`
	Contact           string
	SuggestedLocation string
	CreatedAt         time.Time
}

type WednesdayDate struct {
	Value             string
	Label             string
	Reserved          bool
	Contact           string
	SuggestedLocation string
}

type pageData struct {
	Dates             []WednesdayDate
	SelectedDate      string
	Contact           string
	SuggestedLocation string
	Error             string
	Saved             bool
}

func OpenSQLite(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&LunchRequest{}); err != nil {
		return nil, err
	}
	return db, nil
}

func NewApp(db *gorm.DB) (*App, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &App{
		db:        db,
		templates: tmpl,
		now:       time.Now,
	}, nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("POST /requests", a.handleCreateRequest)
	return mux
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	dates, err := a.calendarDates()
	if err != nil {
		http.Error(w, "Could not load lunch calendar.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		Dates: dates,
		Saved: r.URL.Query().Get("saved") == "1",
	}
	a.renderIndex(w, data)
}

func (a *App) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission.", http.StatusBadRequest)
		return
	}

	dates, err := a.calendarDates()
	if err != nil {
		http.Error(w, "Could not load lunch calendar.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		Dates:             dates,
		SelectedDate:      strings.TrimSpace(r.FormValue("date")),
		Contact:           strings.TrimSpace(r.FormValue("contact")),
		SuggestedLocation: strings.TrimSpace(r.FormValue("suggested_location")),
	}

	selected, err := validateSubmission(data.Contact, data.SelectedDate, dates)
	if err != nil {
		data.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		a.renderIndex(w, data)
		return
	}

	request := LunchRequest{
		SelectedDate:      selected,
		Contact:           data.Contact,
		SuggestedLocation: data.SuggestedLocation,
	}
	if err := a.db.Create(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "unique") {
			data.Error = "That Wednesday was already reserved."
			data.Dates, _ = a.calendarDates()
			w.WriteHeader(http.StatusConflict)
			a.renderIndex(w, data)
			return
		}
		http.Error(w, "Could not save your lunch request.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/?saved=1", http.StatusSeeOther)
}

func (a *App) renderIndex(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Could not render page.", http.StatusInternalServerError)
	}
}

func (a *App) calendarDates() ([]WednesdayDate, error) {
	dates := currentWednesdays(a.now())
	if len(dates) == 0 {
		return dates, nil
	}

	start, err := time.ParseInLocation("2006-01-02", dates[0].Value, time.Local)
	if err != nil {
		return nil, err
	}
	end, err := time.ParseInLocation("2006-01-02", dates[len(dates)-1].Value, time.Local)
	if err != nil {
		return nil, err
	}

	var requests []LunchRequest
	if err := a.db.Where("selected_date >= ? AND selected_date <= ?", start, end).Order("created_at asc").Find(&requests).Error; err != nil {
		return nil, err
	}

	reserved := make(map[string]LunchRequest, len(requests))
	for _, request := range requests {
		key := request.SelectedDate.Format("2006-01-02")
		if _, exists := reserved[key]; !exists {
			reserved[key] = request
		}
	}

	for i := range dates {
		if request, exists := reserved[dates[i].Value]; exists {
			dates[i].Reserved = true
			dates[i].Contact = request.Contact
			dates[i].SuggestedLocation = request.SuggestedLocation
		}
	}

	return dates, nil
}

func currentWednesdays(now time.Time) []WednesdayDate {
	return wednesdaysBetween(now, now.AddDate(0, 2, 0))
}

func wednesdaysBetween(start, end time.Time) []WednesdayDate {
	loc := start.Location()
	day := dateOnly(start.In(loc))
	last := dateOnly(end.In(loc))

	for day.Weekday() != time.Wednesday {
		day = day.AddDate(0, 0, 1)
	}

	var dates []WednesdayDate
	for !day.After(last) {
		dates = append(dates, WednesdayDate{
			Value: day.Format("2006-01-02"),
			Label: day.Format("Wednesday, Jan 2, 2006"),
		})
		day = day.AddDate(0, 0, 7)
	}
	return dates
}

func validateSubmission(contact, selectedDate string, dates []WednesdayDate) (time.Time, error) {
	contact = strings.TrimSpace(contact)
	selectedDate = strings.TrimSpace(selectedDate)

	if contact == "" {
		return time.Time{}, fmt.Errorf("Please enter your name or email.")
	}
	if selectedDate == "" {
		return time.Time{}, fmt.Errorf("Please choose a Wednesday.")
	}

	for _, date := range dates {
		if selectedDate == date.Value {
			if date.Reserved {
				return time.Time{}, fmt.Errorf("That Wednesday is already reserved.")
			}
			parsed, err := time.ParseInLocation("2006-01-02", selectedDate, time.Local)
			if err != nil {
				return time.Time{}, fmt.Errorf("Please choose a valid Wednesday.")
			}
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("Please choose one of the available Wednesdays.")
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
