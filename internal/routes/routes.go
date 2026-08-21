package routes

import (
	"net/http"

	adhkarHandler "github.com/ilmnafi/backend/internal/adhkar/handler"
	authHandler "github.com/ilmnafi/backend/internal/auth/handler"
	"github.com/ilmnafi/backend/internal/health"
	"github.com/ilmnafi/backend/internal/middleware"
	quranHandler "github.com/ilmnafi/backend/internal/quran/handler"
	userHandler "github.com/ilmnafi/backend/internal/user/handler"
	"github.com/gorilla/mux"
)

func SetupRoutes(
	authHdl *authHandler.AuthHandler,
	userHdl *userHandler.UserHandler,
	quranHdl *quranHandler.QuranHandler,
	adhkarHdl *adhkarHandler.AdhkarHandler,
	authMiddleware func(http.Handler) http.Handler,
	rateLimiter *middleware.RateLimiter,
	healthChecker *health.HealthChecker,
) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.SecurityHeadersMiddleware)

	r.HandleFunc("/health", healthChecker.Check).Methods("GET")
	r.HandleFunc("/ready", healthChecker.Ready).Methods("GET")

	api := r.PathPrefix("/auth").Subrouter()
	api.HandleFunc("/register", authHdl.Register).Methods("POST")
	api.Handle("/login", rateLimiter.Middleware(http.HandlerFunc(authHdl.Login))).Methods("POST")
	api.HandleFunc("/logout", authHdl.Logout).Methods("POST")
	api.HandleFunc("/refresh", authHdl.Refresh).Methods("POST")
	api.HandleFunc("/verify-email", authHdl.VerifyEmail).Methods("POST")
	api.Handle("/resend-verification", rateLimiter.Middleware(http.HandlerFunc(authHdl.ResendVerification))).Methods("POST")
	api.Handle("/forgot-password", rateLimiter.Middleware(http.HandlerFunc(authHdl.ForgotPassword))).Methods("POST")
	api.HandleFunc("/reset-password", authHdl.ResetPassword).Methods("POST")

	protected := r.PathPrefix("/auth").Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("/me", authHdl.GetMe).Methods("GET")
	protected.HandleFunc("/sessions", authHdl.GetSessions).Methods("GET")
	protected.HandleFunc("/delete-account", authHdl.DeleteAccount).Methods("DELETE")

	userAPI := r.PathPrefix("/user").Subrouter()
	userAPI.Use(authMiddleware)
	userAPI.HandleFunc("/me", userHdl.GetMe).Methods("GET")
	userAPI.HandleFunc("/profile", userHdl.UpdateProfile).Methods("PUT")
	userAPI.HandleFunc("/delete-account", userHdl.DeleteAccount).Methods("DELETE")
	userAPI.HandleFunc("/sessions", userHdl.GetSessions).Methods("GET")

	quran := r.PathPrefix("/quran").Subrouter()
	quran.HandleFunc("/surahs", quranHdl.GetSurahs).Methods("GET")
	quran.HandleFunc("/surahs/{id}", quranHdl.GetSurah).Methods("GET")
	quran.HandleFunc("/surahs/{id}/ayahs", quranHdl.GetSurahAyahs).Methods("GET")
	quran.HandleFunc("/ayahs/{id}", quranHdl.GetAyah).Methods("GET")
	quran.HandleFunc("/juz/{id}", quranHdl.GetJuz).Methods("GET")
	quran.HandleFunc("/search", quranHdl.Search).Methods("GET")
	quran.HandleFunc("/reciters", quranHdl.GetReciters).Methods("GET")
	quran.HandleFunc("/reciters/{id}", quranHdl.GetReciter).Methods("GET")
	quran.HandleFunc("/audio/surah/{surahID}", quranHdl.GetSurahAudio).Methods("GET")
	quran.HandleFunc("/audio/ayah/{ayahID}", quranHdl.GetAyahAudio).Methods("GET")

	quranProtected := r.PathPrefix("/quran").Subrouter()
	quranProtected.Use(authMiddleware)
	quranProtected.HandleFunc("/bookmarks", quranHdl.GetBookmarks).Methods("GET")
	quranProtected.HandleFunc("/bookmarks", quranHdl.CreateBookmark).Methods("POST")
	quranProtected.HandleFunc("/bookmarks/{ayahID}", quranHdl.DeleteBookmark).Methods("DELETE")
	quranProtected.HandleFunc("/progress", quranHdl.GetProgress).Methods("GET")
	quranProtected.HandleFunc("/progress", quranHdl.UpdateProgress).Methods("PUT")
	quranProtected.HandleFunc("/continue", quranHdl.ContinueReading).Methods("GET")

	adhkar := r.PathPrefix("/adhkar").Subrouter()
	adhkar.HandleFunc("/categories", adhkarHdl.GetCategories).Methods("GET")
	adhkar.HandleFunc("/categories/{id}", adhkarHdl.GetCategory).Methods("GET")
	adhkar.HandleFunc("/categories/{id}/adhkar", adhkarHdl.GetAdhkarByCategory).Methods("GET")
	adhkar.HandleFunc("/{slug}", adhkarHdl.GetAdhkarBySlug).Methods("GET")
	adhkar.HandleFunc("/{id}", adhkarHdl.GetDhikr).Methods("GET")

	adhkarProtected := r.PathPrefix("/adhkar").Subrouter()
	adhkarProtected.Use(authMiddleware)
	adhkarProtected.HandleFunc("/{id}/complete", adhkarHdl.RecordCompletion).Methods("POST")
	adhkarProtected.HandleFunc("/progress/today", adhkarHdl.GetDailyProgress).Methods("GET")
	adhkarProtected.HandleFunc("/progress/history", adhkarHdl.GetProgressHistory).Methods("GET")
	adhkarProtected.HandleFunc("/reminders", adhkarHdl.GetReminders).Methods("GET")
	adhkarProtected.HandleFunc("/reminders", adhkarHdl.CreateReminder).Methods("POST")
	adhkarProtected.HandleFunc("/reminders/{id}", adhkarHdl.UpdateReminder).Methods("PUT")
	adhkarProtected.HandleFunc("/reminders/{id}", adhkarHdl.DeleteReminder).Methods("DELETE")

	return r
}
