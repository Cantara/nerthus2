package message

type Action struct {
	Action          string            `json:"action"`
	AnsiblePlaybook string            `json:"ansible_playbook"`
	ExtraVars       map[string]string `json:"extra_vars"`
	Response        *Response         `json:"response,omitempty"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   error  `json:"error"`
}

const (
	RoleUpdate = "RoleUpdate"
	Playbook   = "Playbook"
)
