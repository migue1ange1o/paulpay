package main

import (
	"database/sql"
	"encoding/base64"

	//"encoding/hex"
	"encoding/json"
	"fmt"

	//	"github.com/davecgh/go-spew/spew"
	"html"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"shadowchat/internal/models"
	"shadowchat/utils"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	//"github.com/realclientip/realclientip-go"
)

const username = "admin"

var USDMinimum float64 = 5
var MediaMin float64 = 0.025 // Currently unused
var MessageMaxChar int = 250
var NameMaxChar int = 25
var starting_port int = 28088

var host_url string = "https://ferret.cash/"

var checked string = ""
var killDono = 35.00 * time.Minute // hours it takes for a dono to be unfulfilled before it is no longer checked.
var indexTemplate *template.Template
var overflowTemplate *template.Template
var tosTemplate *template.Template
var registerTemplate *template.Template
var donationTemplate *template.Template
var payTemplate *template.Template

var alertTemplate *template.Template
var accountPayTemplate *template.Template
var billPayTemplate *template.Template
var progressbarTemplate *template.Template
var userOBSTemplate *template.Template
var viewTemplate *template.Template

var loginTemplate *template.Template
var footerTemplate *template.Template
var incorrectLoginTemplate *template.Template
var userTemplate *template.Template
var cryptoSettingsTemplate *template.Template
var logoutTemplate *template.Template
var incorrectPasswordTemplate *template.Template
var baseCheckingRate = 25

var minSolana, minMonero, minEthereum, minPaint, minHex, minPolygon, minBusd, minShib, minUsdc, minTusd, minWbtc, minPnk float64 // Global variables to hold minimum values required to equal the global value.
var minDonoValue float64 = 5.0

var PublicRegistrationsEnabled = false

var ServerMinMediaDono = 5
var ServerMediaEnabled = true

var db *sql.DB
var amountNeeded = 1000.00
var amountSent = 200.00

var a utils.AlertPageData
var pb utils.ProgressbarData
var obsData utils.OBSDataStruct

var prices utils.CryptoPrice

var pbMessage = "Stream Tomorrow"

type Route_ struct {
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
}

var routes_ []Route_

// Define a new template that only contains the table content
var tableTemplate = template.Must(template.New("table").Parse(`
	{{range .}}
	<tr id="{{.ID}}">
                    <td>
                        <button onclick="replayDono('{{.ID}}')">Replay</button>
                    </td>
                    <td>{{.UpdatedAt.Format "15:04:05 01-02-2006"}}</td>
                    <td>{{.Name}}</td>
                    <td>{{.Message}}</td>
                    <td>{{.MediaURL}}</td>
                    <td>${{.USDAmount}}</td>
                    <td>{{.AmountSent}}</td>
                    <td>{{.CurrencyType}}</td>
                </tr>
	{{end}}
`))

func checkLoggedIn(w http.ResponseWriter, r *http.Request, ur *models.UserRepository) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_, valid := ur.GetUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
}

func checkLoggedInAdmin(w http.ResponseWriter, r *http.Request, ur *models.UserRepository) bool {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return false
	}

	user, valid := ur.GetUserBySessionCached(cookie.Value)
	if !valid {
		return false
	}

	if user.Username == "admin" {
		return true
	} else {
		return false
	}
}

