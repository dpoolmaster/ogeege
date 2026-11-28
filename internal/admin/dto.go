package admin

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/content"
)

type examRequest struct {
	Code    string `json:"code" form:"code"`
	Title   string `json:"title" form:"title"`
	Subject string `json:"subject" form:"subject"`
}

type examYearRequest struct {
	ExamID          string `json:"exam_id" form:"exam_id"`
	Year            int    `json:"year" form:"year"`
	DurationMinutes int    `json:"duration_minutes" form:"duration_minutes"`
}

type kimNumberRequest struct {
	ExamYearID      string `json:"exam_year_id" form:"exam_year_id"`
	Number          int    `json:"number" form:"number"`
	Part            int    `json:"part" form:"part"`
	AnswerType      string `json:"answer_type" form:"answer_type"`
	MaxPrimaryScore int    `json:"max_primary_score" form:"max_primary_score"`
	Title           string `json:"title" form:"title"`
}

type kimTopicRequest struct {
	TopicID string  `json:"topic_id"`
	Weight  float64 `json:"weight"`
}

type kimTopicsRequest struct {
	Topics []kimTopicRequest `json:"topics"`
}

type topicRequest struct {
	ExamID     string  `json:"exam_id" form:"exam_id"`
	ParentID   *string `json:"parent_id" form:"parent_id"`
	Code       string  `json:"code" form:"code"`
	Title      string  `json:"title" form:"title"`
	OrderIndex int     `json:"order_index" form:"order_index"`
}

type theoryRequest struct {
	TopicID   string `json:"topic_id" form:"topic_id"`
	ContentMD string `json:"content_md" form:"content_md"`
}

type taskRequest struct {
	ExamYearID    string              `json:"exam_year_id" form:"exam_year_id"`
	KimNumberID   string              `json:"kim_number_id" form:"kim_number_id"`
	TopicID       *string             `json:"topic_id" form:"topic_id"`
	Difficulty    string              `json:"difficulty" form:"difficulty"`
	StatementMD   string              `json:"statement_md" form:"statement_md"`
	AnswerSpec    json.RawMessage     `json:"answer_spec"`
	AnswerDisplay string              `json:"answer_display" form:"answer_display"`
	Source        string              `json:"source" form:"source"`
	SourceRef     string              `json:"source_ref" form:"source_ref"`
	Options       []taskOptionRequest `json:"options"`
}

type taskOptionRequest struct {
	ID        string `json:"id"`
	Position  int    `json:"position"`
	ContentMD string `json:"content_md"`
	IsCorrect bool   `json:"is_correct"`
}

type importItemRequest struct {
	ExamYearID    string              `json:"exam_year_id"`
	KimNumberID   string              `json:"kim_number_id"`
	TopicID       *string             `json:"topic_id"`
	Difficulty    string              `json:"difficulty"`
	StatementMD   string              `json:"statement_md"`
	AnswerSpec    json.RawMessage     `json:"answer_spec"`
	AnswerDisplay string              `json:"answer_display"`
	Source        string              `json:"source"`
	SourceRef     string              `json:"source_ref"`
	Options       []taskOptionRequest `json:"options"`
}

type importRequest struct {
	Items []importItemRequest `json:"items"`
}

type variantTaskRequest struct {
	TaskID string `json:"task_id"`
	Points int    `json:"points"`
}

type variantRequest struct {
	ExamYearID      string               `json:"exam_year_id"`
	Title           string               `json:"title"`
	Kind            string               `json:"kind"`
	DurationMinutes int                  `json:"duration_minutes"`
	Tasks           []variantTaskRequest `json:"tasks"`
}

type generationRuleRequest struct {
	KimNumberID string  `json:"kim_number_id"`
	Count       int     `json:"count"`
	Difficulty  *string `json:"difficulty"`
	TopicID     *string `json:"topic_id"`
}

