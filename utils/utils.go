package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var contracts = map[string]string{
	"PAINT":     "0x4c6ec08cf3fc987c6c4beb03184d335a2dfc4042",
	"HEX":       "0x2b591e99afE9f32eAA6214f7B7629768c40Eeb39",
	"MATIC":     "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0",
	"BUSD":      "0x4Fabb145d64652a948d72533023f6E7A623C7C53",
	"SHIBA_INU": "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE",
	"PNK":       "0x93ed3fbe21207ec2e8f2d3c3de6e058cb73bc04d",
}

var prices = CryptoPrice{}

var cryptoMap = map[string]map[string]interface{}{
	"paint": {
		"name":     "Paint",
		"code":     "PAINT",
		"svg":      "paint.svg",
		"min":      "{{.MinPaint}}",
		"contract": contracts["PAINT"],
		"decimals": 18,
	},
	"hex": {
		"name":     "Hexcoin",
		"code":     "HEX",
		"svg":      "hex.svg",
		"min":      "{{.MinHex}}",
		"contract": contracts["HEX"],
		"decimals": 8,
	},
	"matic": {
		"name":     "Polygon",
		"code":     "MATIC",
		"svg":      "matic.svg",
		"min":      "{{.MinPolygon}}",
		"contract": contracts["MATIC"],
		"decimals": 18,
	},
	"busd": {
		"name":     "Binance USD",
		"code":     "BUSD",
		"svg":      "busd.svg",
		"min":      "{{.MinBusd}}",
		"contract": contracts["BUSD"],
		"decimals": 18,
	},
	"shiba_inu": {
		"name":     "Shiba Inu",
		"code":     "SHIB",
		"svg":      "shiba_inu.svg",
		"min":      "{{.MinShib}}",
		"contract": contracts["SHIBA_INU"],
		"decimals": 18,
	},
	"pnk": {
		"name":     "Kleros",
		"code":     "PNK",
		"svg":      "pnk.svg",
		"min":      "{{.MinPnk}}",
		"contract": contracts["PNK"],
		"decimals": 18,
	},
}

func GetTransactionAmount(t Transfer) string {
	d := decimal.NewFromFloat(t.Value)
	return d.String()
}

func GetTransactionToken(t Transfer) string {
	asset := ""
	if t.RawContract.Address == "" {
		asset = "ETH"
	} else {
		asset = GetTokenName(t.RawContract.Address)
	}
	return asset
}

func GetTokenName(contractAddr string) string {
	switch contractAddr {
	case contracts["PAINT"]:
		return "PAINT"
	case contracts["HEX"]:
		return "HEX"
	case contracts["MATIC"]:
		return "MATIC"
	case contracts["BUSD"]:
		return "BUSD"
	case contracts["SHIBA_INU"]:
		return "SHIBA_INU"
	case contracts["PNK"]:
		return "PNK"
	default:
		return "UNKNOWN"
	}
}

func GetCryptoPrices() (CryptoPrice, error) {

	// Call the Coingecko API to get the current price for each cryptocurrency
	url := "https://api.coingecko.com/api/v3/simple/price?ids=monero,solana,ethereum,paint,hex,matic-network,binance-usd,shiba-inu,kleros&vs_currencies=usd"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var data map[string]map[string]float64
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return prices, err
	}

	prices = CryptoPrice{
		Monero:     data["monero"]["usd"],
		Solana:     data["solana"]["usd"],
		Ethereum:   data["ethereum"]["usd"],
		Paint:      data["paint"]["usd"],
		Hexcoin:    data["hex"]["usd"],
		Polygon:    data["matic-network"]["usd"],
		BinanceUSD: data["binance-usd"]["usd"],
		ShibaInu:   data["shiba-inu"]["usd"],
		Kleros:     data["kleros"]["usd"],
	}

	return prices, nil
}

func GetCryptoContractByCode(code string) (string, error) {
	code = strings.ToUpper(code)
	for _, cryptoInfo := range cryptoMap {
		if cryptoInfo["code"] == code {
			contract, ok := cryptoInfo["contract"].(string)
			if !ok {
				return "", fmt.Errorf("contract value for %s is not a string", code)
			}
			return contract, nil
		}
	}
	return "", fmt.Errorf("crypto with code %s not found", code)
}

func GetCryptoDecimalsByCode(code string) (int, error) {
	if code == "ETH" {
		return 18, nil
	} else {
		code = strings.ToUpper(code)
		for _, cryptoInfo := range cryptoMap {
			if cryptoInfo["code"] == code {
				decimals, ok := cryptoInfo["decimals"].(int)
				if !ok {
					return 0, fmt.Errorf("decimals value for crypto with code %s is not an integer", code)
				}
				return decimals, nil
			}
		}
		return 0, fmt.Errorf("crypto with code %s not found", code)
	}
}

