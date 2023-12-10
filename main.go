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
	Debug         bool
	ShowHeader    bool
	Interval      time.Duration
	DateFormat    string
	DateOutFormat string
	Domain        string
	User          string
	Token         string
	URL           string
	UserAgent     string
	Client        http.Client
}

var config Config = Config{
	Debug:         false,
	ShowHeader:    false,
	Interval:      0,
	DateFormat:    "2006-01-02T15:04:05.000-0700",
	DateOutFormat: time.RFC1123,
	Domain:        "",
	User:          "dashboard",
	Token:         "",
	URL:           "https://%s/blue/rest/users/%s/favorites/",
	UserAgent:     "bodash/0.5",
	Client:        http.Client{Timeout: time.Second * 5},
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
	var parse_err error
	config, parse_err = ParseArgs(os.Args[1:])
	if parse_err != nil {
		fmt.Fprintln(os.Stderr, "error:", parse_err)
		os.Exit(2)
	}

	if config.Interval == 0 {
		FetchAndPrint()
	} else {
		RunDashboardLoop()
	}
}

func ParseArgs(args []string) (Config, error) {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	cfg := config

	ReadConfigStrFromEnv("BODASH_URL", &cfg.URL)
	ReadConfigStrFromEnv("BODASH_DOMAIN", &cfg.Domain)
	ReadConfigStrFromEnv("BODASH_TOKEN", &cfg.Token)
	ReadConfigStrFromEnv("BODASH_USER", &cfg.User)

	f.BoolVar(&cfg.Debug, "debug", cfg.Debug, "show debug info")
	f.StringVar(&cfg.URL, "url", cfg.URL, "Blue Ocean favorites url (e.g https://ci.example/blue/rest/users/mydashboarduser/favorites)")
	f.StringVar(&cfg.Domain, "domain", cfg.Domain, "Blue Ocean domain (e.g. ci.example)")
	f.BoolVar(&cfg.ShowHeader, "header", cfg.ShowHeader, "show dashboard header row")
	f.DurationVar(&cfg.Interval, "interval", cfg.Interval, "dashboard refresh interval")
	f.StringVar(&cfg.Token, "token", cfg.Token, "Blue Ocean user API token")
	f.StringVar(&cfg.User, "user", cfg.User, "Blue Ocean user")
	f.StringVar(&cfg.DateOutFormat, "dateoutformat", cfg.DateOutFormat, "header output date format")

	parse_err := f.Parse(args)
	if parse_err != nil {
		return Config{}, parse_err
	}

	var required_err error
	if required_err = AssertFlagArgProvided(cfg.User, "-user"); required_err != nil {
		return Config{}, required_err
	}
	if required_err = AssertFlagArgProvided(cfg.Token, "-token"); required_err != nil {
		return Config{}, required_err
	}

	if !IsFlagArgProvided(f, "url") && !IsEnvVarProvided("BODASH_URL") {
		if required_err = AssertFlagArgProvided(cfg.Domain, "-domain"); required_err != nil {
			return Config{}, required_err
		}
		cfg.URL = fmt.Sprintf(cfg.URL, cfg.Domain, cfg.User)
	}

	if cfg.Interval != 0 && cfg.Interval < time.Second {
		return Config{}, errors.New("argument for flag -interval cannot be shorter than 1s")
	}

	if cfg.Debug {
		fmt.Println("debug:", cfg.Debug)
		fmt.Println("domain:", cfg.Domain)
		fmt.Println("header:", cfg.ShowHeader)
		fmt.Println("interval:", cfg.Interval)
		fmt.Println("token:", cfg.Token)
		fmt.Println("url:", cfg.URL)
		fmt.Println("user:", cfg.User)
		fmt.Println("dateoutformat:", cfg.DateOutFormat)
	}

	return cfg, nil
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
		fmt.Fprintln(os.Stderr, "error:", err)
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

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("API returned status code %v", resp.StatusCode))
	}

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
		currentTime := now.Format(config.DateOutFormat)
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

func AssertFlagArgProvided(arg any, flag string) error {
	if reflect.ValueOf(arg).IsZero() {
		return errors.New(fmt.Sprintf("missing required argument for flag %s", flag))
	}
	return nil
}

func IsFlagArgProvided(f *flag.FlagSet, flagName string) bool {
	isProvided := false
	f.Visit(func(f *flag.Flag) {
		if flagName == f.Name {
			isProvided = true
			return
		}
	})
	return isProvided
}

func IsEnvVarProvided(key string) bool {
	_, isProvided := os.LookupEnv(key)
	return isProvided
}

func ReadConfigStrFromEnv(key string, destination *string) {
	if IsEnvVarProvided(key) {
		value := os.Getenv(key)
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
