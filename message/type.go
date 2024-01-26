package message

type Action struct {
	Action    string            `json:"action"`
	Data      []byte            `json:"data"`
	ExtraVars map[string]string `json:"extra_vars"`
	Response  *Response         `json:"response,omitempty"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   error  `json:"error"`
}

const (
	RoleUpdate     = "RoleUpdate"
	Playbook       = "Playbook"
	Config         = "Config"
	AuthorizedKeys = "AuthorizedKeys"
)