// Handler function for the "/donations" endpoint
func donationsHandler(w http.ResponseWriter, r *http.Request, dr *models.DonoRepository) {
	log.Println("donationsHandler Called")

	cookie, err := r.Cookie("session_token")
	if err != nil {
		return
	}

	user, valid := dr.UserRepo.GetUserBySessionCached(cookie.Value)
	if !valid {
		return
	}
	// Fetch the latest data from your database or other data source

	// Retrieve data from the donos table
	rows, err := dr.Db.Query("SELECT * FROM donos WHERE fulfilled = 1 AND amount_sent != '0.0' ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create a slice to hold the data
	var donos []utils.Dono
	for rows.Next() {
		var dono utils.Dono
		var name, message, address, currencyType, encryptedIP, amountToSend, amountSent, mediaURL sql.NullString
		var usdAmount sql.NullFloat64
		var userID sql.NullInt64
		var anonDono, fulfilled sql.NullBool
		err := rows.Scan(&dono.ID, &userID, &address, &name, &message, &amountToSend, &amountSent, &currencyType, &anonDono, &fulfilled, &encryptedIP, &dono.CreatedAt, &dono.UpdatedAt, &usdAmount, &mediaURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dono.UserID = int(userID.Int64)
		dono.Address = address.String
		dono.Name = name.String
		dono.Message = message.String
		dono.AmountToSend = amountToSend.String
		dono.AmountSent = amountSent.String
		dono.CurrencyType = currencyType.String
		dono.AnonDono = anonDono.Bool
		dono.Fulfilled = fulfilled.Bool
		dono.EncryptedIP = encryptedIP.String
		dono.USDAmount = usdAmount.Float64
		dono.MediaURL = mediaURL.String
		if dono.UserID == user.UserID {
			if s, err := strconv.ParseFloat(dono.AmountSent, 64); err == nil {
				if s > 0 {
					donos = append(donos, dono)
				}
			}
		}
	}

	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := donos

	// Execute the table template with the latest data
	err = tableTemplate.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {

	var err error

	// Open a new database connection
	db, err = sql.Open("sqlite3", "users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// set up repositories
	sr := models.NewSolRepository(db)
	ir := models.NewInviteRepository(db)
	br := models.NewBillingRepository(db)
	ur := models.NewUserRepository(db, sr, br)
	mr := models.NewMoneroRepository(db, ur)
	dr := models.NewDonoRepository(db, ur, mr)

	// Check if the database and tables exist, and create them if they don't
	err = models.CreateDatabaseIfNotExists(db, ur)
	if err != nil {
		log.Printf("Error creating database: %v", err)
	}

	// Run migrations on database
	err = models.RunDatabaseMigrations(db)
	if err != nil {
		log.Printf("Error migrating database: %v", err)
	}

	go models.StartWallets(ur, mr, sr)

	time.Sleep(5 * time.Second)
	log.Println("Starting server")

	setupRoutes(ur)

	time.Sleep(2 * time.Second)
	// Schedule a function to run fetchExchangeRates every three minutes
	go models.FetchExchangeRates(ur)
	go dr.Check()
	go models.CheckPendingAccounts(dr)
	go models.CheckBillingAccounts(dr)

	go dr.CheckAccountBillings()

	a.Refresh = 10
	pb.Refresh = 1
	_ = ur.GetObsData(1)
	inviteCodeMap = ir.GetAllCodes()
	models.SetServerVars(ur)

	//go createTestDono(2, "Big Bob", "XMR", "This Cruel Message is Bob's Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! ", "50", 100, "https://www.youtube.com/watch?v=6iseNlvH2_s")
	// go createTestDono("Medium Bob", "XMR", "Hey it's medium Bob ", 0.1, 3, "https://www.youtube.com/watch?v=6iseNlvH2_s")

	err = http.ListenAndServe(":8900", nil)
	if err != nil {
		panic(err)
	}

}

func updateCryptosHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var updateRequest utils.UpdateCryptosRequest
	err = json.Unmarshal(body, &updateRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, err := strconv.Atoi(updateRequest.UserID)

	log.Println(userID, user.UserID)

	if userID == user.UserID {
		user.CryptosEnabled = mapToCryptosEnabled(updateRequest.SelectedCryptos)
		if user.CryptosEnabled.XMR && !user.WalletUploaded {
			user.CryptosEnabled.XMR = false
		}
		log.Println(user.CryptosEnabled)
		err = updateUser(user)
		if err != nil {
			log.Println(err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func mapToCryptosEnabled(selectedCryptos map[string]bool) utils.CryptosEnabled {
	// Create a new instance of the CryptosEnabled struct
	cryptosEnabled := utils.CryptosEnabled{}

	// Set each field of the CryptosEnabled struct based on the corresponding value in the map
	cryptosEnabled.XMR = selectedCryptos["monero"]
	cryptosEnabled.SOL = selectedCryptos["solana"]
	cryptosEnabled.ETH = selectedCryptos["ethereum"]
	cryptosEnabled.PAINT = selectedCryptos["paint"]
	cryptosEnabled.HEX = selectedCryptos["hex"]
	cryptosEnabled.MATIC = selectedCryptos["matic"]
	cryptosEnabled.BUSD = selectedCryptos["busd"]
	cryptosEnabled.SHIB = selectedCryptos["shiba_inu"]
	cryptosEnabled.PNK = selectedCryptos["pnk"]

	// Return the populated CryptosEnabled struct
	return cryptosEnabled
}
func setupRoutes(ur *models.UserRepository) {
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/css/style.css")
	})

	http.HandleFunc("/xmr.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/xmr.svg")
	})

	http.HandleFunc("/bignumber.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/js/bignumber.js")
	})

	http.HandleFunc("/checkmark.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/xmr.png")
	})

	http.HandleFunc("/fcash.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/fcash.png")
	})

	http.HandleFunc("/indexfcash.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/indexfcash.png")
	})

	http.HandleFunc("/loader.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/loader.svg")
	})

	http.HandleFunc("/eth.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/eth.svg")
	})

	http.HandleFunc("/sol.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/sol.svg")
	})

	http.HandleFunc("/busd.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/busd.svg")
	})

	http.HandleFunc("/hex.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/hex.svg")
	})

	http.HandleFunc("/matic.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/matic.svg")
	})

	http.HandleFunc("/paint.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/paint.svg")
	})

	http.HandleFunc("/pnk.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/pnk.svg")
	})

	http.HandleFunc("/shiba_inu.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/shiba_inu.svg")
	})

	http.HandleFunc("/tether.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/tether.svg")
	})

	http.HandleFunc("/usdc.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/usdc.svg")
	})

	http.HandleFunc("/wbtc.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/img/wbtc.svg")
	})

	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir("web/obs/media/"))))
	http.Handle("/users/", http.StripPrefix("/users/", http.FileServer(http.Dir("users/"))))

	routes_ = []Route_{
		{"/updatecryptos", updateCryptosHandler},
		{"/update-links", updateLinksHandler},
		{"/check_donation_status/", checkDonationStatusHandler},
		{"/donations", donationsHandler},
		{"/", indexHandler},
		{"/termsofservice", tosHandler},
		{"/pay", paymentHandler},
		{"/alert", alertOBSHandler},
		{"/viewdonos", viewDonosHandler},
		{"/replaydono", replayDonoHandler},
		{"/progressbar", progressbarOBSHandler},
		{"/login", func(w http.ResponseWriter, r *http.Request) {
			loginHandler(w, r, ur)
		}},
		{"/incorrect_login", incorrectLoginHandler},
		{"/user", userHandler},
		{"/userobs", userOBSHandler},
		{"/logout", logoutHandler},
		{"/changepassword", changePasswordHandler},
		{"/changeuser", changeUserHandler},
		{"/register", registerUserHandler},
		{"/newaccount", newAccountHandler},
		{"/overflow", overflowHandler},
		{"/billing", accountBillingHandler},
		{"/changeusermonero", changeUserMoneroHandler},
		{"/usermanager", allUsersHandler},
		{"/refresh", refreshHandler},
		{"/toggleUserRegistrations", toggleUserRegistrationsHandler},
		{"/generatecodes", generateCodesHandler},
		{"/cryptosettings", cryptoSettingsHandler},
	}

	for _, route_ := range routes_ {
		http.HandleFunc(route_.Path, logging(route_.Handler))
	}

	indexTemplate, _ = template.ParseFiles("web/templates/index.html")

	overflowTemplate, _ = template.ParseFiles("web/overflow.html")
	tosTemplate, _ = template.ParseFiles("web/templates/tos.html")
	registerTemplate, _ = template.ParseFiles("web/templates/new_account.html")
	donationTemplate, _ = template.ParseFiles("web/templates/donation.html")
	footerTemplate, _ = template.ParseFiles("web/templates/footer.html")
	payTemplate, _ = template.ParseFiles("web/templates/pay.html")
	alertTemplate, _ = template.ParseFiles("web/templates/alert.html")
	accountPayTemplate, _ = template.ParseFiles("web/templates/accountpay.html")

	billPayTemplate, _ = template.ParseFiles("web/templates/billpay.html")

	userOBSTemplate, _ = template.ParseFiles("web/templates/obs/settings.html")
	progressbarTemplate, _ = template.ParseFiles("web/templates/obs/progressbar.html")

	loginTemplate, _ = template.ParseFiles("web/templates/login.html")
	incorrectLoginTemplate, _ = template.ParseFiles("web/templates/incorrect_login.html")
	userTemplate, _ = template.ParseFiles("web/templates/user.html")
	cryptoSettingsTemplate, _ = template.ParseFiles("web/templates/cryptoselect.html")

	logoutTemplate, _ = template.ParseFiles("web/templates/logout.html")
	incorrectPasswordTemplate, _ = template.ParseFiles("web/templates/password_change_failed.html")
}

func logging(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/overflow" && r.URL.Path != "/progressbar" && r.URL.Path != "/donations" && r.URL.Path != "/check_donation_status" && r.URL.Path != "/replaydono" && r.URL.Path != "/viewdonos" {
			ip := getIPAddress(r)
			matchingIP := CheckRecentIPRequests(ip)
			if matchingIP >= 200 {
				http.Redirect(w, r, "/overflow", http.StatusSeeOther)
				return
			}
		}
		f(w, r)
	}
}

func replayDonoHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, valid := getLoggedInUser(w, r)

	var donation utils.Donation
	err := json.NewDecoder(r.Body).Decode(&donation)
	if err != nil {
		fmt.Printf("Error decoding JSON")
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	// Process the donation information as needed
	fmt.Printf("Received donation replay: %+v\n", donation)

	if valid {
		replayDono(donation, user.UserID)
	} else {
		http.Error(w, "Invalid donation trying to be replayed", http.StatusBadRequest)
		return
	}

	// Send response indicating success
	w.WriteHeader(http.StatusOK)
}

func checkValidSubscription(DateEnabled time.Time) bool {
	oneMonthAhead := DateEnabled.AddDate(0, 1, 0)
	if oneMonthAhead.After(time.Now().UTC()) {
		log.Println("User valid")
		return true
	}
	log.Println("checkValidSubscription() User not valid")
	return false
}

