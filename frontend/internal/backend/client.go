package backend

import (
	"encoding/json"
	"fmt"
	"nix-timemach/internal/models"
	"os/exec"
	// "time"
)

type Client struct {
	backendBinary string
}

func NewClient(binaryPath string) *Client {
	return &Client{
		backendBinary: binaryPath,
	}
}

func (c *Client) GetGenerations() ([]models.Generation, error) {
	cmd := exec.Command(c.backendBinary, "list-generations")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get generations: %w", err)
	}

	var generations []models.Generation
	if err := json.Unmarshal(output, &generations); err != nil {
		return nil, fmt.Errorf("failed to parse generations: %w", err)
	}

	return generations, nil
}

func (c *Client) GetDiff(fromID, toID string) (models.GenerationDiff, error) {
	cmd := exec.Command(c.backendBinary, "diff", fromID, toID)
	output, err := cmd.Output()
	if err != nil {
		return models.GenerationDiff{}, fmt.Errorf("failed to get diff: %w", err)
	}

	var diff models.GenerationDiff
	if err := json.Unmarshal(output, &diff); err != nil {
		return models.GenerationDiff{}, fmt.Errorf("failed to parse diff: %w", err)
	}

	return diff, nil
}

/*   for testing

func (c *Client) GetGenerations() ([]models.Generation, error) {
	return []models.Generation{
		{
			ID:          "1",
			Timestamp:   time.Now().Add(-24 * time.Hour),
			Description: "Yesterday's system state",
			Profiles:    []string{"/nix/var/nix/profiles/system-1-link"},
		},
		{
			ID:          "2",
			Timestamp:   time.Now(),
			Description: "Current system state",
			Profiles:    []string{"/nix/var/nix/profiles/system-2-link"},
		},
	}, nil
}

func (c *Client) GetDiff(fromID, toID string) (models.GenerationDiff, error) {
	return models.GenerationDiff{
		Added:    []string{"package-1", "package-2"},
		Removed:  []string{"old-package"},
		Modified: []string{"modified-package"},
	}, nil
}
*/