func CheckDonos(transfers []Transfer, pending_donos []SuperChat) []SuperChat {
	var completed_donos []SuperChat
	for i, pending_dono := range pending_donos {
		if !pending_dono.Completed {
			for _, transfer := range transfers {
				log.Println("Transfer value:", transfer.Value)
				log.Println("Needed amount:", pending_dono.AmountNeeded)
				if isEqual(transfer.Value, pending_dono.AmountNeeded) {
					pending_donos[i].Completed = true
					pending_donos[i].CheckedAt = time.Now().String()
					completed_donos = append(completed_donos, pending_donos[i])
					log.Printf("Completed donation from %v for %.18f ETH", transfer.From, transfer.Value)
				}
			}
		}
	}

	fmt.Println("Completed Donations:")
	for _, dono := range completed_donos {
		fmt.Printf("Amount: %s %v, Completed: %v, Checked At: %v\n", dono.CryptoCode, dono.AmountNeeded, dono.Completed, dono.CheckedAt)
	}

	return completed_donos
}

func CreatePendingDono(name string, message string, mediaURL string, amountNeeded float64, cryptoCode string) SuperChat {
	amountNeeded = FuzzDono(amountNeeded, cryptoCode)
	pendingDono := SuperChat{
		Name:         name,
		Message:      message,
		MediaURL:     mediaURL,
		AmountNeeded: amountNeeded,
		Completed:    false,
		CreatedAt:    time.Now().String(),
		CheckedAt:    time.Now().String(),
		CryptoCode:   cryptoCode,
	}
	return pendingDono
}

func AppendPendingDono(pending_donos []SuperChat, new_dono SuperChat) []SuperChat {
	pending_donos = append(pending_donos, new_dono)
	return pending_donos
}

func CheckMatchingDono(amount float64, cryptoCode string, pending_donos []SuperChat) bool {
	for _, potential_dono := range pending_donos {
		if potential_dono.AmountNeeded == amount && potential_dono.CryptoCode == cryptoCode {
			return true
		}
	}
	return false
}

func isEqual(a, b float64) bool {
	const epsilon = 1e-18 // threshold for difference
	return math.Abs(a-b) < epsilon
}

func IsEqual(a, b float64) bool {
	const epsilon = 1e-18 // threshold for difference
	return math.Abs(a-b) < epsilon
}

func GenerateUniqueURL() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const length = 30
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return (string(randomString))
}

// extractVideoID extracts the video ID from a YouTube URL
func ExtractVideoID(url string) string {
	videoID := ""
	// Use a regular expression to extract the video ID from the YouTube URL
	re := regexp.MustCompile(`v=([\w-]+)`)
	match := re.FindStringSubmatch(url)
	if len(match) == 2 {
		videoID = match[1]
	}
	return videoID
}

func ReturnIPPenalty(ips []string, currentDonoIP string) float64 {
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

func IsYouTubeLink(link string) (bool, int, string) {
	var timecode int
	var properLink string

	youtubeRegex := regexp.MustCompile(`^(?:https?://)?(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([^&]+)(?:\?t=)?(\d*)$`)
	embedRegex := regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/embed/([^?]+)(?:\?start=)?(\d*)$`)

	if youtubeMatches := youtubeRegex.FindStringSubmatch(link); youtubeMatches != nil {
		if len(youtubeMatches[2]) > 0 {
			fmt.Sscanf(youtubeMatches[2], "%d", &timecode)
		}
		properLink = "https://www.youtube.com/watch?v=" + youtubeMatches[1]
		return true, timecode, properLink
	}

	if embedMatches := embedRegex.FindStringSubmatch(link); embedMatches != nil {
		if len(embedMatches[2]) > 0 {
			fmt.Sscanf(embedMatches[2], "%d", &timecode)
		}
		properLink = "https://www.youtube.com/watch?v=" + embedMatches[1]
		return true, timecode, properLink
	}

	return false, 0, ""
}

// should go in /models/media.go
func FormatMediaURL(media_url string) string {
	isValid, timecode, properLink := IsYouTubeLink(media_url)
	log.Println(isValid, timecode, properLink)

	embedLink := ""
	if isValid {
		videoID := ExtractVideoID(properLink)
		embedLink = fmt.Sprintf(videoID)
	}
	return embedLink
}

func CheckDonoForMediaUSDThreshold(media_url string, dono_usd float64, minDono int) (bool, string) {
	valid := true
	if dono_usd < float64(minDono) {
		media_url = ""
		valid = false

	}
	return valid, media_url
}

func ConvertToFloat64(value string) float64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return f
}
