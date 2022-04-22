package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"runtime"
	"sort"
	"syscall"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/fatih/color"
	"github.com/hako/durafmt"
)

type Config struct {
	Debug      bool
	ShowHeader bool
	Interval   time.Duration
	DateFormat string
	Domain     string
	User       string
	Token      string
	URL        string
	UserAgent  string
	Client     http.Client
}

var config Config = Config{
	Debug:      false,
	ShowHeader: false,
	Interval:   0,
	DateFormat: "2006-01-02T15:04:05.000-0700",
	Domain:     "",
	User:       "dashboard",
	Token:      "",
	URL:        "https://%s/blue/rest/users/%s/favorites/",
	UserAgent:  "bodash/0.1",
	Client:     http.Client{Timeout: time.Second * 5},
}

type FavoriteWrapper struct {
	FavoriteItem struct {
		DisplayName string `json:"displayName"`
		LatestRun   struct {
			ID        string `json:"id"`
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
			Result    string `json:"result"`
			State     string `json:"state"`
		} `json:"latestRun"`
	} `json:"item"`
}

type Job struct {
	ID          string
	DisplayName string
	StartTime   string
	EndTime     string
	Result      string
	State       string
}

func (job *Job) PrintDebugInfo() {
	fmt.Println("id:", job.ID)
	fmt.Println("name:", job.DisplayName)
	fmt.Println("startTime:", job.StartTime)
	fmt.Println("endTime:", job.EndTime)
	fmt.Println("result:", job.Result)
	fmt.Println("state:", job.State)
	fmt.Println()
}

func main() {
	ParseArgs()
	if config.Interval == 0 {
		FetchAndPrint()
	} else {
		RunDashboardLoop()
	}
}

func ParseArgs() {
	ReadConfigStrFromEnv("BODASH_DOMAIN", &config.Domain)
	ReadConfigStrFromEnv("BODASH_TOKEN", &config.Token)
	ReadConfigStrFromEnv("BODASH_USER", &config.User)

	flag.BoolVar(&config.Debug, "debug", config.Debug, "show debug info")
	flag.StringVar(&config.Domain, "domain", config.Domain, "Blue Ocean domain (e.g. myjenkinsdomain.example)")
	flag.BoolVar(&config.ShowHeader, "header", config.ShowHeader, "show dashboard header row")
	flag.DurationVar(&config.Interval, "interval", config.Interval, "dashboard refresh interval")
	flag.StringVar(&config.Token, "token", config.Token, "Blue Ocean user API token")
	flag.StringVar(&config.User, "user", config.User, "Blue Ocean user")
	flag.Parse()

	AssertFlagArgProvided(config.Domain, "-domain")
	AssertFlagArgProvided(config.Token, "-token")
	AssertFlagArgProvided(config.User, "-user")

	if config.Interval != 0 && config.Interval < time.Second {
		fmt.Println("error: argument for flag -interval cannot be shorter than 1s")
		os.Exit(2)
	}

	config.URL = fmt.Sprintf(config.URL, config.Domain, config.User)

	if config.Debug {
		fmt.Println("debug:", config.Debug)
		fmt.Println("domain:", config.Domain)
		fmt.Println("header:", config.ShowHeader)
		fmt.Println("interval:", config.Interval)
		fmt.Println("token:", config.Token)
		fmt.Println("url:", config.URL)
		fmt.Println("user:", config.User)
	}
}

func RunDashboardLoop() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		DisableTermCursor()
		<-signals
		EnableTermCursor()
		os.Exit(0)
	}()

	FetchAndPrint()
	for range time.Tick(config.Interval) {
		FetchAndPrint()
	}
}

func FetchAndPrint() {
	jobs, err := FetchJobs(config.URL)

	if config.Interval != 0 {
		ClearTermScreen()
	}

	if err != nil {
		fmt.Println("error:", err)
	} else {
		PrintJobs(jobs)
	}
}

func FetchJobs(url string) ([]Job, error) {
	var favorites []FavoriteWrapper
	var jobs []Job

	req, reqErr := http.NewRequest(http.MethodGet, url, nil)
	if reqErr != nil {
		return nil, reqErr
	}
	req.SetBasicAuth(config.User, config.Token)
	req.Header.Set("User-Agent", config.UserAgent)

	resp, httpErr := config.Client.Do(req)
	if httpErr != nil {
		return nil, httpErr
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("API returned status code %v", resp.StatusCode))
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	if len(body) == 0 {
		return nil, errors.New("API returned empty response body")
	}

	jsonErr := json.Unmarshal(body, &favorites)
	if jsonErr != nil {
		return nil, jsonErr
	}

	for _, fav := range favorites {
		item := fav.FavoriteItem
		job := Job{
			DisplayName: item.DisplayName,
			ID:          item.LatestRun.ID,
			StartTime:   item.LatestRun.StartTime,
			EndTime:     item.LatestRun.EndTime,
			Result:      item.LatestRun.Result,
			State:       item.LatestRun.State,
		}
		jobs = append(jobs, job)
	}

	SortJobs(jobs)

	return jobs, nil
}

