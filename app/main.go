package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// --- SOAP СТРУКТУРЫ ЗАПРОСА (Перестроены под поиск по email) ---
type SoapEnvelopeRequest struct {
	XMLName xml.Name        `xml:"http://xmlsoap.org Envelope"`
	Body    SoapBodyRequest `xml:"Body"`
}

type SoapBodyRequest struct {
	FetchData FetchSecureEmployeeDataRequest `xml:"fetch_secure_employee_data"`
}

type FetchSecureEmployeeDataRequest struct {
	Email       string `xml:"email"`        // ИСПРАВЛЕНО: Теперь принимаем email вместо user_id
	MasterToken string `xml:"master_token"`
}

// --- SOAP СТРУКТУРЫ ОТВЕТА (Добавлено поле статуса проверки) ---
type SoapEnvelopeResponse struct {
	XMLName   xml.Name         `xml:"soapenv:Envelope"`
	Soapenv   string           `xml:"xmlns:soapenv,attr"`
	Gov       string           `xml:"xmlns:gov,attr"`
	Body      SoapBodyResponse `xml:"soapenv:Body"`
}

type SoapBodyResponse struct {
	Response FetchSecureEmployeeDataResponse `xml:"soapenv:fetch_secure_employee_dataResponse"`
}

type FetchSecureEmployeeDataResponse struct {
	Result GovEmployeeProfile `xml:"gov:fetch_secure_employee_dataResult"`
}

type GovEmployeeProfile struct {
	UserID          int    `xml:"gov:user_id"`
	INN             string `xml:"gov:inn,omitempty"`
	FullName        string `xml:"gov:full_name,omitempty"`
	Passport        string `xml:"gov:passport,omitempty"`
	SNILS           string `xml:"gov:snils,omitempty"`
	SignatureStatus string `xml:"gov:signature_status,omitempty"`
	GovStatus       string `xml:"gov:gov_status,omitempty"` // ДОБАВЛЕНО: Поле для передачи ACTIVE / BLOCKED
	Authorized      bool   `xml:"gov:authorized"`
	ErrorMessage    string `xml:"gov:error_message,omitempty"`
}

var db *sql.DB

// Вспомогательная функция для дефолтных значений переменных окружения
func getEnvOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func initDB() {
	var err error
	// ИСПРАВЛЕНО: Теперь используем getEnvOr для корректной обработки дефолтов
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnvOr("DB_HOST", "localhost"), 
		getEnvOr("DB_PORT", "5432"), 
		getEnvOr("DB_USER", "gov_admin"), 
		getEnvOr("DB_PASSWORD", "gov_secret_pass"), 
		getEnvOr("DB_NAME", "gov_registry_db"),
	)

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Println("⌛ Ожидание готовности гос-базы данных PostgreSQL...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ Ошибка подключения к СУБД: %v", err)
	}
	log.Println("🎯 Успешное подключение к Гос-СУБД Postgres.")
}


func soapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var soapReq SoapEnvelopeRequest
	if err := xml.Unmarshal(bodyBytes, &soapReq); err != nil {
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	reqData := soapReq.Body.FetchData
	profile := GovEmployeeProfile{
		Authorized: false,
	}

	// 1. Проверяем валидность системного токена авторизации самого auth-service
	tokenClean := strings.ToLower(strings.TrimSpace(reqData.MasterToken))
	var isTokenActive bool
	err = db.QueryRow(`SELECT is_active FROM service_access_tokens WHERE token_value = $1;`, tokenClean).Scan(&isTokenActive)

	if err == sql.ErrNoRows || !isTokenActive {
		profile.ErrorMessage = "ACCESS_DENIED: Invalid or suspended master token"
		sendSoapResponse(w, profile)
		return
	} else if err != nil {
		profile.ErrorMessage = fmt.Sprintf("INTERNAL_SERVER_ERROR: %v", err)
		sendSoapResponse(w, profile)
		return
	}

	// 2. ИСПРАВЛЕНО: Делаем запрос к таблице госреестра по email вместо старого user_id
	var id int
	var inn, passportNumber, firstName, lastName, digitalSignatureStatus, insuranceNumber, govStatus string

	emailClean := strings.ToLower(strings.TrimSpace(reqData.Email))
	err = db.QueryRow(`
		SELECT user_id, inn, passport_number, first_name, last_name, digital_signature_status, insurance_number, gov_status 
		FROM digital_employees WHERE email = $1;`, emailClean).
		Scan(&id, &inn, &passportNumber, &firstName, &lastName, &digitalSignatureStatus, &insuranceNumber, &govStatus)

	// ТЕСТ 3: Пользователь отсутствует в госреестре (Сценарий twin.3@example.com)
	if err == sql.ErrNoRows {
		profile.ErrorMessage = "USER_NOT_FOUND"
		sendSoapResponse(w, profile)
		return
	} else if err != nil {
		profile.ErrorMessage = fmt.Sprintf("INTERNAL_SERVER_ERROR: %v", err)
		sendSoapResponse(w, profile)
		return
	}

	// Наполняем структуру ответа данными из Гос-БД
	profile.UserID = id
	profile.INN = inn
	profile.FullName = fmt.Sprintf("%s %s", lastName, firstName)
	profile.Passport = passportNumber
	profile.SNILS = insuranceNumber
	profile.SignatureStatus = digitalSignatureStatus
	profile.GovStatus = govStatus // Передаем ACTIVE или BLOCKED

	// ТЕСТ 2: Профиль найден, но заблокирован государством (Сценарий twin.2@example.com)
	if govStatus == "BLOCKED" {
		profile.Authorized = false
		profile.ErrorMessage = "GOV_ACCOUNT_BLOCKED"
	} else {
		// Все проверки пройдены успешно (Сценарии mobile.user и twin.1)
		profile.Authorized = true
		profile.ErrorMessage = "SUCCESS"
	}

	sendSoapResponse(w, profile)
}

func sendSoapResponse(w http.ResponseWriter, profile GovEmployeeProfile) {
	soapResp := SoapEnvelopeResponse{
		Soapenv: "http://xmlsoap.org",
		Gov:     "gov.service.m4",
		Body: SoapBodyResponse{
			Response: FetchSecureEmployeeDataResponse{
				Result: profile,
			},
		},
	}

	outputXML, err := xml.MarshalIndent(soapResp, "", "  ")
	if err != nil {
		http.Error(w, "XML generation error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xml.Header))
	w.Write(outputXML)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/", soapHandler)

	log.Println("🏛️  Гос SOAP-сервер на Go запущен на порту :8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Ошибка старта сервера: %v", err)
	}
}
