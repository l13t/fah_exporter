package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var namespace string

// type Configuration struct {
// 	teamID   int    `yaml:"team_id"`
// 	userName string `yaml:"user_name"`
// 	passkey  string `yaml:"passkey"`
// 	port     int    `yaml:"port"`
// 	apu_url  string `yaml:"api_url"`
// }

// FoldingAtHomeClient represents a client for the Folding@Home API
type FoldingAtHomeClient struct {
	BaseURL  string
	TeamID   int
	UserName string
}

// StatsResponse represents the API response for Folding@Home statistics
type StatsResponse struct {
	TotalPoints   int `json:"total_points"`
	WorkUnits     int `json:"wus"`
	ActiveClients int `json:"active_clients"`
}

// FetchStats fetches statistics from the Folding@Home API
func (c *FoldingAtHomeClient) FetchTeamUserStats() (*StatsResponse, error) {
	url := fmt.Sprintf("%s/team/%d", c.BaseURL, c.TeamID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d. %s", resp.StatusCode, url)
	}

	var stats StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return &stats, nil
}

type TeamStats struct {
	TeamID   int    `json:"id"`
	TeamName string `json:"name"`
	Founder  string `json:"founder"`
	Score    int    `json:"score"`
	Wus      int    `json:"wus"`
	Rank     int    `json:"rank"`
}

func (c *FoldingAtHomeClient) FetchTeamStats() (*TeamStats, error) {
	url := fmt.Sprintf("%s/team/%d", c.BaseURL, c.TeamID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d. %s", resp.StatusCode, url)
	}

	var stats TeamStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return &stats, nil
}

type UsersStats struct {
	User  string `json:"user"`
	ID    int    `json:"id"`
	Rank  int    `json:"rank"`
	Score int    `json:"score"`
	Wus   int    `json:"wus"`
}

func (c *FoldingAtHomeClient) FetchUsersStats() (*[]UsersStats, error) {
	url := fmt.Sprintf("%s/team/%d/members", c.BaseURL, c.TeamID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d. %s", resp.StatusCode, url)
	}

	var stats []UsersStats
	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshal into a slice of slices
	var rawData [][]interface{}
	err = json.Unmarshal(body, &rawData)
	if err != nil {
		panic(err)
	}
	headers := rawData[0]
	for _, row := range rawData[1:] {
		r := UsersStats{}

		// Map each field to the struct
		for i, header := range headers {
			switch header {
			case "name":
				r.User = row[i].(string)
			case "id":
				r.ID = int(row[i].(float64)) // JSON numbers are float64
			case "rank":
				if row[i] == nil {
					r.Rank = 0
				} else {
					rank := int(row[i].(float64))
					r.Rank = rank
				}
			case "score":
				r.Score = int(row[i].(float64))
			case "wus":
				r.Wus = int(row[i].(float64))
			}
		}

		stats = append(stats, r)
	}

	return &stats, nil
}

type UserStats struct {
	Name           string `json:"name"`
	Score          int    `json:"score"`
	Wus            int    `json:"wus"`
	Rank           int    `json:"rank"`
	Active_7_days  int    `json:"active_7_days"`
	Active_50_days int    `json:"active_50_days"`
}

func (c *FoldingAtHomeClient) FetchUserStats() (*UserStats, error) {
	url := fmt.Sprintf("%s/user/%s", c.BaseURL, c.UserName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d. %s", resp.StatusCode, url)
	}

	var stats UserStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return &stats, nil
}

func (c *FoldingAtHomeClient) Up() int {
	// reach out to the API to check if it's up
	resp, err := http.Get(c.BaseURL)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return 1
	}
	return 0
}

// Exporter collects Folding@Home stats and exposes them as Prometheus metrics
type Exporter struct {
	client          *FoldingAtHomeClient
	up              prometheus.Gauge
	teamTotalPoints *prometheus.GaugeVec
	teamWorkUnits   *prometheus.GaugeVec
	teamRank        *prometheus.GaugeVec
	usersScore      *prometheus.GaugeVec
	usersWus        *prometheus.GaugeVec
	usersRank       *prometheus.GaugeVec
	userScore       *prometheus.GaugeVec
	userWus         *prometheus.GaugeVec
	userRank        *prometheus.GaugeVec
	userActive7     *prometheus.GaugeVec
	userActive50    *prometheus.GaugeVec
	mu              sync.Mutex
}