func SortJobs(jobs []Job) {
	sort.Slice(jobs, func(i, j int) bool {
		jobA := jobs[i]
		jobB := jobs[j]
		if jobA.State == "RUNNING" && jobB.State != "RUNNING" {
			return true
		}
		if jobA.State != "RUNNING" && jobB.State == "RUNNING" {
			return false
		}
		var a, b time.Time
		var aErr, bErr error
		if jobA.State == "RUNNING" {
			a, aErr = ParseDate(jobA.StartTime)
		} else {
			a, aErr = ParseDate(jobA.EndTime)
		}
		if jobB.State == "RUNNING" {
			b, bErr = ParseDate(jobB.StartTime)
		} else {
			b, bErr = ParseDate(jobB.EndTime)
		}
		if aErr != nil || bErr != nil {
			return false
		}
		return a.After(b)
	})
}

func PrintJobs(jobs []Job) {
	table := tabby.New()
	now := time.Now()

	if config.ShowHeader {
		currentTime := time.Now().Format(time.RFC1123)
		table.AddHeader(currentTime)
	}

	for _, job := range jobs {
		var durationString string
		var resultString string

		if config.Debug {
			job.PrintDebugInfo()
		}

		if job.State == "FINISHED" {
			durationString = FormattedDurationString(job.EndTime, now) + " ago"
			resultString = ColorizedJobState(job.Result)
		} else {
			durationString = FormattedDurationString(job.StartTime, now)
			resultString = ColorizedJobState(job.State)
		}

		table.AddLine(job.DisplayName, job.ID, resultString, durationString)
	}

	table.Print()
}

func ParseDate(dateStr string) (time.Time, error) {
	t, err := time.Parse(config.DateFormat, dateStr)
	if err != nil {
		return t, err
	}
	return t.Local(), nil
}

func AbsoluteDuration(a, b time.Time) time.Duration {
	if a.Before(b) {
		return b.Sub(a)
	}
	return a.Sub(b)
}

func FormattedDurationString(timeString string, referenceTime time.Time) string {
	timeValue, err := ParseDate(timeString)
	if err != nil {
		return timeString
	}

	duration := AbsoluteDuration(timeValue, referenceTime)
	durationString := durafmt.Parse(duration.Truncate(time.Second)).LimitFirstN(2).String()
	return durationString
}

func ColorizedJobState(state string) string {
	switch state {
	case "SUCCESS":
		return color.GreenString(state)
	case "FAILURE":
		return color.RedString(state)
	case "ABORTED":
		return color.YellowString(state)
	case "RUNNING":
		return color.CyanString(state)
	default:
		return state
	}
}

func AssertFlagArgProvided(arg any, flag string) {
	if reflect.ValueOf(arg).IsZero() {
		fmt.Println("error: missing required argument for flag", flag)
		os.Exit(2)
	}
}

func ReadConfigStrFromEnv(key string, destination *string) {
	value := os.Getenv(key)
	if !reflect.ValueOf(value).IsZero() {
		*destination = value
	}
}

func ClearTermScreen() {
	if IsUnixLikeOS() {
		clear := exec.Command("clear")
		clear.Stdout = os.Stdout
		clear.Run()
	} else if IsWindowsOS() {
		clear := exec.Command("cmd", "/c", "cls")
		clear.Stdout = os.Stdout
		clear.Run()
	} else {
		fmt.Printf("\033[2J\033[0;0H")
	}
}

func IsUnixLikeOS() bool {
	systems := []string{"android", "darwin", "dragonfly", "freebsd", "linux", "netbsd", "openbsd"}
	for _, system := range systems {
		if runtime.GOOS == system {
			return true
		}
	}
	return false
}

func IsWindowsOS() bool {
	return runtime.GOOS == "windows"
}

func EnableTermCursor() {
	fmt.Print("\033[?25h")
}

func DisableTermCursor() {
	fmt.Print("\033[?25l")
}