func getLoggedInUser(w http.ResponseWriter, r *http.Request) (utils.User, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return utils.User{}, false // Return an instance of utils.User with empty/default values
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		return utils.User{}, false // Return an instance of utils.User with empty/default values
	}

	return user, true
}

func allUsersHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if user.Username == "admin" {

		// Define the data to be passed to the HTML template
		data := struct {
			Title            string
			RegistrationOpen bool
			Users            map[int]utils.User
			InviteCodes      map[string]utils.InviteCode
		}{
			Title:            "Users Dashboard",
			RegistrationOpen: PublicRegistrationsEnabled,
			Users:            globalUsers,
			InviteCodes:      inviteCodeMap,
		}

		// Parse the HTML template and execute it with the data
		tmpl, err := template.ParseFiles("web/templates/view_users.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}

func generateCodesHandler(w http.ResponseWriter, r *http.Request) {
	if checkLoggedInAdmin(w, r) {
		generateMoreInviteCodes(5)
		http.Redirect(w, r, "/usermanager", http.StatusSeeOther)
		allUsersHandler(w, r)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

}

func toggleUserRegistrationsHandler(w http.ResponseWriter, r *http.Request) {

	if checkLoggedInAdmin(w, r) {
		PublicRegistrationsEnabled = !PublicRegistrationsEnabled
		http.Redirect(w, r, "/usermanager", http.StatusSeeOther)
		allUsersHandler(w, r)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if checkLoggedInAdmin(w, r) {
		user, _ := getUserByUsernameCached(r.FormValue("username"))
		renewUserSubscription(user)
	}
	allUsersHandler(w, r)
}

func viewDonosHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cookie = cookie

	// Retrieve data from the donos table
	rows, err := db.Query("SELECT * FROM donos WHERE fulfilled = 1 AND amount_sent != 0 ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create a slice to hold the data
	var donos []utils.Dono
	for rows.Next() {
		var dono utils.Dono
		var name, amountToSend, amountSent, message, address, currencyType, encryptedIP, mediaURL sql.NullString
		var usdAmount sql.NullFloat64
		var userID sql.NullInt64
		var anonDono, fulfilled sql.NullBool
		err := rows.Scan(&dono.ID, &userID, &address, &name, &message, &amountToSend, &amountSent, &currencyType, &anonDono, &fulfilled, &encryptedIP, &dono.CreatedAt, &dono.UpdatedAt, &usdAmount, &mediaURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dono.UserID = int(userID.Int64)
		dono.Address = address.String
		dono.Name = name.String
		dono.Message = message.String
		dono.AmountToSend = amountToSend.String
		dono.AmountSent = amountSent.String
		dono.CurrencyType = currencyType.String
		dono.AnonDono = anonDono.Bool
		dono.Fulfilled = fulfilled.Bool
		dono.EncryptedIP = encryptedIP.String
		dono.USDAmount = usdAmount.Float64
		dono.MediaURL = mediaURL.String
		if dono.UserID == user.UserID {
			if s, err := strconv.ParseFloat(dono.AmountSent, 64); err == nil {
				if s > 0 {
					donos = append(donos, dono)
				}
			}
		}
	}

	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort the data based on user input
	sortParam := r.FormValue("sort")
	switch sortParam {
	case "date":
		sort.Slice(donos, func(i, j int) bool {
			return donos[i].UpdatedAt.Before(donos[j].UpdatedAt)
		})
	case "amount":
		sort.Slice(donos, func(i, j int) bool {
			return donos[i].USDAmount < donos[j].USDAmount
		})
	}

	// Send the data to the template
	tpl, err := template.ParseFiles("web/templates/view_donos.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := utils.ViewDonosData{
		Username: user.Username,
		Donos:    donos,
	}
	tpl.Execute(w, data)
}

func createNewEthDono(name string, message string, mediaURL string, amountNeeded float64, cryptoCode string, encrypted_ip string) utils.SuperChat {
	new_dono := utils.CreatePendingDono(name, message, mediaURL, amountNeeded, cryptoCode, encrypted_ip)
	pending_donos = utils.AppendPendingDono(pending_donos, new_dono)

	return new_dono
}

// verify that the entered password matches the stored hashed password for a user
func verifyPassword(user utils.User, password string) bool {
	err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	return err == nil
}

func loginHandler(w http.ResponseWriter, r *http.Request, ur *models.UserRepository) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		username = strings.ToLower(username)
		password := r.FormValue("password")

		user, valid := ur.GetUserByUsernameCached(username)

		if !valid {
			log.Println("Can't find username")
			http.Redirect(w, r, "/incorrect_login", http.StatusFound)
			return

		}

		if user.UserID == 0 || !ur.VerifyPassword(user, password) {
			http.Redirect(w, r, "/incorrect_login", http.StatusFound)
			return
		}

		sessionToken, err := createSession(user.UserID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
			Secure:   true,
		})
		http.Redirect(w, r, "/user", http.StatusFound)
		return
	}
	tmpl := template.Must(template.ParseFiles("web/templates/login.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func userOBSHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	obsData.URLdonobar = host_url + "progressbar?value=" + user.AlertURL
	obsData.URLdisplay = host_url + "alert?value=" + user.AlertURL
	obsData_ := getObsData(db, user.UserID)
	obsData_.Username = user.Username

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(5 << 10) // max file size of 10 MB
		userDir := fmt.Sprintf("users/%d/", user.UserID)

		// Get the files from the request
		fileGIF, handlerGIF, err := r.FormFile("dono_animation")
		if err == nil {
			defer fileGIF.Close()
			fileNameGIF := handlerGIF.Filename
			fileBytesGIF, err := ioutil.ReadAll(fileGIF)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = os.WriteFile(userDir+"/gifs/default.gif", fileBytesGIF, 0644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			obsData_.FilenameGIF = fileNameGIF
		}

		fileMP3, handlerMP3, err := r.FormFile("dono_sound")
		if err == nil {
			defer fileMP3.Close()
			fileNameMP3 := handlerMP3.Filename
			fileBytesMP3, err := ioutil.ReadAll(fileMP3)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = os.WriteFile(userDir+"/sounds/default.mp3", fileBytesMP3, 0644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			obsData_.FilenameMP3 = fileNameMP3
		}

		pbMessage = r.FormValue("message")

		amountNeededStr := r.FormValue("needed")

		amountSentStr := r.FormValue("sent")

		amountNeeded, err = strconv.ParseFloat(amountNeededStr, 64)
		if err != nil {
			// handle the error
			log.Println(err)
		}

		amountSent, err = strconv.ParseFloat(amountSentStr, 64)
		if err != nil {
			// handle the error
			log.Println(err)
		}

		obsData_.Message = pbMessage
		obsData_.Needed = amountNeeded
		obsData_.Sent = amountSent

		pb.Message = pbMessage
		pb.Needed = amountNeeded
		pb.Sent = amountSent

		err = updateObsData(db, user.UserID, obsData_.FilenameGIF, obsData_.FilenameMP3, "alice", pb)

		if err != nil {
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {

	}

	log.Println(obsData_.Message)
	log.Println(obsData_.Needed)
	log.Println(obsData_.Sent)
	obsData_.URLdonobar = host_url + "progressbar?value=" + user.AlertURL
	obsData_.URLdisplay = host_url + "alert?value=" + user.AlertURL
	log.Println(obsData.URLdonobar)
	log.Println(obsData.URLdisplay)

	tmpl, err := template.ParseFiles("web/templates/obs/settings.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, obsData_)

}

// handle requests to modify user data
func userHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, valid := getUserBySessionCached(cookie.Value)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		user.EthAddress = r.FormValue("ethaddress")
		user.SolAddress = r.FormValue("soladdress")
		user.HexcoinAddress = r.FormValue("hexcoinaddress")
		user.XMRWalletPassword = r.FormValue("xmrwalletpassword")
		user.MinDono, _ = strconv.Atoi(r.FormValue("mindono"))
		user.MinMediaDono, _ = strconv.Atoi(r.FormValue("minmediadono"))
		mediaEnabled := r.FormValue("mediaenabled") == "on"
		user.MediaEnabled = mediaEnabled

		user = setUserMinDonos(user)
		err := updateUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}

	if user.Links == "" {
		user.Links = "[]"
	}

	data := struct {
		User  utils.User
		Links string // Changed to string to hold JSON
	}{
		User:  user,
		Links: user.Links, // Convert byte slice to string
	}

	tmpl, err := template.ParseFiles("web/templates/user.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)

}

func updateLinksHandler(w http.ResponseWriter, r *http.Request) {
	// Get the links parameter from the POST request
	linksJson := r.PostFormValue("links")
	username := r.PostFormValue("username")

	cookie, _ := r.Cookie("session_token")
	user, _ := getUserBySessionCached(cookie.Value)

	if user.Username == username {

		// Parse the JSON string into a slice of Link structs
		var links []utils.Link
		err := json.Unmarshal([]byte(linksJson), &links)
		if err != nil {
			// Handle error
			return
		}

		user.Links = linksJson
		updateUser(user)
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/user", http.StatusSeeOther)
	return
}

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	// retrieve user from session
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	user, valid := getUserBySessionCached(sessionToken.Value)
	if !valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	// initialize user page data struct
	data := utils.UserPageData{}

	// process form submission
	if r.Method == "POST" {
		// check current password
		if !verifyPassword(user, r.FormValue("current_password")) {
			// set user page data values
			data.ErrorMessage = "Current password entered was incorrect"
			// render password change failed form
			tmpl, err := template.ParseFiles("web/templates/password_change_failed.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		} else {
			// hash new password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("new_password")), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// update user password in database
			user.HashedPassword = hashedPassword
			err = updateUser(user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// redirect to user page
			http.Redirect(w, r, "/user", http.StatusSeeOther)
			return
		}
	}

	// render change password form
	tmpl, err := template.ParseFiles("web/templates/user.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func changeUserMoneroHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting change user handler function")
	// retrieve user from session
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	user, valid := getUserBySessionCached(sessionToken.Value)
	if !valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// initialize user page data struct
	data := utils.UserPageData{}

	// process form submission
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if user.WalletUploaded {
			stopMoneroWallet(user)
		}

		// Get the uploaded monero wallet file and save it to disk
		moneroDir := fmt.Sprintf("users/%d/monero", user.UserID)
		file, header, err := r.FormFile("moneroWallet")
		walletUploadServer := false
		walletKeysUploadServer := false
		if err == nil {
			defer file.Close()
			walletPath := filepath.Join(moneroDir, "wallet")
			err = saveFileToDisk(file, header, walletPath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				user.WalletUploaded = false
				return
			} else {
				walletUploadServer = true
			}

		}

		// Get the uploaded monero wallet keys file and save it to disk
		file, header, err = r.FormFile("moneroWalletKeys")
		if err == nil {
			defer file.Close()
			walletKeyPath := filepath.Join(moneroDir, "wallet.keys")
			err = saveFileToDisk(file, header, walletKeyPath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				user.WalletUploaded = false
				return
			} else {
				walletKeysUploadServer = true
			}
		}

		if walletUploadServer && walletKeysUploadServer {

			// convert xmrWallets to a map
			existingWallets := make(map[int]int)
			for _, wallet := range xmrWallets {
				existingWallets[wallet[0]*10000+wallet[1]] = 1
			}

			user.WalletUploaded = true
			walletRunning := true
			log.Println("Monero wallet uploaded")
			// check if the element exists in the map and append if not
			if _, ok := existingWallets[user.UserID*10000+starting_port]; !ok {
				xmrWallets = append(xmrWallets, []int{user.UserID, starting_port})
				walletRunning = false
			}
			go startMoneroWallet(starting_port, user.UserID, user)
			if !walletRunning {
				starting_port++
			}
		}

		// Update the user with the new data
		err = updateUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to the user page
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}

	// render change password form
	tmpl, err := template.ParseFiles("web/templates/user.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		user, valid := getUserBySessionCached(cookie.Value)
		if valid && !checkLoggedInAdmin(w, r) {
			log.Println("Already logged in as", user.Username, " - redirecting from registration to user panel.")
			http.Redirect(w, r, "/user", http.StatusSeeOther)
			return
		}
	}

	err = registerTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func changeUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting change user handler function")
	// retrieve user from session
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	user, valid := getUserBySessionCached(sessionToken.Value)
	if !valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// initialize user page data struct
	data := utils.UserPageData{}

	// process form submission
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user.EthAddress = r.FormValue("ethereumAddress")
		user.SolAddress = r.FormValue("solanaAddress")
		user.HexcoinAddress = r.FormValue("hexcoinAddress")
		minDono, _ := strconv.Atoi(r.FormValue("minUsdAmount"))
		user.MinDono = minDono
		minDonoValue = float64(minDono)

		// Update the user with the new data

		user = setUserMinDonos(user)
		err = updateUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to the user page
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}

	// render change password form
	tmpl, err := template.ParseFiles("web/templates/user.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func saveFileToDisk(file multipart.File, header *multipart.FileHeader, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return err
	}

	return nil
}

func renderChangePasswordForm(w http.ResponseWriter, data utils.UserPageData) {
	tmpl, err := template.ParseFiles("web/templates/user.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// invalidate session token and redirect user to home page
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func incorrectLoginHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/incorrect_login.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func createSession(userID int) (string, error) {
	sessionToken := uuid.New().String()
	userSessions[sessionToken] = userID
	return sessionToken, nil
}

func validateSession(r *http.Request) (int, error) {
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		return 0, fmt.Errorf("no session token found")
	}
	userID, ok := userSessions[sessionToken.Value]
	if !ok {
		return 0, fmt.Errorf("invalid session token")
	}
	return userID, nil
}

func condenseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateStrings(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n]
}

func getUserPathByID(id int) string {
	return fmt.Sprintf("users/%d/", id)
}

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		// File exists
		return true
	} else {
		return false
	}

}

func checkUserGIF(userpath string) bool {
	up := userpath + "gifs/default.gif"
	//log.Println("checking", up)
	b := checkFileExists(up)
	if b {
		log.Println("user gif exists")
	} else {
		log.Println("user gif doesn't exist")
	}
	return b
}

func checkUserSound(userpath string) bool {
	up := userpath + "sounds/default.mp3"
	//log.Println("checking", up)
	b := checkFileExists(up)
	if b {
		log.Println("user sound exists")
	} else {
		log.Println("user sound doesn't exist")
	}
	return b
}

func checkUserMoneroWallet(userpath string) bool {
	up := userpath + "monero/wallet"
	//log.Println("checking", up)
	b := checkFileExists(up)
	if b {
		log.Println("user wallet exists")
	} else {
		log.Println("user wallet doesn't exist")
	}
	return b
}

func checkUserMoneroWalletKeys(userpath string) bool {
	up := userpath + "monero/wallet"
	//log.Println("checking", up)
	b := checkFileExists(up)
	if b {
		log.Println("user wallet keys exists")
	} else {
		log.Println("user wallet keys doesn't exist")
	}
	return b
}

func alertOBSHandler(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("value")
	user, _ := getUserByAlertURL(value)

	newDono, err := checkDonoQueue(db, user.UserID)
	a.Userpath = getUserPathByID(user.UserID)

	if !checkUserGIF(a.Userpath) || !checkUserSound(a.Userpath) { // check if user has uploaded custom gif/sounds for alert
		a.Userpath = "media/"
	}

	if err != nil {
		log.Printf("Error checking donation queue: %v\n", err)
	}

	if newDono {
		fmt.Println("Showing NEW DONO!")
		a.DisplayToggle = ""
	} else {
		a.MediaURL = ""
		a.DisplayToggle = "display: none;"
		a.Refresh = 3
	}
	err = alertTemplate.Execute(w, a)
	if err != nil {
		fmt.Println(err)
	}
}

func progressbarOBSHandler(w http.ResponseWriter, r *http.Request, ur *models.UserRepository) {
	value := r.URL.Query().Get("value")
	obsData, err := getOBSDataByAlertURL(value)

	if err != nil {
		log.Println(err)
		err_ := indexTemplate.Execute(w, nil)
		if err_ != nil {
			http.Error(w, err_.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	/*log.Println("Progress bar message:", obsData.Message)
	log.Println("Progress bar needed:", obsData.Needed)
	log.Println("Progress bar sent:", obsData.Sent)*/

	pb.Message = obsData.Message
	pb.Needed = obsData.Needed
	pb.Sent = obsData.Sent

	err = progressbarTemplate.Execute(w, pb)
	if err != nil {
		fmt.Println(err)
	}
}

func cryptosStructToJSONString(s utils.CryptosEnabled) string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Println("cryptosStructToJSONString error:", err)
		return ""
	}
	return string(bytes)
}

func cryptosJsonStringToStruct(jsonStr string) utils.CryptosEnabled {
	var s utils.CryptosEnabled
	err := json.Unmarshal([]byte(jsonStr), &s)
	if err != nil {
		log.Println("cryptosJsonStringToStruct error:", err)
		return utils.CryptosEnabled{}
	}
	return s
}

func cryptoSettingsHandler(w http.ResponseWriter, r *http.Request, ur *models.UserRepository, mr *models.MoneroRepository) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, valid := ur.GetUserBySessionCached(cookie.Value)
	if !valid || r.Method == http.MethodPost {
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	} else {
		userPath := getUserPathByID(user.UserID)
		moneroWalletString := "monero wallet not uploaded"
		moneroWalletKeysString := "monero wallet not key uploaded"

		if checkUserMoneroWallet(userPath) && !mr.CheckMoneroPort(user.UserID) {
			moneroWalletString = "monero wallet uploaded but not running correctly. Please ensure you have created a view only wallet with no password."
			moneroWalletKeysString = "monero wallet key uploaded but not running correctly. Please ensure you have created a view only wallet with no password."
		}

		data := struct {
			UserID                 int
			Username               string
			HashedPassword         []byte
			EthAddress             string
			SolAddress             string
			HexcoinAddress         string
			XMRWalletPassword      string
			MinDono                int
			MinMediaDono           int
			MediaEnabled           bool
			CreationDatetime       string
			ModificationDatetime   string
			Links                  string
			DonoGIF                string
			DonoSound              string
			AlertURL               string
			MinSol                 float64
			MinEth                 float64
			MinXmr                 float64
			MinPaint               float64
			MinHex                 float64
			MinMatic               float64
			MinBusd                float64
			MinShib                float64
			MinUsdc                float64
			MinTusd                float64
			MinWbtc                float64
			MinPnk                 float64
			DateEnabled            time.Time
			WalletUploaded         bool
			WalletPending          bool
			CryptosEnabled         utils.CryptosEnabled
			BillingData            utils.BillingData
			MoneroWalletString     string
			MoneroWalletKeysString string
		}{
			UserID:                 user.UserID,
			Username:               user.Username,
			HashedPassword:         user.HashedPassword,
			EthAddress:             user.EthAddress,
			SolAddress:             user.SolAddress,
			HexcoinAddress:         user.HexcoinAddress,
			XMRWalletPassword:      user.XMRWalletPassword,
			MinDono:                user.MinDono,
			MinMediaDono:           user.MinMediaDono,
			MediaEnabled:           user.MediaEnabled,
			CreationDatetime:       user.CreationDatetime,
			ModificationDatetime:   user.ModificationDatetime,
			Links:                  user.Links,
			DonoGIF:                user.DonoGIF,
			DonoSound:              user.DonoSound,
			AlertURL:               user.AlertURL,
			MinSol:                 user.MinSol,
			MinEth:                 user.MinEth,
			MinXmr:                 user.MinXmr,
			MinPaint:               user.MinPaint,
			MinHex:                 user.MinHex,
			MinMatic:               user.MinMatic,
			MinBusd:                user.MinBusd,
			MinShib:                user.MinShib,
			MinUsdc:                user.MinUsdc,
			MinTusd:                user.MinTusd,
			MinWbtc:                user.MinWbtc,
			MinPnk:                 user.MinPnk,
			DateEnabled:            user.DateEnabled,
			WalletUploaded:         user.WalletUploaded,
			WalletPending:          user.WalletPending,
			CryptosEnabled:         user.CryptosEnabled,
			BillingData:            user.BillingData,
			MoneroWalletString:     moneroWalletString,
			MoneroWalletKeysString: moneroWalletKeysString,
		}

		tmpl, err := template.ParseFiles("web/templates/cryptoselect.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, header, subheader, message string) {
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		data := struct {
			ErrorHeader    string
			ErrorSubHeader string
			ErrorMessage   string
		}{
			ErrorHeader:    header,
			ErrorSubHeader: subheader,
			ErrorMessage:   message,
		}

		tmpl, err := template.ParseFiles("web/err.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func tosHandler(w http.ResponseWriter, r *http.Request) {
	// Ignore requests for the favicon
	if r.URL.Path == "/favicon.ico" {
		return
	}

	err := tosTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func overflowHandler(w http.ResponseWriter, r *http.Request) {
	err := overflowTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getIPAddress(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}

	if ip == "" {
		ip = r.RemoteAddr
	}

	return ip
}

func indexHandler(w http.ResponseWriter, r *http.Request, ur *models.UserRepository) {

	// Ignore requests for the favicon
	if r.URL.Path == "/favicon.ico" {
		return
	}
	// Get the username from the URL path
	username := r.URL.Path[1:]

	username = strings.ToLower(username)
	user_, valid := ur.GetUserByUsernameCached(username)
	// Calculate all minimum donations
	user := ur.Users[user_.UserID]
	log.Println("user ID in indexHandler =", user.UserID)
	if valid && username != "admin" {
		if user.Links == "" {
			user.Links = "[]"
		}

		if user.DefaultCrypto == "" {
			if user.CryptosEnabled.XMR {
				user.DefaultCrypto = "XMR"
			} else if user.CryptosEnabled.SOL {
				user.DefaultCrypto = "SOL"
			} else if user.CryptosEnabled.ETH {
				user.DefaultCrypto = "ETH"
			} else if user.CryptosEnabled.PAINT {
				user.DefaultCrypto = "PAINT"
			} else if user.CryptosEnabled.HEX {
				user.DefaultCrypto = "HEX"
			} else if user.CryptosEnabled.MATIC {
				user.DefaultCrypto = "MATIC"
			} else if user.CryptosEnabled.BUSD {
				user.DefaultCrypto = "BUSD"
			} else if user.CryptosEnabled.SHIB {
				user.DefaultCrypto = "SHIB"
			} else if user.CryptosEnabled.PNK {
				user.DefaultCrypto = "PNK"
			}

		}

		i := utils.IndexDisplay{
			MaxChar:        MessageMaxChar,
			MinDono:        user.MinDono,
			MinSolana:      user.MinSol,
			MinEthereum:    user.MinEth,
			MinMonero:      user.MinXmr,
			MinHex:         user.MinHex,
			MinPolygon:     user.MinMatic,
			MinBusd:        user.MinBusd,
			MinShib:        user.MinShib,
			MinPnk:         user.MinPnk,
			MinPaint:       user.MinPaint,
			SolPrice:       prices.Solana,
			ETHPrice:       prices.Ethereum,
			XMRPrice:       prices.Monero,
			PolygonPrice:   prices.Polygon,
			HexPrice:       prices.Hexcoin,
			BusdPrice:      prices.BinanceUSD,
			ShibPrice:      prices.ShibaInu,
			PnkPrice:       prices.Kleros,
			PaintPrice:     prices.Paint,
			CryptosEnabled: user.CryptosEnabled,
			Checked:        checked,
			Links:          user.Links,
			WalletPending:  user.WalletPending,
			DefaultCrypto:  user.DefaultCrypto,
			Username:       username,
		}

		err := donationTemplate.Execute(w, i)
		if err != nil {
			fmt.Println(err)
		}
	} else {

		errorHandler(w, r, "User not found", "didn't find a ferret account with that username", "No username was found.")

		// If no username is present in the URL path, serve the indexTemplate
		err := indexTemplate.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func checkDonoQueue(userID int, ur *models.UserRepository) (bool, error) {
	// Fetch oldest entry from queue table where user_id matches userID
	row := db.QueryRow("SELECT name, message, amount, currency, media_url, usd_amount FROM queue WHERE user_id = ? ORDER BY rowid LIMIT 1", userID)

	var name string
	var message string
	var amount float64
	var currency string
	var media_url string
	var usd_amount float64

	err := row.Scan(&name, &message, &amount, &currency, &media_url, &usd_amount)
	if err == sql.ErrNoRows {
		// Queue is empty, do nothing
		return false, nil
	} else if err != nil {
		// Error occurred while fetching row
		return false, err
	}

	fmt.Println("Showing notif:", name, ":", message)
	// update the form in memory
	a.Name = name
	a.Message = message
	a.Amount, _ = strconv.ParseFloat(utils.PruneStringDecimals(fmt.Sprintf("%f", amount), 4), 64)
	a.Currency = currency
	a.MediaURL = media_url
	a.USDAmount = usd_amount
	a.Refresh = getRefreshFromUSDAmount(usd_amount, media_url)
	a.DisplayToggle = "display: block;"

	// Remove fetched entry from queue table
	_, err = ur.Db.Exec("DELETE FROM queue WHERE name = ? AND message = ? AND amount = ? AND currency = ?", name, message, amount, currency)
	if err != nil {
		return false, err
	}

	return true, nil
}

func getRefreshFromUSDAmount(x float64, s string) int {
	if s == "" {
		return 10
	} // if no media then return 10 second time
	minuteCost := 5
	threeMinuteCost := 10

	if x >= float64(threeMinuteCost) {
		return 3 * 60
	} else if x >= float64(minuteCost) {
		return 1 * 60
	}
	return 10
}

func returnIPPenalty(ips []string, currentDonoIP string) float64 {
	// Check if the encrypted IP matches any of the encrypted IPs in the slice of donos
	sameIPCount := 0
	for _, donoIP := range ips {
		if donoIP == currentDonoIP {
			sameIPCount++
		}
	}
	// Calculate the exponential delay factor based on the number of matching IPs
	expoAdder := 1.00
	if sameIPCount > 2 {
		expoAdder = math.Pow(1.3, float64(sameIPCount)) / 1.3
	}
	return expoAdder
}

func newAccountHandler(w http.ResponseWriter, r *http.Request, ur *models.UserRepository, ir *models.InviteRepository, dr *models.DonoRepository, mr *models.MoneroRepository) {
	username := r.FormValue("username")
	invitecode := r.FormValue("invitecode")

	username = utils.SanitizeStringLetters(username)

	password := r.FormValue("password")
	isAdmin := checkLoggedInAdmin(w, r, ur)
	_, validUser := ur.GetUserByUsernameCached(username)
	if r.Method != http.MethodPost || (validUser && !isAdmin) {
		// Redirect to the payment page if the request is not a POST request
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if !PublicRegistrationsEnabled {
		if isAdmin || ir.CheckValidInviteCode(invitecode) {
			if !checkUsernamePendingOrCreated(username, ur) {
				err_ := ur.CreateNew(username, password)
				if err_ != nil {
					log.Println(err_)
				} else {
					ir.InviteCodeMap[invitecode] = models.InviteCode{Value: invitecode, Active: false}
					ir.UpdateInviteCode(ir.InviteCodeMap[invitecode])
				}
				http.Redirect(w, r, "/user", http.StatusSeeOther)
				return
			} else {
				errorHandler(w, r, "Username invalid.", "Woops, the username you tried seems to be already registered.", "Your invite code worked, but the username you chose does not. Please go back and try to register a different username.")
				return
			}
		} else {
			errorHandler(w, r, "Incorrect invite code", "Woops, the invite code you tried to use is invalid.", "Your invite code is not valid. Please go back and try again, making sure your invite code is inputted correctly.")
			return
		}
	} else {
		if !checkUsernamePendingOrCreated(username, ur) {
			pendingUser, err := models.CreateNewPendingUser(username, password, dr, ur, mr)
			if err != nil {
				log.Println(err)
			}

			xmrNeeded, err := strconv.ParseFloat(pendingUser.XMRNeeded, 64)
			if err != nil {
				// Handle the error
			}

			xmrNeededFormatted := fmt.Sprintf("%.5f", xmrNeeded)

			d := utils.AccountPayData{
				Username:    pendingUser.Username,
				AmountXMR:   xmrNeededFormatted,
				AmountETH:   pendingUser.ETHNeeded,
				AddressXMR:  pendingUser.XMRAddress,
				AddressETH:  pendingUser.ETHAddress,
				UserID:      pendingUser.ID,
				DateCreated: time.Now().UTC(),
			}

			tmp, _ := qrcode.Encode(fmt.Sprintf("monero:%s?tx_amount=%s", pendingUser.XMRAddress, pendingUser.XMRNeeded), qrcode.Low, 320)
			d.QRB64XMR = base64.StdEncoding.EncodeToString(tmp)

			donationLink := fmt.Sprintf("ethereum:%s?value=%s", pendingUser.ETHAddress, pendingUser.ETHNeeded)
			tmp, _ = qrcode.Encode(donationLink, qrcode.Low, 320)
			d.QRB64ETH = base64.StdEncoding.EncodeToString(tmp)

			err = accountPayTemplate.Execute(w, d)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			errorHandler(w, r, "Username invalid.", "Woops, the username you tried seems to be already registered.", "The username you chose is already taken or invalid. Please go back and try to register a different username.")
			return
		}
	}
}

func checkUsernamePendingOrCreated(username_ string, ur *models.UserRepository) bool {
	username := strings.ToLower(username_)
	for _, route_ := range routes_ {
		if utils.SanitizeStringLetters(route_.Path) == username {
			return true
		}
	}

	for _, user := range ur.PendingUsers {
		if user.Username == username {
			return true
		}

	}

	for _, user := range ur.Users {
		if user.Username == username {
			return true
		}

	}

	return false

}

func accountBillingHandler(w http.ResponseWriter, r *http.Request) {
	checkLoggedIn(w, r)
	cookie, _ := r.Cookie("session_token")
	user, _ := getUserBySessionCached(cookie.Value)

	if user.BillingData.NeedToPay {

		admin, _ := getUserByUsernameCached("admin")
		xmrAmount, err := strconv.ParseFloat(user.BillingData.XMRAmount, 64)
		if err != nil {
			log.Println("error parsing xmr value")
		}

		xmrNeededFormatted := fmt.Sprintf("%.5f", xmrAmount)
		d := utils.AccountPayData{
			Username:    user.Username,
			AmountXMR:   xmrNeededFormatted,
			BillingData: user.BillingData,
			AmountETH:   user.BillingData.ETHAmount,
			AddressXMR:  user.BillingData.XMRAddress,
			AddressETH:  admin.EthAddress,
			UserID:      user.UserID,
			DateCreated: time.Now().UTC(),
		}

		tmp, _ := qrcode.Encode(fmt.Sprintf("monero:%s?tx_amount=%s", d.AddressXMR, xmrNeededFormatted), qrcode.Low, 320)
		d.QRB64XMR = base64.StdEncoding.EncodeToString(tmp)

		donationLink := fmt.Sprintf("ethereum:%s?value=%s", admin.EthAddress, d.AmountETH)
		tmp, _ = qrcode.Encode(donationLink, qrcode.Low, 320)
		d.QRB64ETH = base64.StdEncoding.EncodeToString(tmp)

		err = billPayTemplate.Execute(w, d)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}
}

func getNewAccountETHPrice() string {
	ethPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.18f", (15.00/prices.Ethereum)), 64)
	ethStr := utils.FuzzDono(ethPrice, "ETH")
	ethStr_, _ := utils.StandardizeFloatToString(ethStr)
	return ethStr_
}
func getNewAccountXMRPrice() string {
	xmrPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", (15.00/prices.Monero)), 64)
	xmrStr, _ := utils.StandardizeFloatToString(xmrPrice)
	return xmrStr
}

func getXMRAmountInUSD(usdAmount float64) string {
	xmrPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", (usdAmount/prices.Monero)), 64)
	xmrStr, _ := utils.StandardizeFloatToString(xmrPrice)
	return xmrStr
}

func getETHAmountInUSD(usdAmount float64) string {
	ethPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.18f", (usdAmount/prices.Ethereum)), 64)
	ethStr := utils.FuzzDono(ethPrice, "ETH")
	ethStr_, _ := utils.StandardizeFloatToString(ethStr)
	return ethStr_
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	user, validUser := getUserByUsernameCached(username)

	if r.Method != http.MethodPost || !validUser {
		// Redirect to the payment page if the request is not a POST request
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get the user's IP address
	ip := getIPAddress(r)
	log.Println("dono ip", ip)

	// Get form values
	fCrypto := r.FormValue("crypto")
	fAmount := r.FormValue("amount")
	fName := r.FormValue("name")
	fMessage := r.FormValue("message")
	fMedia := r.FormValue("media")
	fShowAmount := r.FormValue("showAmount")
	encrypted_ip := encryptIP(ip)
	log.Println("encrypted_ip", encrypted_ip)

	matching_ips := utils.CheckPendingDonosFromIP(pending_donos, ip)

	log.Println("Waiting pending donos from this IP:", matching_ips)
	if matching_ips >= 9 {
		http.Redirect(w, r, "/overflow", http.StatusSeeOther)
		return
	}

	if fAmount == "" {
		fAmount = "0"
	}
	amount, err := strconv.ParseFloat(fAmount, 64)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("fAmount", fAmount)
	fmt.Println("Amount", amount)

	minValues := map[string]float64{
		"XMR":   user.MinXmr,
		"SOL":   user.MinSol,
		"ETH":   user.MinEth,
		"PAINT": user.MinPaint,
		"HEX":   user.MinHex,
		"MATIC": user.MinMatic,
		"BUSD":  user.MinBusd,
		"SHIB":  user.MinShib,
		"PNK":   user.MinPnk,
	}

	if minValue, ok := minValues[fCrypto]; ok && amount < minValue {
		amount = minValue
	}

	name := fName
	if name == "" {
		name = "Anonymous"
	}

	message := fMessage
	if message == "" {
		message = " "
	}

	media := html.EscapeString(fMedia)

	showAmount, _ := strconv.ParseBool(fShowAmount)

	var s utils.CryptoSuperChat
	params := url.Values{}

	params.Add("name", name)
	params.Add("msg", message)
	params.Add("media", condenseSpaces(media))
	params.Add("amount", strconv.FormatFloat(amount, 'f', 4, 64))
	params.Add("show", strconv.FormatBool(showAmount))

	s.Amount = strconv.FormatFloat(amount, 'f', 4, 64)
	s.Name = html.EscapeString(truncateStrings(condenseSpaces(name), NameMaxChar))
	s.Message = html.EscapeString(truncateStrings(condenseSpaces(message), MessageMaxChar))
	s.Media = html.EscapeString(media)

	USDAmount := getUSDValue(amount, fCrypto)
	if fCrypto == "XMR" {
		createNewXMRDono(s.Name, s.Message, s.Media, amount, encrypted_ip)
		handleMoneroPayment(w, &s, params, amount, encrypted_ip, showAmount, USDAmount, user.UserID)
	} else if fCrypto == "SOL" {
		new_dono := createNewSolDono(s.Name, s.Message, s.Media, utils.FuzzDono(amount, "SOL"), encrypted_ip)
		handleSolanaPayment(w, &s, params, new_dono.Name, new_dono.Message, new_dono.AmountNeeded, showAmount, media, encrypted_ip, USDAmount, user.UserID)
	} else {
		s.Currency = fCrypto
		new_dono := createNewEthDono(s.Name, s.Message, s.Media, amount, fCrypto, encrypted_ip)
		handleEthereumPayment(w, &s, new_dono.Name, new_dono.Message, new_dono.AmountNeeded, showAmount, new_dono.MediaURL, fCrypto, encrypted_ip, USDAmount, user.UserID)
	}
}

func createNewSolDono(name string, message string, mediaURL string, amountNeeded float64, encrypted_ip string) utils.SuperChat {
	new_dono := utils.CreatePendingDono(name, message, mediaURL, amountNeeded, "SOL", encrypted_ip)
	pending_donos = utils.AppendPendingDono(pending_donos, new_dono)

	return new_dono
}

func createNewXMRDono(name string, message string, mediaURL string, amountNeeded float64, encrypted_ip string) {
	new_dono := utils.CreatePendingDono(name, message, mediaURL, amountNeeded, "XMR", encrypted_ip)
	pending_donos = utils.AppendPendingDono(pending_donos, new_dono)
}

func checkDonationStatusHandler(w http.ResponseWriter, r *http.Request, dr *models.DonoRepository) {
	donationIDStr := r.FormValue("donation_id") // Get the donation ID from the query string
	donationID, err := strconv.Atoi(donationIDStr)
	log.Println("User Page Checking DonationID:", donationID)
	if err != nil {
		http.Error(w, "Invalid donation ID", http.StatusBadRequest)
		return
	}

	completed := dr.IsDonoFulfilled(donationID)
	if completed {
		fmt.Fprintf(w, `true`) // Return the status as a JSON response
	} else {
		fmt.Fprintf(w, `false`) // Return the status as a JSON response
	}
}

func handleEthereumPayment(w http.ResponseWriter, dr *models.DonoRepository, s *utils.CryptoSuperChat, name_ string, message_ string, amount_ float64, showAmount_ bool, media_ string, fCrypto string, encrypted_ip string, USDAmount float64, userID int) {
	address := ur.GetEthAddressByID(userID)
	log.Println("handleEthereumPayment() address:", address)

	decimals, _ := utils.GetCryptoDecimalsByCode(fCrypto)
	donoStr := fmt.Sprintf("%.*f", decimals, amount_)
	log.Println("handleEthereumPayment() donoStr:", address)

	s.Amount = donoStr
	log.Println("handleEthereumPayment() donoStr:", s.Amount)

	if fCrypto != "ETH" {
		s.ContractAddress, _ = utils.GetCryptoContractByCode(fCrypto)
	} else {
		s.ContractAddress = "ETH"
	}

	if name_ == "" {
		s.Name = "Anonymous"
		name_ = s.Name
	} else {
		s.Name = html.EscapeString(truncateStrings(condenseSpaces(name_), NameMaxChar))
	}

	s.WeiAmount = ethToWei(donoStr)
	s.Media = html.EscapeString(media_)
	s.Address = address

	donationLink := fmt.Sprintf("ethereum:%s?value=%s", address, donoStr)

	tmp, _ := qrcode.Encode(donationLink, qrcode.Low, 320)
	s.QRB64 = base64.StdEncoding.EncodeToString(tmp)
	s.DonationID = dr.Create(userID, address, s.Name, s.Message, s.Amount, fCrypto, encrypted_ip, showAmount_, USDAmount, media_)
	err := payTemplate.Execute(w, s)
	if err != nil {
		fmt.Println(err)
	}
}

func handleSolanaPayment(w http.ResponseWriter, dr *models.DonoRepository, s *utils.CryptoSuperChat, params url.Values, name_ string, message_ string, amount_ float64, showAmount_ bool, media_ string, encrypted_ip string, USDAmount float64, userID int) {
	// Get Solana address and desired balance from request
	address := dr.UserRepo.GetSolAddressByID(userID)
	donoStr := fmt.Sprintf("%.*f", 9, amount_)

	s.Amount = donoStr

	if name_ == "" {
		s.Name = "Anonymous"
	} else {
		s.Name = html.EscapeString(truncateStrings(condenseSpaces(name_), NameMaxChar))
	}

	s.Media = html.EscapeString(media_)
	s.PayID = address
	s.Address = address
	s.Currency = "SOL"

	params.Add("id", s.Address)

	s.CheckURL = params.Encode()

	tmp, _ := qrcode.Encode("solana:"+address+"?amount="+donoStr, qrcode.Low, 320)
	s.QRB64 = base64.StdEncoding.EncodeToString(tmp)

	s.DonationID = dr.Create(userID, address, name_, message_, s.Amount, "SOL", encrypted_ip, showAmount_, USDAmount, media_)

	err := payTemplate.Execute(w, s)
	if err != nil {
		fmt.Println(err)
	}
}

func handleMoneroPayment(w http.ResponseWriter, mr *models.MoneroRepository, s *utils.CryptoSuperChat, params url.Values, amount float64, encrypted_ip string, showAmount bool, USDAmount float64, userID int) {
	payload := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"make_integrated_address"}`)
	portID := mr.GetPortID(mr.XmrWallets, userID)

	found := true
	if portID == -100 {
		found = false
	}

	if found {
		fmt.Println("Port ID for user", userID, "is", portID)
	} else {
		fmt.Println("Port ID not found for user", userID)
	}

	rpcURL_ := "http://127.0.0.1:" + strconv.Itoa(portID) + "/json_rpc"

	req, err := http.NewRequest("POST", rpcURL_, payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("ERROR CREATING")
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("ERROR CREATING")
	}

	resp := &utils.RPCResponse{}
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("ERROR CREATING")
	}

	s.PayID = html.EscapeString(resp.Result.PaymentID)
	s.Address = html.EscapeString(resp.Result.IntegratedAddress)
	s.Currency = "XMR"
	params.Add("id", resp.Result.PaymentID)
	params.Add("address", resp.Result.IntegratedAddress)
	s.CheckURL = params.Encode()

	tmp, _ := qrcode.Encode(fmt.Sprintf("monero:%s?tx_amount=%s", resp.Result.IntegratedAddress, s.Amount), qrcode.Low, 320)
	s.QRB64 = base64.StdEncoding.EncodeToString(tmp)

	s.DonationID = createNewDono(userID, s.PayID, s.Name, s.Message, s.Amount, "XMR", encrypted_ip, showAmount, USDAmount, s.Media)

	err = payTemplate.Execute(w, s)
	if err != nil {
		fmt.Println(err)
	}
}

func ethToWei(ethStr string) *big.Int {
	etherValue := big.NewFloat(1000000000000000000)
	f, err := strconv.ParseFloat(ethStr, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	number := big.NewFloat(f)

	weiValue := new(big.Int)
	weiValue, _ = weiValue.SetString(number.Mul(number, etherValue).Text('f', 0), 10)

	return weiValue
}
