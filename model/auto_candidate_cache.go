package model

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/55gY/new-api-lite/common"
)

const autoRecentSuccessTTL = 5 * time.Minute

type autoRecentSuccess struct {
	ChannelID int
	Model     string
	ExpiresAt time.Time
}

var autoRecentSuccessMu sync.Mutex
var autoRecentSuccessByGroup = make(map[string]autoRecentSuccess)

// RecordAutoModelSuccess records a successful concrete model for a short
// period. It is process-local by design: the preference is advisory and must
// never become a source of truth for availability.
func RecordAutoModelSuccess(group string, channelID int, modelName string) {
	modelName = strings.TrimSpace(modelName)
	if channelID <= 0 || !isConcreteModelName(modelName) {
		return
	}
	autoRecentSuccessMu.Lock()
	autoRecentSuccessByGroup[strings.TrimSpace(group)] = autoRecentSuccess{
		ChannelID: channelID,
		Model:     modelName,
		ExpiresAt: time.Now().Add(autoRecentSuccessTTL),
	}
	autoRecentSuccessMu.Unlock()
}

func getRecentAutoSuccess(group string) (autoRecentSuccess, bool) {
	now := time.Now()
	autoRecentSuccessMu.Lock()
	defer autoRecentSuccessMu.Unlock()
	key := strings.TrimSpace(group)
	recent, ok := autoRecentSuccessByGroup[key]
	if !ok || !now.Before(recent.ExpiresAt) {
		delete(autoRecentSuccessByGroup, key)
		return autoRecentSuccess{}, false
	}
	return recent, true
}

// RefreshAutoCandidatesForChannel incrementally refreshes one channel's auto
// candidates. Database state remains the source of truth; the in-memory map is
// only a derived cache.
// RefreshAllAutoCandidates rebuilds the derived in-memory candidate index from
// current database state. It is intentionally reserved for maintenance points
// such as completion of testing all channels.
func RefreshAllAutoCandidates() {
	if !common.MemoryCacheEnabled {
		return
	}
	var channels []Channel
	if err := DB.Find(&channels).Error; err != nil {
		return
	}
	fresh := make(map[string][]autoCandidate)
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		abilities, err := GetChannelAbilities(channel.Id)
		if err != nil {
			continue
		}
		for _, ability := range abilities {
			if !ability.Enabled || ability.Status != common.ChannelStatusEnabled || ability.TestStatus == AbilityTestStatusUnavailable || !isConcreteChannelModel(&channel, ability.Model) {
				continue
			}
			candidate := autoCandidate{Model: strings.TrimSpace(ability.Model), ChannelId: channel.Id, Priority: int64(abilityPriority(ability)), Weight: ability.Weight}
			fresh[""] = append(fresh[""], candidate)
			fresh[ability.Group] = append(fresh[ability.Group], candidate)
		}
	}
	for group, candidates := range fresh {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Priority != candidates[j].Priority {
				return candidates[i].Priority > candidates[j].Priority
			}
			return candidates[i].ChannelId < candidates[j].ChannelId
		})
		fresh[group] = candidates
	}
	channelSyncLock.Lock()
	group2autoCandidates = fresh
	channelSyncLock.Unlock()
}

func RefreshAutoCandidatesForChannel(channelID int) {
	if !common.MemoryCacheEnabled || channelID <= 0 {
		return
	}
	var channel Channel
	if err := DB.First(&channel, "id = ?", channelID).Error; err != nil {
		return
	}
	abilities, err := GetChannelAbilities(channelID)
	if err != nil {
		return
	}

	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if group2autoCandidates == nil {
		group2autoCandidates = make(map[string][]autoCandidate)
	}
	for group, candidates := range group2autoCandidates {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if candidate.ChannelId != channelID {
				filtered = append(filtered, candidate)
			}
		}
		group2autoCandidates[group] = filtered
	}
	if channel.Status != common.ChannelStatusEnabled {
		return
	}

	for _, ability := range abilities {
		if !ability.Enabled || ability.Status != common.ChannelStatusEnabled || ability.TestStatus == AbilityTestStatusUnavailable || !isConcreteChannelModel(&channel, ability.Model) {
			continue
		}
		candidate := autoCandidate{
			Model:     strings.TrimSpace(ability.Model),
			ChannelId: channelID,
			Priority:  int64(abilityPriority(ability)),
			Weight:    ability.Weight,
		}
		group2autoCandidates[""] = append(group2autoCandidates[""], candidate)
		group2autoCandidates[ability.Group] = append(group2autoCandidates[ability.Group], candidate)
	}
	for group, candidates := range group2autoCandidates {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Priority != candidates[j].Priority {
				return candidates[i].Priority > candidates[j].Priority
			}
			return candidates[i].ChannelId < candidates[j].ChannelId
		})
		group2autoCandidates[group] = candidates
	}
}
