package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func MockJobs() []Job {
	return []Job{
		{
			ID:          "2",
			DisplayName: "Test Job 2",
			StartTime:   "2022-03-10T01:28:22.000-0700",
			Result:      "UNKNOWN",
			State:       "RUNNING",
		},
		{
			ID:          "5",
			DisplayName: "Test Job 5",
			StartTime:   "2022-03-10T01:26:22.000-0700",
			EndTime:     "2022-03-10T01:30:30.000-0700",
			Result:      "FAILURE",
			State:       "FINISHED",
		},
		{
			ID:          "1",
			DisplayName: "Test Job 1",
			StartTime:   "2022-03-10T01:27:50.000-0700",
			EndTime:     "2022-03-10T01:31:12.000-0700",
			Result:      "SUCCESS",
			State:       "FINISHED",
		},
		{
			ID:          "4",
			DisplayName: "Test Job 4",
			StartTime:   "2022-03-15T12:34:56.000-0700",
			Result:      "UNKNOWN",
			State:       "RUNNING",
		},
		{
			ID:          "3",
			DisplayName: "Test Job 3",
			StartTime:   "2022-03-10T01:01:01.000-0700",
			Result:      "UNKNOWN",
			State:       "RUNNING",
		},
	}
}

func MockJobsResponse() []byte {
	return []byte(`[
			{ "item": {
				"displayName": "Test 1",
				"latestRun": {
					"id": "1",
					"startTime": "2022-01-01T00:00:00.000-0000",
					"endTime": "2022-01-01T00:01:00.000-0000",
					"result": "SUCCESS",
					"state": "FINISHED"
				}
			}},
			{ "item": {
				"displayName": "Test 2",
				"latestRun": {
					"id": "2",
					"startTime": "2022-01-01T01:00:00.000-0000",
					"result": "UNKNOWN",
					"state": "RUNNING"
				}
			}},
			{ "item": {
				"displayName": "Test 3",
				"latestRun": {
					"id": "2",
					"startTime": "2022-01-01T02:00:00.000-0000",
					"endTime": "2022-01-01T02:01:00.000-0000",
					"result": "FAILURE",
					"state": "FINISHED"
				}
			}}
			]`)
}

func TestFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authString := "Basic ZGFzaGJvYXJkOg=="
		if auth := r.Header.Get("Authorization"); auth != authString {
			t.Errorf("Request header Authorization = %s, want %s\n", auth, authString)
		}

		if ua := r.Header.Get("User-Agent"); ua != config.UserAgent {
			t.Errorf("Request header User-Agent = %s, want %s\n", ua, config.UserAgent)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(MockJobsResponse())
	}))

	defer server.Close()

	jobs, err := FetchJobs(server.URL)

	if err != nil {
		t.Errorf("FetchJobs error = \"%s\", want nil\n", err)
	}

	if jobsLength := len(jobs); jobsLength != 3 {
		t.Errorf("Response jobs length = %d, want %d\n", jobsLength, 3)
	}

	firstJob := jobs[0]
	wantFirstJob := Job{
		ID:          "2",
		DisplayName: "Test 2",
		StartTime:   "2022-01-01T01:00:00.000-0000",
		Result:      "UNKNOWN",
		State:       "RUNNING",
	}
	if firstJob.ID != wantFirstJob.ID {
		t.Errorf("Response jobs[0].ID = %s, want %s\n", firstJob.ID, wantFirstJob.ID)
	}
	if firstJob.DisplayName != wantFirstJob.DisplayName {
		t.Errorf("Response jobs[0].DisplayName = %s, want %s\n", firstJob.DisplayName, wantFirstJob.DisplayName)
	}
	if firstJob.StartTime != wantFirstJob.StartTime {
		t.Errorf("Response jobs[0].StartTime = %s, want %s\n", firstJob.StartTime, wantFirstJob.StartTime)
	}
	if firstJob.Result != wantFirstJob.Result {
		t.Errorf("Response jobs[0].Result = %s, want %s\n", firstJob.Result, wantFirstJob.Result)
	}
	if firstJob.State != wantFirstJob.State {
		t.Errorf("Response jobs[0].State = %s, want %s\n", firstJob.State, wantFirstJob.State)
	}
}

func TestSortJobs(t *testing.T) {
	jobs := MockJobs()
	SortJobs(jobs)

	if wantID := "4"; jobs[0].ID != wantID {
		t.Errorf("jobs[0].Id = %s, want %s\n", jobs[0].ID, wantID)
	}

	if wantID := "2"; jobs[1].ID != wantID {
		t.Errorf("jobs[1].Id = %s, want %s\n", jobs[1].ID, wantID)
	}

	if wantID := "3"; jobs[2].ID != wantID {
		t.Errorf("jobs[2].Id = %s, want %s\n", jobs[2].ID, wantID)
	}

	if wantID := "1"; jobs[3].ID != wantID {
		t.Errorf("jobs[3].Id = %s, want %s\n", jobs[3].ID, wantID)
	}

	if wantID := "5"; jobs[4].ID != wantID {
		t.Errorf("jobs[4].Id = %s, want %s\n", jobs[4].ID, wantID)
	}
}

func TestAbsoluteDuration(t *testing.T) {
	timeA, _ := time.Parse(config.DateFormat, "2022-01-01T01:01:01.000-0700")
	timeB, _ := time.Parse(config.DateFormat, "2022-01-01T02:02:02.000-0700")
	timeC, _ := time.Parse(config.DateFormat, "2021-12-31T22:58:59.000-0700")

	var wantDur time.Duration

	wantDur = time.Hour*1 + time.Minute*1 + time.Second*1
	if dur := AbsoluteDuration(timeA, timeB); dur != wantDur {
		t.Errorf("AbsoluteDuration(timeA, timeB) = %s, want %s\n", dur, wantDur)
	}

	if dur := AbsoluteDuration(timeB, timeA); dur != wantDur {
		t.Errorf("AbsoluteDuration(timeB, timeA) = %s, want %s\n", dur, wantDur)
	}

	wantDur = time.Hour*2 + time.Minute*2 + time.Second*2
	if dur := AbsoluteDuration(timeA, timeC); dur != wantDur {
		t.Errorf("AbsoluteDuration(timeA, timeC) = %s, want %s\n", dur, wantDur)
	}
	if dur := AbsoluteDuration(timeC, timeA); dur != wantDur {
		t.Errorf("AbsoluteDuration(timeC, timeA) = %s, want %s\n", dur, wantDur)
	}
}

func TestFormattedDurationString(t *testing.T) {
	timeString := "2022-01-01T01:01:01.000-0700"
	referenceTime, _ := time.Parse(config.DateFormat, "2022-01-02T02:02:02.000-0700")

	wantDur := "1 day 1 hour"
	if durString := FormattedDurationString(timeString, referenceTime); durString != wantDur {
		t.Errorf("FormattedDurationString(timeString, referenceTime) = %s, want %s\n", durString, wantDur)
	}
}
