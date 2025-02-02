package main

import (
	"flag"
	"fmt"
	immich "immich-exp/immich"
	"immich-exp/models"

	"net/http"
	"strconv"
	"strings"

	"os"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const DEFAULTPORT = 8090

var (
	Version     = "dev"
	Author      = "martabal"
	ProjectName = "immich-exporter"
)

func main() {
	loadenv()
	fmt.Printf("%s (version %s)\n", ProjectName, Version)
	fmt.Println("Author: ", Author)
	fmt.Println("Using log level: ", log.GetLevel())
	log.Info("Immich URL: ", models.Getbaseurl())
	log.Info("Started")
	http.HandleFunc("/metrics", metrics)
	addr := ":" + strconv.Itoa(models.GetPort())
	log.Info("Listening on port ", models.GetPort())
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func metrics(w http.ResponseWriter, r *http.Request) {
	log.Trace("New request")
	registry := prometheus.NewRegistry()
	immich.Allrequests(registry)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func loadenv() {
	var envfile bool
	flag.BoolVar(&envfile, "e", false, "Use .env file")
	flag.Parse()
	_, err := os.Stat(".env")
	if !os.IsNotExist(err) && !envfile {
		err := godotenv.Load(".env")
		if err != nil {
			log.Panic("Error loading .env file:", err)
		}
		// fmt.Println("Using .env file")
	}

	immichapikey := getEnv("IMMICH_API_KEY", "", false, "Immich API Key is not set", true)
	immichURL := getEnv("IMMICH_BASE_URL", "http://localhost:8080", true, "Qbittorrent base_url is not set. Using default base_url", false)
	exporterPort := getEnv("EXPORTER_PORT", strconv.Itoa(DEFAULTPORT), false, "", false)

	num, err := strconv.Atoi(exporterPort)

	if err != nil {
		log.Panic("EXPORTER_PORT must be an integer")
	}
	if num < 0 || num > 65353 {
		log.Panic("EXPORTER_PORT must be > 0 and < 65353")
	}

	setLogLevel(getEnv("LOG_LEVEL", "INFO", false, "", false))
	models.SetApp(num, false)
	models.Setuser(immichURL, immichapikey)
}
func setLogLevel(logLevel string) {
	logLevels := map[string]log.Level{
		"TRACE": log.TraceLevel,
		"DEBUG": log.DebugLevel,
		"INFO":  log.InfoLevel,
		"WARN":  log.WarnLevel,
		"ERROR": log.ErrorLevel,
	}

	level, found := logLevels[strings.ToUpper(logLevel)]
	if !found {
		level = log.InfoLevel
	}

	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
}

func getEnv(key string, fallback string, printLog bool, logPrinted string, needed bool) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		if needed {
			log.Panicln("Please set a value for", key)
		}
		if printLog {
			log.Warn(logPrinted)
		}
		return fallback
	}
	return value
}
