package ui

import (
	"github.com/NiladriHazra/filerepo/internal/config"
)

func (m *model) recordRecentRepo(rawURL string) {
	if stringsTrimmed(rawURL) == "" {
		return
	}

	m.configState.AddRecentRepo(rawURL)
	if err := config.Save(m.configState); err != nil {
		m.showToast("Failed to save recent repos.", toastWarning)
	}
}

func (m *model) toggleFavorite(rawURL string) bool {
	if m.configState.IsFavorite(rawURL) {
		m.configState.RemoveFavorite(rawURL)
		if err := config.Save(m.configState); err != nil {
			m.showToast("Failed to update favorites.", toastWarning)
			return false
		}
		return false
	}

	m.configState.AddFavorite(rawURL)
	if err := config.Save(m.configState); err != nil {
		m.showToast("Failed to update favorites.", toastWarning)
		return false
	}
	return true
}
