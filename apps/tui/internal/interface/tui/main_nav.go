package tui

var mainNavViews = []viewID{
	viewLibrary,
	viewCommunities,
	viewSettings,
	viewAccount,
}

func (m model) isMainNavView(view viewID) bool {
	for _, candidate := range mainNavViews {
		if view == candidate {
			return true
		}
	}
	return false
}

func (m *model) stepMainView(delta int) {
	if delta == 0 || !m.isMainNavView(m.currentView) {
		return
	}

	currentIdx := 0
	for i, view := range mainNavViews {
		if m.currentView == view {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + delta) % len(mainNavViews)
	if nextIdx < 0 {
		nextIdx += len(mainNavViews)
	}

	nextView := mainNavViews[nextIdx]
	if nextView == viewSettings {
		m.openSettings(m.currentView)
		return
	}

	m.currentView = nextView
}
