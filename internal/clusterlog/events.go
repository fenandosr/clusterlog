package clusterlog

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	reviewEventsDirName = "review-events"
	taskEventsDirName   = "task-events"
)

func listJSONFiles(dir string) ([]string, error) {
	files := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return files, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func reviewEventFiles(root string) ([]string, error) {
	return listJSONFiles(filepath.Join(root, "data", reviewEventsDirName))
}

func taskEventFiles(root string) ([]string, error) {
	return listJSONFiles(filepath.Join(root, "data", taskEventsDirName))
}

func eventFilePath(root, dirName, id string) string {
	return filepath.Join(root, "data", dirName, strings.ToLower(id)+".json")
}

func writeEventFile(path string, value any) error {
	if _, err := os.Stat(path); err == nil {
		return NewError(ExitConflict, "event_exists", "ya existe un evento con el mismo identificador", filepath.ToSlash(path))
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeJSONAtomic(path, value)
}

func writeReviewEvent(root string, event ReviewEvent) (string, error) {
	path := eventFilePath(root, reviewEventsDirName, event.ID)
	return path, writeEventFile(path, event)
}

func writeTaskEvents(root string, events []TaskEvent) ([]string, error) {
	paths := make([]string, 0, len(events))
	for _, event := range events {
		path := eventFilePath(root, taskEventsDirName, event.ID)
		if err := writeEventFile(path, event); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func parseEventTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func sortReviewEvents(events []ReviewEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		ti := parseEventTime(events[i].ReviewedAt)
		tj := parseEventTime(events[j].ReviewedAt)
		if ti.Equal(tj) {
			return events[i].ID < events[j].ID
		}
		return ti.Before(tj)
	})
}

func sortTaskEvents(events []TaskEvent) {
	typeOrder := map[string]int{"start": 0, "stop": 1, "adjust": 2, "status": 3}
	sort.SliceStable(events, func(i, j int) bool {
		ti := parseEventTime(events[i].At)
		tj := parseEventTime(events[j].At)
		if ti.Equal(tj) {
			oi, iOK := typeOrder[events[i].Type]
			oj, jOK := typeOrder[events[j].Type]
			if !iOK {
				oi = 9
			}
			if !jOK {
				oj = 9
			}
			if oi == oj {
				return events[i].ID < events[j].ID
			}
			return oi < oj
		}
		return ti.Before(tj)
	})
}
