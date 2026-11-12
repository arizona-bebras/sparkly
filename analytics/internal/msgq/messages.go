package msgq

type GroupMembershipMsg struct {
	ID       string   `json:"id"`
	DeviceID string   `json:"device_id"`
	Version  string   `json:"version"`
	GroupIDs []string `json:"group_ids"`
}