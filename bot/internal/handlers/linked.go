package handlers

import "github.com/zemdenalex/neuroboost-bot/internal/api"

// openTasks is what the lists show: waiting and scheduled alike.
//
// 🔴 SCHEDULED is set by the quick «⏰ Запланировать» (the web relies on it),
// and every screen here used to ask the API for TODO only — so scheduling a
// task made it vanish from the bot. Denis 21.09: it stays, marked with its time.
func openTasks(tasks []api.Task) []api.Task {
	out := make([]api.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == "TODO" || t.Status == "SCHEDULED" {
			out = append(out, t)
		}
	}
	return out
}