type generateVariantRequest struct {
	ExamYearID      string                  `json:"exam_year_id"`
	Title           string                  `json:"title"`
	Kind            string                  `json:"kind"`
	DurationMinutes int                     `json:"duration_minutes"`
	Rules           []generationRuleRequest `json:"rules"`
}

type rubricRequest struct {
	KimNumberID string                    `json:"kim_number_id"`
	ExamYearID  string                    `json:"exam_year_id"`
	Criteria    []content.RubricCriterion `json:"criteria"`
}

type rubricUpdateRequest struct {
	Criteria []content.RubricCriterion `json:"criteria"`
}

type scoreScaleEntryRequest struct {
	PrimaryScore int  `json:"primary_score"`
	TestScore    int  `json:"test_score"`
	Grade        *int `json:"grade"`
	MinThreshold *int `json:"min_threshold"`
}

type scoreScaleRequest struct {
	Entries []scoreScaleEntryRequest `json:"entries"`
}

type userPatchRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type examResponse struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Title     string    `json:"title"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type examYearResponse struct {
	ID              uuid.UUID `json:"id"`
	ExamID          uuid.UUID `json:"exam_id"`
	Year            int       `json:"year"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type kimNumberResponse struct {
	ID              uuid.UUID `json:"id"`
	ExamYearID      uuid.UUID `json:"exam_year_id"`
	Number          int       `json:"number"`
	Part            int       `json:"part"`
	AnswerType      string    `json:"answer_type"`
	MaxPrimaryScore int       `json:"max_primary_score"`
	Title           string    `json:"title"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type kimTopicResponse struct {
	TopicID uuid.UUID `json:"topic_id"`
	Weight  float64   `json:"weight"`
}

type topicResponse struct {
	ID         uuid.UUID  `json:"id"`
	ExamID     uuid.UUID  `json:"exam_id"`
	ParentID   *uuid.UUID `json:"parent_id"`
	Code       string     `json:"code"`
	Title      string     `json:"title"`
	OrderIndex int        `json:"order_index"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type theoryResponse struct {
	ID        uuid.UUID  `json:"id"`
	TopicID   uuid.UUID  `json:"topic_id"`
	ContentMD string     `json:"content_md"`
	Version   int        `json:"version"`
	Status    string     `json:"status"`
	UpdatedBy *uuid.UUID `json:"updated_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type taskResponse struct {
	ID            uuid.UUID            `json:"id"`
	ExamYearID    uuid.UUID            `json:"exam_year_id"`
	KimNumberID   uuid.UUID            `json:"kim_number_id"`
	TopicID       *uuid.UUID           `json:"topic_id"`
	Difficulty    string               `json:"difficulty"`
	StatementMD   string               `json:"statement_md"`
	AnswerSpec    json.RawMessage      `json:"answer_spec"`
	AnswerDisplay string               `json:"answer_display"`
	Source        string               `json:"source"`
	SourceRef     string               `json:"source_ref"`
	Status        string               `json:"status"`
	IsPublic      bool                 `json:"is_public"`
	Version       int                  `json:"version"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Options       []taskOptionResponse `json:"options"`
}

type taskOptionResponse struct {
	ID        uuid.UUID `json:"id"`
	Position  int       `json:"position"`
	ContentMD string    `json:"content_md"`
	IsCorrect bool      `json:"is_correct"`
}

type taskVersionResponse struct {
	ID            uuid.UUID       `json:"id"`
	TaskID        uuid.UUID       `json:"task_id"`
	Version       int             `json:"version"`
	StatementMD   string          `json:"statement_md"`
	AnswerSpec    json.RawMessage `json:"answer_spec"`
	AnswerDisplay string          `json:"answer_display"`
	Difficulty    string          `json:"difficulty"`
	KimNumberID   uuid.UUID       `json:"kim_number_id"`
	TopicID       *uuid.UUID      `json:"topic_id"`
	Source        string          `json:"source"`
	SourceRef     string          `json:"source_ref"`
	CreatedAt     time.Time       `json:"created_at"`
}

type taskAttachmentResponse struct {
	ID         uuid.UUID `json:"id"`
	TaskID     uuid.UUID `json:"task_id"`
	Kind       string    `json:"kind"`
	StorageKey string    `json:"storage_key"`
	URL        string    `json:"url"`
	Alt        string    `json:"alt"`
	CreatedAt  time.Time `json:"created_at"`
}

type variantTaskResponse struct {
	Position int       `json:"position"`
	TaskID   uuid.UUID `json:"task_id"`
	Points   int       `json:"points"`
}

type variantResponse struct {
	ID              uuid.UUID             `json:"id"`
	ExamYearID      uuid.UUID             `json:"exam_year_id"`
	Title           string                `json:"title"`
	Kind            string                `json:"kind"`
	DurationMinutes int                   `json:"duration_minutes"`
	Status          string                `json:"status"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Tasks           []variantTaskResponse `json:"tasks,omitempty"`
}

type rubricResponse struct {
	ID          uuid.UUID                 `json:"id"`
	KimNumberID uuid.UUID                 `json:"kim_number_id"`
	ExamYearID  uuid.UUID                 `json:"exam_year_id"`
	Version     int                       `json:"version"`
	Criteria    []content.RubricCriterion `json:"criteria"`
	Status      string                    `json:"status"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type scoreScaleResponse struct {
	PrimaryScore int  `json:"primary_score"`
	TestScore    int  `json:"test_score"`
	Grade        *int `json:"grade"`
	MinThreshold *int `json:"min_threshold"`
}

type importReportResponse struct {
	Created  []importCreatedResponse  `json:"created"`
	Skipped  []importSkippedResponse  `json:"skipped"`
	Rejected []importRejectedResponse `json:"rejected"`
}

type importCreatedResponse struct {
	Index  int       `json:"index"`
	TaskID uuid.UUID `json:"task_id"`
}

type importSkippedResponse struct {
	Index  int       `json:"index"`
	TaskID uuid.UUID `json:"task_id,omitempty"`
	Reason string    `json:"reason"`
}

type importRejectedResponse struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

type userResponse struct {
	ID        uuid.UUID   `json:"id"`
	Email     string      `json:"email"`
	Role      auth.Role   `json:"role"`
	Status    auth.Status `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type auditResponse struct {
	ID        uuid.UUID       `json:"id"`
	ActorID   uuid.UUID       `json:"actor_id"`
	ActorRole string          `json:"actor_role"`
	Action    string          `json:"action"`
	Entity    string          `json:"entity"`
	EntityID  *uuid.UUID      `json:"entity_id"`
	Before    json.RawMessage `json:"before,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
	IP        string          `json:"ip"`
	CreatedAt time.Time       `json:"created_at"`
}

func toExamResponse(exam *content.Exam) examResponse {
	return examResponse{
		ID: exam.ID, Code: exam.Code, Title: exam.Title, Subject: exam.Subject,
		Status: string(exam.Status), CreatedAt: exam.CreatedAt, UpdatedAt: exam.UpdatedAt,
	}
}

func toExamResponses(exams []content.Exam) []examResponse {
	response := make([]examResponse, 0, len(exams))
	for i := range exams {
		response = append(response, toExamResponse(&exams[i]))
	}

	return response
}

func toExamYearResponse(year *content.ExamYear) examYearResponse {
	return examYearResponse{
		ID: year.ID, ExamID: year.ExamID, Year: year.Year, DurationMinutes: year.DurationMinutes,
		Status: string(year.Status), CreatedAt: year.CreatedAt, UpdatedAt: year.UpdatedAt,
	}
}

func toExamYearResponses(years []content.ExamYear) []examYearResponse {
	response := make([]examYearResponse, 0, len(years))
	for i := range years {
		response = append(response, toExamYearResponse(&years[i]))
	}

	return response
}

func toKimNumberResponse(number *content.KimNumber) kimNumberResponse {
	return kimNumberResponse{
		ID: number.ID, ExamYearID: number.ExamYearID, Number: number.Number, Part: number.Part,
		AnswerType: string(number.AnswerType), MaxPrimaryScore: number.MaxPrimaryScore, Title: number.Title,
		CreatedAt: number.CreatedAt, UpdatedAt: number.UpdatedAt,
	}
}

func toKimNumberResponses(numbers []content.KimNumber) []kimNumberResponse {
	response := make([]kimNumberResponse, 0, len(numbers))
	for i := range numbers {
		response = append(response, toKimNumberResponse(&numbers[i]))
	}

	return response
}

func toKimTopicResponses(topics []content.KimTopic) []kimTopicResponse {
	response := make([]kimTopicResponse, 0, len(topics))
	for _, topic := range topics {
		response = append(response, kimTopicResponse{TopicID: topic.TopicID, Weight: topic.Weight})
	}

	return response
}

func toTopicResponse(topic *content.Topic) topicResponse {
	return topicResponse{
		ID: topic.ID, ExamID: topic.ExamID, ParentID: topic.ParentID, Code: topic.Code,
		Title: topic.Title, OrderIndex: topic.OrderIndex, CreatedAt: topic.CreatedAt, UpdatedAt: topic.UpdatedAt,
	}
}

func toTopicResponses(topics []content.Topic) []topicResponse {
	response := make([]topicResponse, 0, len(topics))
	for i := range topics {
		response = append(response, toTopicResponse(&topics[i]))
	}

	return response
}

func toTheoryResponse(theory *content.Theory) theoryResponse {
	return theoryResponse{
		ID: theory.ID, TopicID: theory.TopicID, ContentMD: theory.ContentMD, Version: theory.Version,
		Status: string(theory.Status), UpdatedBy: theory.UpdatedBy, CreatedAt: theory.CreatedAt, UpdatedAt: theory.UpdatedAt,
	}
}

func toTheoryResponses(versions []content.Theory) []theoryResponse {
	response := make([]theoryResponse, 0, len(versions))
	for i := range versions {
		response = append(response, toTheoryResponse(&versions[i]))
	}

	return response
}

func toTaskResponse(task *content.Task) taskResponse {
	response := taskResponse{
		ID: task.ID, ExamYearID: task.ExamYearID, KimNumberID: task.KimNumberID, TopicID: task.TopicID,
		Difficulty: string(task.Difficulty), StatementMD: task.StatementMD, AnswerSpec: task.AnswerSpec,
		AnswerDisplay: task.AnswerDisplay, Source: task.Source, SourceRef: task.SourceRef,
		Status: string(task.Status), IsPublic: task.IsPublic, Version: task.Version,
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
		Options: make([]taskOptionResponse, 0, len(task.Options)),
	}
	for _, option := range task.Options {
		response.Options = append(response.Options, taskOptionResponse{
			ID: option.ID, Position: option.Position, ContentMD: option.ContentMD, IsCorrect: option.IsCorrect,
		})
	}

	return response
}

func toTaskResponses(tasks []content.Task) []taskResponse {
	response := make([]taskResponse, 0, len(tasks))
	for i := range tasks {
		response = append(response, toTaskResponse(&tasks[i]))
	}

	return response
}

func toTaskVersionResponses(versions []content.TaskVersion) []taskVersionResponse {
	response := make([]taskVersionResponse, 0, len(versions))
	for _, version := range versions {
		response = append(response, taskVersionResponse{
			ID: version.ID, TaskID: version.TaskID, Version: version.Version, StatementMD: version.StatementMD,
			AnswerSpec: version.AnswerSpec, AnswerDisplay: version.AnswerDisplay, Difficulty: string(version.Difficulty),
			KimNumberID: version.KimNumberID, TopicID: version.TopicID, Source: version.Source,
			SourceRef: version.SourceRef, CreatedAt: version.CreatedAt,
		})
	}

	return response
}

func toTaskAttachmentResponse(attachment *content.TaskAttachment) taskAttachmentResponse {
	return taskAttachmentResponse{
		ID: attachment.ID, TaskID: attachment.TaskID, Kind: attachment.Kind, StorageKey: attachment.StorageKey,
		URL: attachment.URL, Alt: attachment.Alt, CreatedAt: attachment.CreatedAt,
	}
}

func toTaskAttachmentResponses(attachments []content.TaskAttachment) []taskAttachmentResponse {
	response := make([]taskAttachmentResponse, 0, len(attachments))
	for i := range attachments {
		response = append(response, toTaskAttachmentResponse(&attachments[i]))
	}

	return response
}

func toVariantResponse(variant *content.Variant) variantResponse {
	response := variantResponse{
		ID: variant.ID, ExamYearID: variant.ExamYearID, Title: variant.Title, Kind: string(variant.Kind),
		DurationMinutes: variant.DurationMinutes, Status: string(variant.Status),
		CreatedAt: variant.CreatedAt, UpdatedAt: variant.UpdatedAt,
	}
	for _, task := range variant.Tasks {
		response.Tasks = append(response.Tasks, variantTaskResponse{
			Position: task.Position, TaskID: task.TaskID, Points: task.Points,
		})
	}

	return response
}

func toVariantResponses(variants []content.Variant) []variantResponse {
	response := make([]variantResponse, 0, len(variants))
	for i := range variants {
		response = append(response, toVariantResponse(&variants[i]))
	}

	return response
}

func toRubricResponse(rubric *content.Rubric) rubricResponse {
	return rubricResponse{
		ID: rubric.ID, KimNumberID: rubric.KimNumberID, ExamYearID: rubric.ExamYearID,
		Version: rubric.Version, Criteria: rubric.Criteria, Status: string(rubric.Status),
		CreatedAt: rubric.CreatedAt, UpdatedAt: rubric.UpdatedAt,
	}
}

func toRubricResponses(rubrics []content.Rubric) []rubricResponse {
	response := make([]rubricResponse, 0, len(rubrics))
	for i := range rubrics {
		response = append(response, toRubricResponse(&rubrics[i]))
	}

	return response
}

func toScoreScaleResponses(scales []content.ScoreScale) []scoreScaleResponse {
	response := make([]scoreScaleResponse, 0, len(scales))
	for _, scale := range scales {
		response = append(response, scoreScaleResponse{
			PrimaryScore: scale.PrimaryScore, TestScore: scale.TestScore,
			Grade: scale.Grade, MinThreshold: scale.MinThreshold,
		})
	}

	return response
}

func toImportReportResponse(report *content.ImportReport) importReportResponse {
	response := importReportResponse{
		Created:  make([]importCreatedResponse, 0, len(report.Created)),
		Skipped:  make([]importSkippedResponse, 0, len(report.Skipped)),
		Rejected: make([]importRejectedResponse, 0, len(report.Rejected)),
	}
	for _, item := range report.Created {
		response.Created = append(response.Created, importCreatedResponse{Index: item.Index, TaskID: item.TaskID})
	}
	for _, item := range report.Skipped {
		response.Skipped = append(response.Skipped, importSkippedResponse{Index: item.Index, TaskID: item.TaskID, Reason: item.Reason})
	}
	for _, item := range report.Rejected {
		response.Rejected = append(response.Rejected, importRejectedResponse{Index: item.Index, Reason: item.Reason})
	}

	return response
}

func toUserResponse(summary UserSummary) userResponse {
	return userResponse(summary)
}

func toUserResponses(summaries []UserSummary) []userResponse {
	response := make([]userResponse, 0, len(summaries))
	for _, summary := range summaries {
		response = append(response, toUserResponse(summary))
	}

	return response
}

func toAuditResponses(entries []AuditEntry) []auditResponse {
	response := make([]auditResponse, 0, len(entries))
	for _, entry := range entries {
		response = append(response, auditResponse(entry))
	}

	return response
}
