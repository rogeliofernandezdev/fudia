package domain

type BillRequestInput struct {
	ConversationID string `json:"conversationId"`
	CustomerPhone  string `json:"customerPhone"`
	// RequestID identifica idempotentemente una solicitud de cuenta: repetirlo
	// devuelve la cuenta vigente sin registrar otra solicitud.
	RequestID string `json:"requestId"`
}
type SettingsView struct {
	Active            bool   `json:"active"`
	OrganizationName  string `json:"organizationName"`
	LocationName      string `json:"locationName"`
	ManagedByPlatform bool   `json:"managedByPlatform"`
}
type HandoffView struct {
	ID             string  `json:"id"`
	TableID        string  `json:"tableId"`
	TableName      string  `json:"tableName"`
	ConversationID string  `json:"conversationId"`
	CustomerPhone  string  `json:"customerPhone"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"`
	RequestedAt    string  `json:"requestedAt"`
	ResolvedAt     *string `json:"resolvedAt"`
	ResolvedByName *string `json:"resolvedByName"`
}
type HandoffInput struct {
	ConversationID string `json:"conversationId"`
	CustomerPhone  string `json:"customerPhone"`
	Reason         string `json:"reason"`
}
