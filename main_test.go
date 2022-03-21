package main

import (
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
