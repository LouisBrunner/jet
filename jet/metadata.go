package jet

import (
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
)

const (
	jetMetadataEntry = "__jet"
)

type slackMetadataJet struct {
	Flow  string              `json:"f" mapstructure:"f"`
	Hooks []slackMetadataHook `json:"h,omitempty" mapstructure:"h"`

	Original slack.SlackMetadata `json:"-"`
}

func deserializeMetadata(meta *slack.SlackMetadata) (*slackMetadataJet, error) {
	jetEntryRaw, found := meta.EventPayload[jetMetadataEntry]
	if !found {
		return nil, fmt.Errorf("missing jet metadata")
	}
	jetEntryJSON, err := json.Marshal(jetEntryRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jet metadata: %w", err)
	}
	var jetEntry slackMetadataJet
	err = json.Unmarshal(jetEntryJSON, &jetEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal jet metadata: %w", err)
	}
	jetEntry.Original = *meta
	return &jetEntry, nil
}

func serializeMetadata(prev *slackMetadataJet, name string, hooks []slackMetadataHook) slack.SlackMetadata {
	meta := slackMetadataJet{
		Flow:  name,
		Hooks: hooks,
	}
	if prev != nil {
		final := prev.Original
		final.EventPayload[jetMetadataEntry] = meta
		return final
	}
	return slack.SlackMetadata{
		EventType: "jet",
		EventPayload: map[string]interface{}{
			jetMetadataEntry: meta,
		},
	}
}
