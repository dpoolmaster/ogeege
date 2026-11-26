package auth

type consentDTO struct {
	Kind            string `json:"kind"`
	Granted         bool   `json:"granted"`
	DocumentVersion string `json:"document_version"`
}

type registerRequest struct {
	Email               string       `json:"email" form:"email"`
	Password            string       `json:"password" form:"password"`
	Role                string       `json:"role" form:"role"`
	Consents            []consentDTO `json:"consents"`
	PersonalData        bool         `json:"personal_data" form:"personal_data"`
	LegalRepresentative bool         `json:"legal_representative" form:"legal_representative"`
}

type loginRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type verifyEmailRequest struct {
	Token string `json:"token" form:"token"`
}

type resendVerificationRequest struct {
	Email string `json:"email" form:"email"`
}

type passwordResetRequest struct {
	Email string `json:"email" form:"email"`
}

type passwordResetApplyRequest struct {
	Token       string `json:"token" form:"token"`
	NewPassword string `json:"new_password" form:"new_password"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password" form:"current_password"`
	NewPassword     string `json:"new_password" form:"new_password"`
}

func (r registerRequest) grants() []ConsentGrant {
	if len(r.Consents) > 0 {
		grants := make([]ConsentGrant, 0, len(r.Consents))
		for _, item := range r.Consents {
			grants = append(grants, ConsentGrant(item))
		}
		return grants
	}

	return []ConsentGrant{
		{Kind: ConsentPersonalData, Granted: r.PersonalData},
		{Kind: ConsentLegalRepresentative, Granted: r.LegalRepresentative},
	}
}
