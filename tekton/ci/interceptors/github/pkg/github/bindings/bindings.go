package bindings

type Git struct {
	URL      string `json:"url,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type GitHub struct {
	Owner        string `json:"owner,omitempty"`
	Repo         string `json:"repo,omitempty"`
	Installation int64  `json:"installation,omitempty"`
}
