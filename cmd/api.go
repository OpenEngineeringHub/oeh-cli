package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// fetchTaskSpec fetches the task specification from the OEH platform.
func fetchTaskSpec(taskID string) (*TaskSpec, error) {
	platformURL := getPlatformURL()
	url := fmt.Sprintf("%s/api/tasks/%s/spec", platformURL, taskID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("platform unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("task %q not found on platform", taskID)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("platform returned HTTP %d", resp.StatusCode)
	}

	var spec TaskSpec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("invalid spec response: %w", err)
	}
	return &spec, nil
}