// NewExporter creates a new Exporter
func NewExporter(client *FoldingAtHomeClient, namespace string) *Exporter {
	return &Exporter{
		client: client,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "FAH Metric Collection Operational",
		}),
		teamTotalPoints: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_score",
			Help:      "Total score of the team.",
		}, []string{"team_name", "team_id"}),
		teamWorkUnits: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_work_units",
			Help:      "Total work units completed by the team.",
		}, []string{"team_name", "team_id"}),
		teamRank: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_rank",
			Help:      "Team rank.",
		}, []string{"team_name", "team_id"}),
		usersScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_users_score",
			Help:      "User score",
		}, []string{"user", "team"}),
		usersRank: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_users_rank",
			Help:      "User rank",
		}, []string{"user", "team"}),
		usersWus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "team_users_wus",
			Help:      "User work units",
		}, []string{"user", "team"}),
		userScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "user_score",
			Help:      "User score",
		}, []string{"user"}),
		userRank: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "user_rank",
			Help:      "User rank",
		}, []string{"user"}),
		userWus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "user_wus",
			Help:      "User work units",
		}, []string{"user"}),
		userActive7: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "user_active_7_days",
			Help:      "User active in the last 7 days",
		}, []string{"user"}),
		userActive50: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "user_active_50_days",
			Help:      "User active in the last 50 days",
		}, []string{"user"}),
	}
}

// Describe sends the descriptors of each metric over to the provided channel
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.teamTotalPoints.Describe(ch)
	e.teamWorkUnits.Describe(ch)
	e.teamRank.Describe(ch)
	e.usersScore.Describe(ch)
	e.usersWus.Describe(ch)
	e.usersRank.Describe(ch)
	e.userScore.Describe(ch)
	e.userWus.Describe(ch)
	e.userRank.Describe(ch)
	e.userActive7.Describe(ch)
	e.userActive50.Describe(ch)
}

// Collect fetches the stats and delivers them to Prometheus
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.up.Set(float64(e.client.Up()))
	e.up.Collect(ch)

	if e.client.TeamID != -1 {
		teamStats, err := e.client.FetchTeamStats()
		if err != nil {
			log.Printf("Error fetching stats: %v", err)
			return
		}

		e.teamTotalPoints.WithLabelValues(teamStats.TeamName, strconv.Itoa(e.client.TeamID)).Set(float64(teamStats.Score))
		e.teamWorkUnits.WithLabelValues(teamStats.TeamName, strconv.Itoa(e.client.TeamID)).Set(float64(teamStats.Wus))
		e.teamRank.WithLabelValues(teamStats.TeamName, strconv.Itoa(e.client.TeamID)).Set(float64(teamStats.Rank))

		userStats, err := e.client.FetchUsersStats()
		if err != nil {
			log.Printf("Error fetching user stats: %v", err)
			return
		}

		for _, stat := range *userStats {
			e.usersScore.WithLabelValues(stat.User, strconv.Itoa(e.client.TeamID)).Set(float64(stat.Score))
			e.usersWus.WithLabelValues(stat.User, strconv.Itoa(e.client.TeamID)).Set(float64(stat.Wus))
			e.usersRank.WithLabelValues(stat.User, strconv.Itoa(e.client.TeamID)).Set(float64(stat.Rank))
		}

		e.teamTotalPoints.Collect(ch)
		e.teamWorkUnits.Collect(ch)
		e.teamRank.Collect(ch)
		e.usersScore.Collect(ch)
		e.usersWus.Collect(ch)
		e.usersRank.Collect(ch)
	}

	if e.client.UserName != "" {
		stats, err := e.client.FetchUserStats()
		if err != nil {
			log.Printf("Error fetching user stats: %v", err)
			return
		}

		e.userScore.WithLabelValues(e.client.UserName).Set(float64(stats.Score))
		e.userWus.WithLabelValues(e.client.UserName).Set(float64(stats.Wus))
		e.userRank.WithLabelValues(e.client.UserName).Set(float64(stats.Rank))
		e.userActive7.WithLabelValues(e.client.UserName).Set(float64(stats.Active_7_days))
		e.userActive50.WithLabelValues(e.client.UserName).Set(float64(stats.Active_50_days))

		e.userScore.Collect(ch)
		e.userWus.Collect(ch)
		e.userRank.Collect(ch)
		e.userActive7.Collect(ch)
		e.userActive50.Collect(ch)
	}
}

func main() {
	var (
		listenAddress = flag.String("listen-address", ":9401", "Address to listen on for HTTP requests.")
		fahBaseURL    = flag.String("fah-api-url", "https://api.foldingathome.org", "Base URL for Folding@Home API.")
		teamID        = flag.Int("team-id", -1, "Team ID to fetch stats for.")
		userName      = flag.String("user-name", "", "User name to fetch stats for.")
		ns            = flag.String("namespace", "foldingathome", "Namespace for the Prometheus metrics.")
	)
	flag.Parse()

	client := &FoldingAtHomeClient{
		BaseURL:  *fahBaseURL,
		TeamID:   *teamID,
		UserName: *userName,
	}
	namespace = *ns

	exporter := NewExporter(client, namespace)
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting server on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
