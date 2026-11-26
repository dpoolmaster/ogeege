package profile

type profileRequest struct {
	FirstName string         `json:"first_name" form:"first_name"`
	LastName  string         `json:"last_name" form:"last_name"`
	BirthDate string         `json:"birth_date" form:"birth_date"`
	Grade     *int           `json:"grade" form:"grade"`
	Timezone  string         `json:"timezone" form:"timezone"`
	Contacts  map[string]any `json:"contacts"`
}

type goalRequest struct {
	ExamID        string `json:"exam_id" form:"exam_id"`
	ExamDate      string `json:"exam_date" form:"exam_date"`
	TargetScore   int    `json:"target_score" form:"target_score"`
	WeeklyMinutes int    `json:"weekly_minutes" form:"weekly_minutes"`
}

type settingsRequest struct {
	EmailNotifications *bool `json:"email_notifications" form:"email_notifications"`
	WeeklyReport       *bool `json:"weekly_report" form:"weekly_report"`
}
